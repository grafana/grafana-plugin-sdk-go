package configgen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/model"
)

func TestBuildSchemaInModule(t *testing.T) {
	dir := writeRuntimeFixture(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1
`,
		"pkg/models/settings.go": `
package models

type Settings struct {
	Name string ` + "`json:\"name\"`" + `
	Enabled bool ` + "`json:\"enabled\"`" + `
}
`,
	})

	schema, err := BuildSchemaInModule(RuntimeOptions{Dir: dir}, RuntimeRegistration{
		PackagePath: "fixture/pkg/models",
		TypeName:    "Settings",
	})
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}

	properties, ok := nestedMap(schema, "properties")
	if !ok {
		t.Fatalf("expected properties in schema, got %#v", schema)
	}
	if _, ok := properties["name"]; !ok {
		t.Fatalf("expected name property, got %#v", properties)
	}
	if _, ok := properties["enabled"]; !ok {
		t.Fatalf("expected enabled property, got %#v", properties)
	}
}

func TestBuildSchemaFromFindingsWarnsOnMultipleTypes(t *testing.T) {
	dir := writeRuntimeFixture(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1
`,
		"pkg/models/settings.go": `
package models

type Settings struct {
	Name string ` + "`json:\"name\"`" + `
}

type OtherSettings struct {
	Enabled bool ` + "`json:\"enabled\"`" + `
}
`,
	})

	schema, warnings, err := BuildSchemaFromFindings(RuntimeOptions{Dir: dir}, []model.Finding{
		{
			Source: model.SourceKindDatasourceJSON,
			Target: &model.TargetRef{PackagePath: "fixture/pkg/models", TypeName: "Settings"},
		},
		{
			Source: model.SourceKindDatasourceJSON,
			Target: &model.TargetRef{PackagePath: "fixture/pkg/models", TypeName: "OtherSettings"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema == nil {
		t.Fatalf("expected generated schema")
	}
	if len(warnings) != 1 || warnings[0].Code != "datasource_multiple_types" {
		t.Fatalf("expected multiple type warning, got %#v", warnings)
	}
}

func TestBuildSchemaFromFindingsPrefersLoadSettingsTargetWithoutHelperWarning(t *testing.T) {
	dir := writeRuntimeFixture(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1
`,
		"pkg/plugin/settings.go": `
package plugin

type Settings struct {
	User string ` + "`json:\"user\"`" + `
}
`,
		"pkg/kerberos/kerberos.go": `
package kerberos

type Auth struct {
	CredentialCache string ` + "`json:\"credentialCache\"`" + `
}
`,
	})
	settingsFile := filepath.Join(dir, "pkg/plugin/settings.go")
	kerberosFile := filepath.Join(dir, "pkg/kerberos/kerberos.go")
	settingsLine, settingsColumn := positionOfSnippet(t, settingsFile, "Settings struct")
	kerberosLine, kerberosColumn := positionOfSnippet(t, kerberosFile, "Auth struct")

	schema, warnings, err := BuildSchemaFromFindings(RuntimeOptions{Dir: dir}, []model.Finding{
		{
			Source:       model.SourceKindDatasourceJSON,
			FunctionName: "GetKerberosSettings",
			Position: model.Position{
				File:   kerberosFile,
				Line:   kerberosLine,
				Column: kerberosColumn,
			},
			Target: &model.TargetRef{
				PackagePath: "fixture/pkg/kerberos",
				TypeName:    "Auth",
			},
		},
		{
			Source:       model.SourceKindDatasourceJSON,
			FunctionName: "LoadSettings",
			Position: model.Position{
				File:   settingsFile,
				Line:   settingsLine,
				Column: settingsColumn,
			},
			Target: &model.TargetRef{
				PackagePath: "fixture/pkg/plugin",
				TypeName:    "Settings",
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warning for low-scoring helper type, got %#v", warnings)
	}
	if _, ok := nestedMap(schema, "properties", "user"); !ok {
		t.Fatalf("expected schema for Settings target, got %#v", schema)
	}
	if _, ok := nestedMap(schema, "properties", "credentialCache"); ok {
		t.Fatalf("did not expect kerberos schema, got %#v", schema)
	}
}

func TestBuildSchemaFromFindingsSuppressesGenericHelperDecodeWarning(t *testing.T) {
	dir := writeRuntimeFixture(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1
`,
		"pkg/openapids/options.go": `
package openapids

type ServersOptions struct {
	URL string ` + "`json:\"url\"`" + `
}

type Options struct {
	Servers ServersOptions ` + "`json:\"servers\"`" + `
	Auth    struct {
		ID string ` + "`json:\"id\"`" + `
	} ` + "`json:\"auth\"`" + `
}

type OptionsWithCreds struct {
	Auth map[string]interface{} ` + "`json:\"auth\"`" + `
}
`,
	})

	schema, warnings, err := BuildSchemaFromFindings(RuntimeOptions{Dir: dir}, []model.Finding{
		{
			Source:       model.SourceKindDatasourceJSON,
			FunctionName: "loadOptionsFromPluginSettings",
			Target: &model.TargetRef{
				PackagePath: "fixture/pkg/openapids",
				TypeName:    "Options",
			},
		},
		{
			Source:       model.SourceKindDatasourceJSON,
			FunctionName: "loadOptionsFromPluginSettings",
			Target: &model.TargetRef{
				PackagePath: "fixture/pkg/openapids",
				TypeName:    "OptionsWithCreds",
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected helper decode to be ignored without warning, got %#v", warnings)
	}
	if _, ok := nestedMap(schema, "properties", "servers"); !ok {
		t.Fatalf("expected schema for Options target, got %#v", schema)
	}
	if _, ok := nestedMap(schema, "properties", "auth", "properties", "id"); !ok {
		t.Fatalf("expected auth.id property in schema, got %#v", schema)
	}
}

func TestBuildSchemaFromFindingsSupportsUnexportedTypes(t *testing.T) {
	dir := writeRuntimeFixture(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1
`,
		"pkg/models/settings.go": `
package models

type settings struct {
	Name string ` + "`json:\"name\"`" + `
}
`,
	})

	schema, warnings, err := BuildSchemaFromFindings(RuntimeOptions{Dir: dir}, []model.Finding{{
		Source: model.SourceKindDatasourceJSON,
		Target: &model.TargetRef{PackagePath: "fixture/pkg/models", TypeName: "settings"},
	}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	if _, ok := nestedMap(schema, "properties", "name"); !ok {
		t.Fatalf("expected name property in schema, got %#v", schema)
	}
}

func TestBuildSchemaFromFindingsSupportsAnonymousTargets(t *testing.T) {
	dir := writeRuntimeFixture(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1
`,
		"pkg/plugin/settings.go": `
package plugin

func decodeTarget() any {
	var cfg struct {
		Name string ` + "`json:\"name\"`" + `
	}
	return &cfg
}
`,
	})
	file := filepath.Join(dir, "pkg/plugin/settings.go")
	line, column := positionOfSnippet(t, file, "&cfg")

	schema, warnings, err := BuildSchemaFromFindings(RuntimeOptions{Dir: dir}, []model.Finding{{
		Source: model.SourceKindDatasourceJSON,
		Target: &model.TargetRef{
			PackagePath: "fixture/pkg/plugin",
			TypeString:  "struct{Name string \"json:\\\"name\\\"\"}",
			Expr: &model.Position{
				File:   file,
				Line:   line,
				Column: column,
			},
		},
	}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	if _, ok := nestedMap(schema, "properties", "name"); !ok {
		t.Fatalf("expected name property in schema, got %#v", schema)
	}
}

func TestBuildSchemaFromFindingsFallsBackToSimpleUntaggedFields(t *testing.T) {
	dir := writeRuntimeFixture(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1
`,
		"pkg/httpclient/options.go": `
package httpclient

type Options struct {
	Timeout int
}
`,
		"pkg/models/settings.go": `
package models

import "fixture/pkg/httpclient"

type Settings struct {
	URL               string
	User              string
	Hosting           string
	HttpClientOptions httpclient.Options
}
`,
	})

	schema, warnings, err := BuildSchemaFromFindings(RuntimeOptions{Dir: dir}, []model.Finding{{
		Source: model.SourceKindDatasourceJSON,
		Target: &model.TargetRef{
			PackagePath: "fixture/pkg/models",
			TypeName:    "Settings",
		},
	}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}
	if _, ok := nestedMap(schema, "properties", "URL"); !ok {
		t.Fatalf("expected URL property in schema, got %#v", schema)
	}
	if _, ok := nestedMap(schema, "properties", "User"); !ok {
		t.Fatalf("expected User property in schema, got %#v", schema)
	}
	if _, ok := nestedMap(schema, "properties", "Hosting"); !ok {
		t.Fatalf("expected Hosting property in schema, got %#v", schema)
	}
	if _, ok := nestedMap(schema, "properties", "HttpClientOptions"); ok {
		t.Fatalf("did not expect complex untagged helper field in schema, got %#v", schema)
	}
}

func writeRuntimeFixture(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	for name, content := range files {
		fullPath := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("mkdir failed for %s: %v", fullPath, err)
		}
		if err := os.WriteFile(fullPath, []byte(strings.TrimLeft(content, "\n")), 0o644); err != nil {
			t.Fatalf("write failed for %s: %v", fullPath, err)
		}
	}

	return dir
}

func nestedMap(value map[string]any, keys ...string) (map[string]any, bool) {
	current := value
	for _, key := range keys {
		next, ok := current[key]
		if !ok {
			return nil, false
		}

		current, ok = next.(map[string]any)
		if !ok {
			return nil, false
		}
	}

	return current, true
}

func positionOfSnippet(t *testing.T, path string, snippet string) (int, int) {
	t.Helper()

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	lines := strings.Split(string(body), "\n")
	for lineIndex, line := range lines {
		column := strings.Index(line, snippet)
		if column >= 0 {
			return lineIndex + 1, column + 1
		}
	}

	t.Fatalf("snippet %q not found in %s", snippet, path)
	return 0, 0
}
