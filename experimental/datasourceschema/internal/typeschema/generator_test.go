package typeschema

import (
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/load"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/testutil"
	"github.com/stretchr/testify/require"
)

const (
	objectSchemaType = "object"
	stringSchemaType = "string"
)

func TestBuildNamedTypeSchemaHandlesStructTagsAndRequiredFields(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1
`,
		"pkg/models/settings.go": `
package models

// Credentials for authenticated requests.
type Credentials struct {
	// Token used for bearer auth.
	Token string ` + "`json:\"token\"`" + `
}

// Settings describe datasource configuration.
type Settings struct {
	// Human readable datasource name.
	Name       string            ` + "`json:\"name\"`" + `
	// Whether the datasource is enabled.
	Enabled    bool              ` + "`json:\"enabled,omitempty\"`" + `
	Creds      *Credentials      ` + "`json:\"creds,omitempty\"`" + `
	Headers    map[string]string ` + "`json:\"headers\"`" + `
	Namespaces []string          ` + "`json:\"namespaces\"`" + `
	Ignored    string            ` + "`json:\"-\"`" + `
	privateVal string
}
`,
	})

	loadRes, err := load.Packages(load.Config{
		Dir:      dir,
		Patterns: []string{"./..."},
	})
	require.NoError(t, err, "load failed")

	schema, err := BuildNamedTypeSchema(loadRes, "fixture/pkg/models", "Settings")
	require.NoError(t, err, "schema build failed")

	require.Equal(t, draft04, schema["$schema"], "expected draft-04 schema")
	require.Equal(t, objectSchemaType, schema["type"], "expected object type")
	require.Equal(t, "Settings describe datasource configuration.", schema["description"], "expected schema description from type comment")
	require.Equal(t, false, schema["additionalProperties"], "expected additionalProperties=false")

	properties, ok := testutil.NestedMap(schema, "properties")
	require.True(t, ok, "expected properties object, got %#v", schema)
	require.NotContains(t, properties, "Ignored", "did not expect ignored field in schema")
	require.NotContains(t, properties, "privateVal", "did not expect unexported field in schema")

	nameField, ok := testutil.NestedMap(schema, "properties", "name")
	require.True(t, ok, "expected string property for name, got %#v", nameField)
	require.Equal(t, stringSchemaType, nameField["type"], "expected string property for name")
	require.Equal(t, "Human readable datasource name.", nameField["description"], "expected field description for name")

	headersField, ok := testutil.NestedMap(schema, "properties", "headers")
	require.True(t, ok, "expected object property for headers, got %#v", headersField)
	require.Equal(t, objectSchemaType, headersField["type"], "expected object property for headers")
	require.NotContains(t, headersField, "description", "did not expect description on headers field")
	additionalProperties, ok := headersField["additionalProperties"].(map[string]any)
	require.True(t, ok, "expected string additionalProperties for headers, got %#v", headersField["additionalProperties"])
	require.Equal(t, stringSchemaType, additionalProperties["type"], "expected string additionalProperties for headers")

	namespacesField, ok := testutil.NestedMap(schema, "properties", "namespaces")
	require.True(t, ok, "expected array property for namespaces, got %#v", namespacesField)
	require.Equal(t, "array", namespacesField["type"], "expected array property for namespaces")
	items, ok := namespacesField["items"].(map[string]any)
	require.True(t, ok, "expected string array items for namespaces, got %#v", namespacesField["items"])
	require.Equal(t, stringSchemaType, items["type"], "expected string array items for namespaces")

	credsField, ok := testutil.NestedMap(schema, "properties", "creds")
	require.True(t, ok, "expected object property for creds, got %#v", credsField)
	require.Equal(t, objectSchemaType, credsField["type"], "expected object property for creds")
	tokenField, ok := testutil.NestedMap(credsField, "properties", "token")
	require.True(t, ok, "expected nested token string property, got %#v", tokenField)
	require.Equal(t, stringSchemaType, tokenField["type"], "expected nested token string property")
	require.Equal(t, "Token used for bearer auth.", tokenField["description"], "expected nested field description for token")

	require.NotContains(t, schema, "required", "did not expect required list")
}

func TestBuildNamedTypeSchemaExtractsSimpleEnums(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1
`,
		"pkg/models/query.go": `
package models

// Query execution mode.
type Mode string

const (
	// Use mode one.
	ModeOne Mode = "one"
	// Use mode two.
	ModeTwo Mode = "two"
)

type Query struct {
	Mode Mode ` + "`json:\"mode\"`" + `
}
`,
	})

	loadRes, err := load.Packages(load.Config{
		Dir:      dir,
		Patterns: []string{"./..."},
	})
	require.NoError(t, err, "load failed")

	schema, err := BuildNamedTypeSchema(loadRes, "fixture/pkg/models", "Query")
	require.NoError(t, err, "schema build failed")

	modeField, ok := testutil.NestedMap(schema, "properties", "mode")
	require.True(t, ok, "expected mode field in schema, got %#v", schema)
	enumValues, ok := modeField["enum"].([]any)
	require.True(t, ok, "expected enum values for mode field, got %#v", modeField["enum"])
	require.Equal(t, []any{"one", "two"}, enumValues, "unexpected enum values")
	description, _ := modeField["description"].(string)
	require.Contains(t, description, "Query execution mode.", "expected enum type description")
	require.Contains(t, description, `Possible enum values:`, "expected enum values section")
	require.Contains(t, description, `Use mode one.`, "expected enum value comments in description")
	require.Contains(t, description, `Use mode two.`, "expected enum value comments in description")
}

func TestBuildNamedTypeSchemaSupportsUnexportedRootTypes(t *testing.T) {
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

	loadRes, err := load.Packages(load.Config{
		Dir:      dir,
		Patterns: []string{"./..."},
	})
	require.NoError(t, err, "load failed")

	schema, err := BuildNamedTypeSchema(loadRes, "fixture/pkg/models", "settings")
	require.NoError(t, err, "schema build failed")
	_, ok := testutil.NestedMap(schema, "properties", "name")
	require.True(t, ok, "expected name property in schema, got %#v", schema)
}

func TestBuildNamedTypeSchemaSuppressesSectionHeaderFieldComments(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1
`,
		"pkg/models/settings.go": `
package models

type Settings struct {
	// Security
	AllowedHosts []string ` + "`json:\"allowedHosts,omitempty\"`" + `

	// Human readable explanation for cookies.
	KeepCookies []string ` + "`json:\"keepCookies,omitempty\"`" + `
}
`,
	})

	loadRes, err := load.Packages(load.Config{
		Dir:      dir,
		Patterns: []string{"./..."},
	})
	require.NoError(t, err, "load failed")

	schema, err := BuildNamedTypeSchema(loadRes, "fixture/pkg/models", "Settings")
	require.NoError(t, err, "schema build failed")

	allowedHostsField, ok := testutil.NestedMap(schema, "properties", "allowedHosts")
	require.True(t, ok, "expected allowedHosts property, got %#v", schema)
	require.NotContains(t, allowedHostsField, "description", "expected section header comment to be suppressed")

	keepCookiesField, ok := testutil.NestedMap(schema, "properties", "keepCookies")
	require.True(t, ok, "expected keepCookies property, got %#v", schema)
	require.Equal(t, "Human readable explanation for cookies.", keepCookiesField["description"], "expected useful field description to remain")
}

func TestBuildNamedTypeSchemaWithOptionsRequiresJSONTags(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1
`,
		"pkg/models/settings.go": `
package models

type Settings struct {
	Name   string ` + "`json:\"name\"`" + `
	Inputs []string
}
`,
	})

	loadRes, err := load.Packages(load.Config{
		Dir:      dir,
		Patterns: []string{"./..."},
	})
	require.NoError(t, err, "load failed")

	schema, err := BuildNamedTypeSchemaWithOptions(loadRes, "fixture/pkg/models", "Settings", SchemaOptions{
		IncludeRequired: true,
		RequireJSONTags: true,
	})
	require.NoError(t, err, "schema build failed")

	properties, ok := testutil.NestedMap(schema, "properties")
	require.True(t, ok, "expected properties object, got %#v", schema)
	require.ElementsMatch(t, []string{"name"}, testutil.KeysOfMap(properties), "expected exact tagged field set")
}

func TestBuildNamedTypeSchemaSkipsDashedOmitEmptyTags(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1
`,
		"pkg/models/settings.go": `
package models

type Settings struct {
	Name     string ` + "`json:\"name\"`" + `
	Password string ` + "`json:\"-,omitempty\"`" + `
}
`,
	})

	loadRes, err := load.Packages(load.Config{
		Dir:      dir,
		Patterns: []string{"./..."},
	})
	require.NoError(t, err, "load failed")

	schema, err := BuildNamedTypeSchema(loadRes, "fixture/pkg/models", "Settings")
	require.NoError(t, err, "schema build failed")

	properties, ok := testutil.NestedMap(schema, "properties")
	require.True(t, ok, "expected properties object, got %#v", schema)
	require.NotContains(t, properties, "-", "did not expect dashed property in schema")
}

func TestBuildNamedTypeSchemaMapsUUIDToString(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1

require github.com/google/uuid v0.0.0

replace github.com/google/uuid => ./stubs/google-uuid
`,
		"stubs/google-uuid/go.mod": `
module github.com/google/uuid

go 1.26.1
`,
		"stubs/google-uuid/uuid.go": `
package uuid

type UUID [16]byte
`,
		"pkg/models/query.go": `
package models

import "github.com/google/uuid"

type Query struct {
	ID *uuid.UUID ` + "`json:\"id\"`" + `
}
`,
	})

	loadRes, err := load.Packages(load.Config{
		Dir:      dir,
		Patterns: []string{"./..."},
	})
	require.NoError(t, err, "load failed")

	schema, err := BuildNamedTypeSchema(loadRes, "fixture/pkg/models", "Query")
	require.NoError(t, err, "schema build failed")

	idField, ok := testutil.NestedMap(schema, "properties", "id")
	require.True(t, ok, "expected id field in schema, got %#v", schema)
	require.Equal(t, stringSchemaType, idField["type"], "expected uuid field to render as string")
	require.Equal(t, "uuid", idField["format"], "expected uuid format")
}

func TestBuildNamedTypeSchemaWithOptionsCanEnableRequired(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1
`,
		"pkg/models/query.go": `
package models

type Query struct {
	QueryType string ` + "`json:\"queryType\"`" + `
	Logs struct {
		Query string ` + "`json:\"query\"`" + `
	} ` + "`json:\"logs\"`" + `
}
`,
	})

	loadRes, err := load.Packages(load.Config{
		Dir:      dir,
		Patterns: []string{"./..."},
	})
	require.NoError(t, err, "load failed")

	schema, err := BuildNamedTypeSchemaWithOptions(loadRes, "fixture/pkg/models", "Query", SchemaOptions{
		IncludeRequired: true,
	})
	require.NoError(t, err, "schema build failed")

	required, ok := testStringSlice(schema["required"])
	require.True(t, ok, "expected required list, got %#v", schema["required"])
	require.Equal(t, "logs,queryType", strings.Join(required, ","), "unexpected required list")

	logsField, ok := testutil.NestedMap(schema, "properties", "logs")
	require.True(t, ok, "expected logs field in schema, got %#v", schema)
	logsRequired, ok := testStringSlice(logsField["required"])
	require.True(t, ok, "expected nested required list, got %#v", logsField["required"])
	require.Equal(t, "query", strings.Join(logsRequired, ","), "unexpected nested required list")
}

func TestBuildNamedTypeSchemaWithOptionsLowerCasesUntaggedQueryFields(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1
`,
		"pkg/models/query.go": `
package models

type Query struct {
	MaxDataPoints int64
	Query string
	APIKey string
}
`,
	})

	loadRes, err := load.Packages(load.Config{
		Dir:      dir,
		Patterns: []string{"./..."},
	})
	require.NoError(t, err, "load failed")

	schema, err := BuildNamedTypeSchemaWithOptions(loadRes, "fixture/pkg/models", "Query", SchemaOptions{
		LowerCamelUntaggedFields: true,
	})
	require.NoError(t, err, "schema build failed")

	properties, ok := testutil.NestedMap(schema, "properties")
	require.True(t, ok, "expected properties object, got %#v", schema)
	require.ElementsMatch(t, []string{"maxDataPoints", "query", "apiKey"}, testutil.KeysOfMap(properties), "expected exact lower camel property set")
}

func testStringSlice(value any) ([]string, bool) {
	items, ok := value.([]any)
	if !ok {
		return nil, false
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if !ok {
			return nil, false
		}
		out = append(out, text)
	}
	return out, true
}
