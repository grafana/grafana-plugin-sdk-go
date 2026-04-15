package configgen

import (
	"path/filepath"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/model"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestBuildSchemaInModule(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
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
	require.NoError(t, err, "build failed")

	properties, ok := testutil.NestedMap(schema, "properties")
	require.True(t, ok, "expected properties in schema, got %#v", schema)
	require.ElementsMatch(t, []string{"name", "enabled"}, testutil.KeysOfMap(properties), "expected exact property set")
}

func TestBuildSchemaFromFindingsWarnsOnMultipleTypes(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
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
	require.NoError(t, err, "unexpected error")
	require.NotNil(t, schema, "expected generated schema")
	require.Len(t, warnings, 1, "expected one warning")
	require.Equal(t, "datasource_multiple_types", warnings[0].Code, "expected multiple type warning")
}

func TestBuildSchemaFromFindingsPrefersLoadSettingsTargetWithoutHelperWarning(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
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
	settingsLine, settingsColumn := testutil.PositionOfSnippet(t, settingsFile, "Settings struct")
	kerberosLine, kerberosColumn := testutil.PositionOfSnippet(t, kerberosFile, "Auth struct")

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
	require.NoError(t, err, "unexpected error")
	require.Empty(t, warnings, "expected no warning for low-scoring helper type")
	_, ok := testutil.NestedMap(schema, "properties", "user")
	require.True(t, ok, "expected schema for Settings target, got %#v", schema)
	_, ok = testutil.NestedMap(schema, "properties", "credentialCache")
	require.False(t, ok, "did not expect kerberos schema, got %#v", schema)
}

func TestBuildSchemaFromFindingsSuppressesGenericHelperDecodeWarning(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
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
	require.NoError(t, err, "unexpected error")
	require.Empty(t, warnings, "expected helper decode to be ignored without warning")
	_, ok := testutil.NestedMap(schema, "properties", "servers")
	require.True(t, ok, "expected schema for Options target, got %#v", schema)
	_, ok = testutil.NestedMap(schema, "properties", "auth", "properties", "id")
	require.True(t, ok, "expected auth.id property in schema, got %#v", schema)
}

func TestBuildSchemaFromFindingsSupportsUnexportedTypes(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
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
	require.NoError(t, err, "unexpected error")
	require.Empty(t, warnings, "expected no warnings")
	_, ok := testutil.NestedMap(schema, "properties", "name")
	require.True(t, ok, "expected name property in schema, got %#v", schema)
}

func TestBuildSchemaFromFindingsSupportsAnonymousTargets(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
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
	line, column := testutil.PositionOfSnippet(t, file, "&cfg")

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
	require.NoError(t, err, "unexpected error")
	require.Empty(t, warnings, "expected no warnings")
	_, ok := testutil.NestedMap(schema, "properties", "name")
	require.True(t, ok, "expected name property in schema, got %#v", schema)
}

func TestBuildSchemaFromFindingsFallsBackToSimpleUntaggedFields(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
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
	require.NoError(t, err, "unexpected error")
	require.Empty(t, warnings, "expected no warnings")
	_, ok := testutil.NestedMap(schema, "properties", "URL")
	require.True(t, ok, "expected URL property in schema, got %#v", schema)
	_, ok = testutil.NestedMap(schema, "properties", "User")
	require.True(t, ok, "expected User property in schema, got %#v", schema)
	_, ok = testutil.NestedMap(schema, "properties", "Hosting")
	require.True(t, ok, "expected Hosting property in schema, got %#v", schema)
	_, ok = testutil.NestedMap(schema, "properties", "HttpClientOptions")
	require.False(t, ok, "did not expect complex untagged helper field in schema, got %#v", schema)
}
