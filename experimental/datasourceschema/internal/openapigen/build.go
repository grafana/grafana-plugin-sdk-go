package openapigen

import (
	"encoding/json"
	"go/ast"
	"go/types"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/tools/go/packages"
	"k8s.io/kube-openapi/pkg/validation/spec"

	v0alpha1 "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/datasource/v0alpha1"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/configgen"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/load"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/model"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/querygen"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/ssaresolve"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/pluginspec"
)

type Options struct {
	Dir             string
	Patterns        []string
	BuildFlags      []string
	Report          model.Report
	GenerateSpec    bool
	GenerateQueries bool
}

type Result struct {
	OpenAPI    *pluginspec.OpenAPIExtension
	QueryTypes *v0alpha1.QueryTypeDefinitionList
	Warnings   []model.Warning
}

type genericSettingsUsage struct {
	UsesURL         bool
	UsesHTTPOptions bool
}

func Build(opts Options) (*Result, error) {
	openAPI := &pluginspec.OpenAPIExtension{}
	openAPI.Settings.SecureValues = secureValuesFromFindings(opts.Report.Findings)

	warnings := append([]model.Warning{}, opts.Report.Warnings...)

	usage, err := detectGenericSettingsUsage(opts)
	if err != nil {
		return nil, err
	}
	openAPI.Settings.SecureValues = mergeSecureValues(openAPI.Settings.SecureValues, genericSecureValues(usage))

	if opts.GenerateSpec {
		schemaMap, specWarnings, err := configgen.BuildSchemaFromFindings(configgen.RuntimeOptions{
			Dir:        opts.Dir,
			Patterns:   opts.Patterns,
			BuildFlags: opts.BuildFlags,
		}, opts.Report.Findings)
		if err != nil {
			return nil, err
		}
		schemaMap = mergeSchemaProperties(schemaMap, genericSettingsSchema(usage))
		pruneSecureSpecProperties(schemaMap, openAPI.Settings.SecureValues)
		typedSchema, err := asJSONSchema(schemaMap)
		if err != nil {
			return nil, err
		}
		openAPI.Settings.Spec = typedSchema
		warnings = append(warnings, specWarnings...)
	}

	if opts.GenerateQueries {
		queries, queryWarnings, err := querygen.BuildDefinitionsFromFindings(querygen.RuntimeOptions{
			Dir:        opts.Dir,
			Patterns:   opts.Patterns,
			BuildFlags: opts.BuildFlags,
		}, opts.Report.Findings)
		if err != nil {
			return nil, err
		}
		warnings = append(warnings, queryWarnings...)
		if queries != nil && len(queries.Items) > 0 {
			return &Result{
				OpenAPI:    openAPI,
				QueryTypes: queries,
				Warnings:   dedupeWarnings(warnings),
			}, nil
		}
	}

	return &Result{
		OpenAPI:  openAPI,
		Warnings: dedupeWarnings(warnings),
	}, nil
}

func detectGenericSettingsUsage(opts Options) (genericSettingsUsage, error) {
	loadRes, err := load.Packages(load.Config{
		Dir:        opts.Dir,
		Patterns:   normalizeOpenAPIPatterns(opts.Patterns),
		BuildFlags: opts.BuildFlags,
		NeedModule: true,
	})
	if err != nil {
		return genericSettingsUsage{}, err
	}

	usage := genericSettingsUsage{}
	for _, pkg := range loadRes.Packages {
		if !isLocalPackage(loadRes, pkg) || pkg == nil || pkg.TypesInfo == nil {
			continue
		}
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(node ast.Node) bool {
				sel, ok := node.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if selection := pkg.TypesInfo.Selections[sel]; selection != nil {
					if isDataSourceInstanceSettingsType(selection.Recv()) {
						switch selection.Obj().Name() {
						case "URL":
							usage.UsesURL = true
						case "HTTPClientOptions", "ProxyOptions", "ProxyOptionsFromContext", "ProxyClient":
							usage.UsesHTTPOptions = true
						}
					}
					return true
				}
				if obj, ok := pkg.TypesInfo.Uses[sel.Sel].(*types.Func); ok && obj.Name() == "HTTPClientOptions" {
					if sig, ok := obj.Type().(*types.Signature); ok && sig.Recv() != nil && isDataSourceInstanceSettingsType(sig.Recv().Type()) {
						usage.UsesHTTPOptions = true
					}
				}
				return true
			})
		}
	}

	resolver, err := ssaresolve.Build(loadRes)
	if err != nil {
		return genericSettingsUsage{}, err
	}
	frameworkUsage := resolver.InferFrameworkDataSourceSettingsUsage()
	usage.UsesURL = usage.UsesURL || frameworkUsage.UsesURL
	usage.UsesHTTPOptions = usage.UsesHTTPOptions || frameworkUsage.UsesHTTPOptions

	return usage, nil
}

func normalizeOpenAPIPatterns(patterns []string) []string {
	if len(patterns) == 0 {
		return []string{"./..."}
	}
	return patterns
}

func secureValuesFromFindings(findings []model.Finding) []pluginspec.SecureValueInfo {
	values := make([]pluginspec.SecureValueInfo, 0)
	seen := map[string]struct{}{}

	for _, finding := range findings {
		if finding.Source != model.SourceKindDatasourceSecure {
			continue
		}

		name := finding.Key
		if name == "" {
			name = finding.Pattern
		}
		if name == "" {
			continue
		}

		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}

		values = append(values, pluginspec.SecureValueInfo{
			Key: name,
		})
	}

	sort.Slice(values, func(i int, j int) bool {
		return values[i].Key < values[j].Key
	})

	return values
}

func mergeSecureValues(values []pluginspec.SecureValueInfo, extras []pluginspec.SecureValueInfo) []pluginspec.SecureValueInfo {
	if len(extras) == 0 {
		return values
	}

	merged := append([]pluginspec.SecureValueInfo{}, values...)
	seen := map[string]struct{}{}
	for _, value := range merged {
		if value.Key == "" {
			continue
		}
		seen[value.Key] = struct{}{}
	}

	for _, extra := range extras {
		if extra.Key == "" {
			continue
		}
		if _, ok := seen[extra.Key]; ok {
			continue
		}
		seen[extra.Key] = struct{}{}
		merged = append(merged, extra)
	}

	sort.Slice(merged, func(i int, j int) bool {
		return merged[i].Key < merged[j].Key
	})
	return merged
}

func genericSecureValues(usage genericSettingsUsage) []pluginspec.SecureValueInfo {
	if !usage.UsesHTTPOptions {
		return nil
	}

	names := []string{
		"basicAuthPassword",
		"httpHeaderValue{dynamic}",
		"password",
		"secureSocksProxyPassword",
		"sigV4AccessKey",
		"sigV4SecretKey",
		"sigV4SessionToken",
		"tlsCACert",
		"tlsClientCert",
		"tlsClientKey",
	}

	values := make([]pluginspec.SecureValueInfo, 0, len(names))
	for _, name := range names {
		values = append(values, pluginspec.SecureValueInfo{Key: name})
	}
	return values
}

func genericSettingsSchema(usage genericSettingsUsage) map[string]any {
	properties := map[string]any{}

	if usage.UsesURL {
		properties["url"] = map[string]any{"type": "string"}
	}

	if usage.UsesHTTPOptions {
		for key, schema := range map[string]any{
			"basicAuth":                 map[string]any{"type": "boolean"},
			"basicAuthUser":             map[string]any{"type": "string"},
			"dialTimeout":               map[string]any{"type": "integer"},
			"enableSecureSocksProxy":    map[string]any{"type": "boolean"},
			"httpExpectContinueTimeout": map[string]any{"type": "integer"},
			"httpIdleConnTimeout":       map[string]any{"type": "integer"},
			"httpKeepAlive":             map[string]any{"type": "integer"},
			"httpMaxConnsPerHost":       map[string]any{"type": "integer"},
			"httpMaxIdleConns":          map[string]any{"type": "integer"},
			"httpMaxIdleConnsPerHost":   map[string]any{"type": "integer"},
			"httpTLSHandshakeTimeout":   map[string]any{"type": "integer"},
			"secureSocksProxyUsername":  map[string]any{"type": "string"},
			"serverName":                map[string]any{"type": "string"},
			"sigV4AssumeRoleArn":        map[string]any{"type": "string"},
			"sigV4Auth":                 map[string]any{"type": "boolean"},
			"sigV4AuthType":             map[string]any{"type": "string"},
			"sigV4ExternalId":           map[string]any{"type": "string"},
			"sigV4Profile":              map[string]any{"type": "string"},
			"sigV4Region":               map[string]any{"type": "string"},
			"timeout":                   map[string]any{"type": "integer"},
			"tlsAuth":                   map[string]any{"type": "boolean"},
			"tlsAuthWithCACert":         map[string]any{"type": "boolean"},
			"tlsSkipVerify":             map[string]any{"type": "boolean"},
			"user":                      map[string]any{"type": "string"},
		} {
			properties[key] = schema
		}
	}

	if len(properties) == 0 {
		return nil
	}

	return map[string]any{
		"type":       "object",
		"properties": properties,
	}
}

func mergeSchemaProperties(primary map[string]any, extra map[string]any) map[string]any {
	if len(extra) == 0 {
		return primary
	}
	if len(primary) == 0 {
		return extra
	}

	primaryProps, ok := primary["properties"].(map[string]any)
	if !ok {
		primaryProps = map[string]any{}
		primary["properties"] = primaryProps
	}
	extraProps, ok := extra["properties"].(map[string]any)
	if !ok {
		return primary
	}

	for key, value := range extraProps {
		if _, exists := primaryProps[key]; exists {
			continue
		}
		primaryProps[key] = value
	}

	if _, ok := primary["type"]; !ok {
		primary["type"] = "object"
	}
	return primary
}

func pruneSecureSpecProperties(spec map[string]any, secureValues []pluginspec.SecureValueInfo) {
	if len(spec) == 0 || len(secureValues) == 0 {
		return
	}

	properties, ok := spec["properties"].(map[string]any)
	if !ok || len(properties) == 0 {
		return
	}

	for _, secureValue := range secureValues {
		if secureValue.Key == "" {
			continue
		}
		normalizedSecure := normalizeSchemaKey(secureValue.Key)
		delete(properties, secureValue.Key)
		for key := range properties {
			if strings.EqualFold(key, secureValue.Key) || normalizeSchemaKey(key) == normalizedSecure {
				delete(properties, key)
			}
		}
	}
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

func normalizeSchemaKey(value string) string {
	if value == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(value))
	for _, r := range value {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

func dedupeWarnings(in []model.Warning) []model.Warning {
	out := make([]model.Warning, 0, len(in))
	seen := map[string]struct{}{}
	for _, warning := range in {
		key := warning.Code + "|" + warning.Message + "|" + warning.Position.File
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, warning)
	}
	return out
}

func isLocalPackage(loadRes *load.Result, pkg *packages.Package) bool {
	if loadRes == nil || pkg == nil {
		return false
	}
	if pkg.Module != nil && pkg.Module.Dir != "" {
		for _, root := range loadRes.RootPackages {
			if root != nil && root.Module != nil && root.Module.Dir == pkg.Module.Dir {
				return true
			}
		}
	}
	for _, file := range pkg.GoFiles {
		if strings.HasPrefix(file, loadRes.Dir) {
			return true
		}
	}
	for _, file := range pkg.CompiledGoFiles {
		if strings.HasPrefix(file, loadRes.Dir) {
			return true
		}
	}
	return false
}

func isDataSourceInstanceSettingsType(typ types.Type) bool {
	for {
		ptr, ok := typ.(*types.Pointer)
		if !ok {
			break
		}
		typ = ptr.Elem()
	}

	named, ok := typ.(*types.Named)
	if !ok || named.Obj() == nil || named.Obj().Pkg() == nil {
		return false
	}
	return named.Obj().Pkg().Path() == "github.com/grafana/grafana-plugin-sdk-go/backend" && named.Obj().Name() == "DataSourceInstanceSettings"
}
