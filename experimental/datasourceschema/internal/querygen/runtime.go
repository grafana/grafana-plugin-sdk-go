package querygen

import (
	"encoding/json"
	"fmt"
	"go/token"
	"sort"
	"strings"

	v0alpha1 "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/datasource/v0alpha1"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/load"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/model"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/typeschema"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

const (
	queryTypeDefinitionListKind       = "QueryTypeDefinitionList"
	queryTypeDefinitionListAPIVersion = "datasource.grafana.app/v0alpha1"
)

type RuntimeOptions struct {
	Dir        string
	Patterns   []string
	BuildFlags []string
	PluginID   []string
}

type RuntimeRegistration struct {
	PackagePath    string
	TypeName       string
	Target         *model.TargetRef
	Name           string
	Description    string
	Discriminators []v0alpha1.DiscriminatorFieldValue
	Examples       []v0alpha1.QueryExample
	Changelog      []string
	FunctionNames  []string
}

func BuildDefinitionsInModule(opts RuntimeOptions, registrations []RuntimeRegistration) (*v0alpha1.QueryTypeDefinitionList, error) {
	if len(registrations) == 0 {
		return &v0alpha1.QueryTypeDefinitionList{
			TypeMeta: v0alpha1.TypeMeta{
				Kind:       queryTypeDefinitionListKind,
				APIVersion: queryTypeDefinitionListAPIVersion,
			},
		}, nil
	}

	loadRes, err := load.Packages(load.Config{
		Dir:        opts.Dir,
		Patterns:   normalizePatterns(opts.Patterns),
		BuildFlags: opts.BuildFlags,
	})
	if err != nil {
		return nil, err
	}

	return buildDefinitions(loadRes, registrations)
}

func buildDefinitions(loadRes *load.Result, registrations []RuntimeRegistration) (*v0alpha1.QueryTypeDefinitionList, error) {
	items := make([]v0alpha1.QueryTypeDefinition, 0, len(registrations))
	for _, registration := range registrations {
		target := registration.Target
		if target == nil {
			target = &model.TargetRef{
				PackagePath: registration.PackagePath,
				TypeName:    registration.TypeName,
			}
		}

		schema, err := typeschema.BuildTargetSchema(loadRes, target, typeschema.SchemaOptions{
			IncludeRequired:          false,
			LowerCamelUntaggedFields: true,
		})
		if err != nil {
			return nil, err
		}
		typedSchema, err := asJSONSchema(schema)
		if err != nil {
			return nil, err
		}

		for _, discriminators := range queryDefinitionVariants(registration.Discriminators) {
			name := registration.Name
			if len(discriminators) == 1 {
				name = discriminators[0].Value
			}
			if name == "" {
				name = deriveName(discriminators)
			}
			if name == "" || name == registration.TypeName {
				name = normalizedQueryDefinitionName(registration)
			}

			items = append(items, v0alpha1.QueryTypeDefinition{
				ObjectMeta: v0alpha1.ObjectMeta{
					Name: name,
				},
				Spec: v0alpha1.QueryTypeDefinitionSpec{
					Discriminators: discriminators,
					Description:    registration.Description,
					Schema: v0alpha1.JSONSchema{
						Spec: typedSchema,
					},
					Examples:  registration.Examples,
					Changelog: registration.Changelog,
				},
			})
		}
	}

	return &v0alpha1.QueryTypeDefinitionList{
		TypeMeta: v0alpha1.TypeMeta{
			Kind:       queryTypeDefinitionListKind,
			APIVersion: queryTypeDefinitionListAPIVersion,
		},
		Items: items,
	}, nil
}

func BuildDefinitionsFromFindings(opts RuntimeOptions, findings []model.Finding) (*v0alpha1.QueryTypeDefinitionList, []model.Warning, error) {
	loadRes, err := load.Packages(load.Config{
		Dir:        opts.Dir,
		Patterns:   normalizePatterns(opts.Patterns),
		BuildFlags: opts.BuildFlags,
	})
	if err != nil {
		return nil, nil, err
	}

	registrationByKey := map[string]*RuntimeRegistration{}
	registrationOrder := make([]string, 0)

	for _, finding := range findings {
		if finding.Source != model.SourceKindQueryJSON || finding.Target == nil {
			continue
		}

		key := targetKey(finding.Target)
		registration, ok := registrationByKey[key]
		if !ok {
			registration = &RuntimeRegistration{
				PackagePath: finding.Target.PackagePath,
				TypeName:    finding.Target.TypeName,
				Target:      finding.Target,
				Name:        finding.Target.TypeName,
			}
			registrationByKey[key] = registration
			registrationOrder = append(registrationOrder, key)
		}

		if finding.FunctionName != "" && !containsString(registration.FunctionNames, finding.FunctionName) {
			registration.FunctionNames = append(registration.FunctionNames, finding.FunctionName)
		}
	}

	registrations := make([]RuntimeRegistration, 0, len(registrationOrder))
	for _, key := range registrationOrder {
		registration := registrationByKey[key]
		if registration == nil {
			continue
		}
		registration.Discriminators = inferDiscriminators(loadRes, *registration)
		if len(registration.Discriminators) == 1 {
			registration.Name = registration.Discriminators[0].Value
		} else if registration.Name == "" {
			registration.Name = registration.TypeName
		}
		registrations = append(registrations, *registration)
	}

	definitions, err := buildDefinitions(loadRes, registrations)
	if err != nil {
		return nil, nil, err
	}

	return definitions, nil, nil
}

func deriveName(discriminators []v0alpha1.DiscriminatorFieldValue) string {
	if len(discriminators) == 0 {
		return ""
	}

	values := make([]string, 0, len(discriminators))
	for _, discriminator := range discriminators {
		if discriminator.Value == "" {
			continue
		}
		values = append(values, discriminator.Value)
	}

	return strings.Join(values, "-")
}

func asJSONSchema(v any) (*spec.Schema, error) {
	if s, ok := v.(*spec.Schema); ok {
		return s, nil
	}

	body, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	out := &spec.Schema{}
	if err := json.Unmarshal(body, out); err != nil {
		return nil, err
	}

	return out, nil
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

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func normalizedQueryDefinitionName(registration RuntimeRegistration) string {
	typeName := strings.TrimSpace(registration.TypeName)
	if typeName == "" {
		return ""
	}
	if normalized, ok := friendlyInternalQueryTypeName(typeName); ok {
		return normalized
	}
	return typeName
}

func friendlyInternalQueryTypeName(typeName string) (string, bool) {
	if typeName == "" {
		return "", false
	}
	lower := strings.ToLower(typeName)
	if token.IsExported(typeName) && !strings.HasPrefix(lower, "internal") {
		return "", false
	}
	if strings.Contains(lower, "query") {
		return "Query", true
	}
	return "", false
}

func normalizeDiscriminators(in []v0alpha1.DiscriminatorFieldValue) []v0alpha1.DiscriminatorFieldValue {
	if len(in) == 0 {
		return nil
	}

	out := make([]v0alpha1.DiscriminatorFieldValue, 0, len(in))
	seen := map[string]struct{}{}
	for _, discriminator := range in {
		if discriminator.Field == "" || discriminator.Value == "" {
			continue
		}
		key := discriminator.Field + "\x00" + discriminator.Value
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, discriminator)
	}

	sort.Slice(out, func(i int, j int) bool {
		if out[i].Field != out[j].Field {
			return out[i].Field < out[j].Field
		}
		return out[i].Value < out[j].Value
	})

	if len(out) == 0 {
		return nil
	}
	return out
}

func queryDefinitionVariants(discriminators []v0alpha1.DiscriminatorFieldValue) [][]v0alpha1.DiscriminatorFieldValue {
	if len(discriminators) == 0 {
		return [][]v0alpha1.DiscriminatorFieldValue{nil}
	}
	if len(discriminators) == 1 {
		return [][]v0alpha1.DiscriminatorFieldValue{discriminators}
	}

	variants := make([][]v0alpha1.DiscriminatorFieldValue, 0, len(discriminators))
	for _, discriminator := range discriminators {
		variants = append(variants, []v0alpha1.DiscriminatorFieldValue{discriminator})
	}
	return variants
}
