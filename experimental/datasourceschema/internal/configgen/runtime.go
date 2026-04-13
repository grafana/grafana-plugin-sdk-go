package configgen

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/load"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/model"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/typeschema"
)

type RuntimeOptions struct {
	Dir        string
	Patterns   []string
	BuildFlags []string
}

type RuntimeRegistration struct {
	PackagePath  string
	TypeName     string
	Target       *model.TargetRef
	FunctionName string
	Position     model.Position
}

func BuildSchemaInModule(opts RuntimeOptions, registration RuntimeRegistration) (map[string]any, error) {
	target := registration.Target
	if target == nil {
		target = &model.TargetRef{
			PackagePath: registration.PackagePath,
			TypeName:    registration.TypeName,
		}
	}
	if target.PackagePath == "" || (target.TypeName == "" && target.Expr == nil) {
		return nil, fmt.Errorf("missing datasource settings type")
	}

	loadRes, err := load.Packages(load.Config{
		Dir:        opts.Dir,
		Patterns:   normalizePatterns(opts.Patterns),
		BuildFlags: opts.BuildFlags,
	})
	if err != nil {
		return nil, err
	}

	return typeschema.BuildTargetSchema(loadRes, target, typeschema.SchemaOptions{
		RequireJSONTags:                true,
		FallbackToSimpleUntaggedFields: true,
	})
}

func BuildSchemaFromFindings(opts RuntimeOptions, findings []model.Finding) (map[string]any, []model.Warning, error) {
	registrations := make([]RuntimeRegistration, 0)
	warnings := make([]model.Warning, 0)
	seen := map[string]struct{}{}

	for _, finding := range findings {
		if finding.Source != model.SourceKindDatasourceJSON || finding.Target == nil {
			continue
		}

		key := targetKey(finding.Target)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		registrations = append(registrations, RuntimeRegistration{
			PackagePath:  finding.Target.PackagePath,
			TypeName:     finding.Target.TypeName,
			Target:       finding.Target,
			FunctionName: finding.FunctionName,
			Position:     finding.Position,
		})
	}

	if len(registrations) == 0 {
		return nil, warnings, nil
	}

	loadRes, err := load.Packages(load.Config{
		Dir:        opts.Dir,
		Patterns:   normalizePatterns(opts.Patterns),
		BuildFlags: opts.BuildFlags,
	})
	if err != nil {
		return nil, nil, err
	}

	sortRegistrations(loadRes, registrations)

	if len(registrations) > 1 {
		for _, registration := range registrations[1:] {
			if !shouldWarnAboutRegistration(loadRes, registrations[0], registration) {
				continue
			}
			warnings = append(warnings, model.Warning{
				Code:    "datasource_multiple_types",
				Message: fmt.Sprintf("multiple datasource settings types discovered; using %s and ignoring %s", describeRegistration(registrations[0]), describeRegistration(registration)),
			})
		}
	}

	schema, err := buildSchema(loadRes, registrations[0])
	if err != nil {
		return nil, warnings, err
	}

	return schema, warnings, nil
}

func normalizePatterns(patterns []string) []string {
	if len(patterns) == 0 {
		return []string{"./..."}
	}
	return patterns
}

func targetKey(target *model.TargetRef) string {
	if target == nil {
		return ""
	}
	if target.Expr != nil {
		return fmt.Sprintf("%s|%s|%s:%d:%d", target.PackagePath, target.TypeString, target.Expr.File, target.Expr.Line, target.Expr.Column)
	}
	return fmt.Sprintf("%s|%s|%s", target.PackagePath, target.TypeName, target.TypeString)
}

func describeRegistration(registration RuntimeRegistration) string {
	target := registration.Target
	if target == nil {
		target = &model.TargetRef{
			PackagePath: registration.PackagePath,
			TypeName:    registration.TypeName,
		}
	}
	if target.TypeName != "" {
		return target.PackagePath + "." + target.TypeName
	}
	if target.Expr != nil {
		return fmt.Sprintf("%s:%d:%d", target.Expr.File, target.Expr.Line, target.Expr.Column)
	}
	return target.PackagePath
}

func sortRegistrations(loadRes *load.Result, registrations []RuntimeRegistration) {
	sort.SliceStable(registrations, func(i int, j int) bool {
		return registrationScore(loadRes, registrations[i]) > registrationScore(loadRes, registrations[j])
	})
}

func registrationScore(loadRes *load.Result, registration RuntimeRegistration) int {
	target := registration.Target
	if target == nil {
		target = &model.TargetRef{
			PackagePath: registration.PackagePath,
			TypeName:    registration.TypeName,
		}
	}

	score := 0
	functionName := strings.ToLower(registration.FunctionName)
	typeName := strings.ToLower(target.TypeName)
	packagePath := strings.ToLower(target.PackagePath)
	fileName := strings.ToLower(filepath.Base(registration.Position.File))

	switch {
	case functionName == "loadsettings", strings.HasSuffix(functionName, ".loadsettings"):
		score += 1000
	case strings.Contains(functionName, "loadsettings"):
		score += 900
	case strings.Contains(functionName, "settings"):
		score += 400
	}

	switch {
	case typeName == "settings":
		score += 300
	case strings.HasSuffix(typeName, "settings"):
		score += 250
	case strings.Contains(typeName, "config"):
		score += 150
	}

	switch {
	case strings.Contains(packagePath, "/plugin"):
		score += 40
	case strings.Contains(packagePath, "/models"):
		score += 30
	}

	switch fileName {
	case "settings.go":
		score += 80
	case "config.go":
		score += 40
	}

	if strings.Contains(typeName, "auth") {
		score -= 100
	}
	if strings.Contains(typeName, "lookup") {
		score -= 50
	}
	if strings.Contains(packagePath, "/kerberos") {
		score -= 40
	}

	score += schemaSpecificityBonus(loadRes, registration)

	return score
}

func shouldWarnAboutRegistration(loadRes *load.Result, primary RuntimeRegistration, alternative RuntimeRegistration) bool {
	primaryScore := registrationScore(loadRes, primary)
	alternativeScore := registrationScore(loadRes, alternative)
	if primaryScore-alternativeScore >= 300 {
		return false
	}

	if primaryScore > alternativeScore {
		schema, err := buildSchema(loadRes, alternative)
		if err == nil && isGenericHelperSchema(schema) {
			return false
		}
	}

	return true
}

func buildSchema(loadRes *load.Result, registration RuntimeRegistration) (map[string]any, error) {
	target := registration.Target
	if target == nil {
		target = &model.TargetRef{
			PackagePath: registration.PackagePath,
			TypeName:    registration.TypeName,
		}
	}
	if target.PackagePath == "" || (target.TypeName == "" && target.Expr == nil) {
		return nil, fmt.Errorf("missing datasource settings type")
	}

	return typeschema.BuildTargetSchema(loadRes, target, typeschema.SchemaOptions{
		RequireJSONTags:                true,
		FallbackToSimpleUntaggedFields: true,
	})
}

func schemaSpecificityBonus(loadRes *load.Result, registration RuntimeRegistration) int {
	if loadRes == nil {
		return 0
	}

	schema, err := buildSchema(loadRes, registration)
	if err != nil {
		return 0
	}

	return schemaSpecificityScore(schema, 0)
}

func schemaSpecificityScore(schema map[string]any, depth int) int {
	if depth > 6 || len(schema) == 0 {
		return 0
	}

	score := 0
	typ, _ := schema["type"].(string)
	properties, _ := nestedSchemaMap(schema["properties"])

	switch typ {
	case "string", "integer", "number", "boolean":
		score += 30
	case "array":
		score += 10
		if items, ok := schema["items"].(map[string]any); ok {
			score += schemaSpecificityScore(items, depth+1) / 2
		}
	case "object":
		if len(properties) == 0 {
			switch additional := schema["additionalProperties"].(type) {
			case bool:
				if additional {
					score -= 80
				}
			case map[string]any:
				score -= 20
				score += schemaSpecificityScore(additional, depth+1) / 2
			}
		}
	}

	score += len(properties) * 40
	for _, property := range properties {
		child, ok := property.(map[string]any)
		if !ok {
			continue
		}
		score += schemaSpecificityScore(child, depth+1)
	}

	return score
}

func nestedSchemaMap(value any) (map[string]any, bool) {
	items, ok := value.(map[string]any)
	if ok {
		return items, true
	}
	return nil, false
}

func isGenericHelperSchema(schema map[string]any) bool {
	properties, ok := nestedSchemaMap(schema["properties"])
	if !ok || len(properties) != 1 {
		return false
	}

	for _, property := range properties {
		child, ok := property.(map[string]any)
		if !ok {
			return false
		}
		return isBroadObjectSchema(child)
	}

	return false
}

func isBroadObjectSchema(schema map[string]any) bool {
	typ, _ := schema["type"].(string)
	if typ != "object" {
		return false
	}

	properties, _ := nestedSchemaMap(schema["properties"])
	if len(properties) > 0 {
		return false
	}

	switch additional := schema["additionalProperties"].(type) {
	case bool:
		return additional
	case map[string]any:
		return isBroadObjectSchema(additional)
	default:
		return false
	}
}
