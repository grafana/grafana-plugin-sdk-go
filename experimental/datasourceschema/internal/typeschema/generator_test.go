package typeschema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/load"
)

func TestBuildNamedTypeSchemaHandlesStructTagsAndRequiredFields(t *testing.T) {
	dir := writeFixtureModule(t, map[string]string{
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
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	schema, err := BuildNamedTypeSchema(loadRes, "fixture/pkg/models", "Settings")
	if err != nil {
		t.Fatalf("schema build failed: %v", err)
	}

	if schema["$schema"] != draft04 {
		t.Fatalf("expected draft-04 schema, got %#v", schema["$schema"])
	}
	if schema["type"] != "object" {
		t.Fatalf("expected object type, got %#v", schema["type"])
	}
	if schema["description"] != "Settings describe datasource configuration." {
		t.Fatalf("expected schema description from type comment, got %#v", schema["description"])
	}
	if schema["additionalProperties"] != false {
		t.Fatalf("expected additionalProperties=false, got %#v", schema["additionalProperties"])
	}

	properties, ok := testNestedMap(schema, "properties")
	if !ok {
		t.Fatalf("expected properties object, got %#v", schema)
	}
	if _, ok := properties["Ignored"]; ok {
		t.Fatalf("did not expect ignored field in schema, got %#v", properties)
	}
	if _, ok := properties["privateVal"]; ok {
		t.Fatalf("did not expect unexported field in schema, got %#v", properties)
	}

	nameField, ok := testNestedMap(schema, "properties", "name")
	if !ok || nameField["type"] != "string" {
		t.Fatalf("expected string property for name, got %#v", nameField)
	}
	if nameField["description"] != "Human readable datasource name." {
		t.Fatalf("expected field description for name, got %#v", nameField["description"])
	}

	headersField, ok := testNestedMap(schema, "properties", "headers")
	if !ok || headersField["type"] != "object" {
		t.Fatalf("expected object property for headers, got %#v", headersField)
	}
	if _, ok := headersField["description"]; ok {
		t.Fatalf("did not expect description on headers field, got %#v", headersField["description"])
	}
	additionalProperties, ok := headersField["additionalProperties"].(map[string]any)
	if !ok || additionalProperties["type"] != "string" {
		t.Fatalf("expected string additionalProperties for headers, got %#v", headersField["additionalProperties"])
	}

	namespacesField, ok := testNestedMap(schema, "properties", "namespaces")
	if !ok || namespacesField["type"] != "array" {
		t.Fatalf("expected array property for namespaces, got %#v", namespacesField)
	}
	items, ok := namespacesField["items"].(map[string]any)
	if !ok || items["type"] != "string" {
		t.Fatalf("expected string array items for namespaces, got %#v", namespacesField["items"])
	}

	credsField, ok := testNestedMap(schema, "properties", "creds")
	if !ok || credsField["type"] != "object" {
		t.Fatalf("expected object property for creds, got %#v", credsField)
	}
	tokenField, ok := testNestedMap(credsField, "properties", "token")
	if !ok || tokenField["type"] != "string" {
		t.Fatalf("expected nested token string property, got %#v", tokenField)
	}
	if tokenField["description"] != "Token used for bearer auth." {
		t.Fatalf("expected nested field description for token, got %#v", tokenField["description"])
	}

	if _, ok := schema["required"]; ok {
		t.Fatalf("did not expect required list, got %#v", schema["required"])
	}
}

func TestBuildNamedTypeSchemaExtractsSimpleEnums(t *testing.T) {
	dir := writeFixtureModule(t, map[string]string{
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
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	schema, err := BuildNamedTypeSchema(loadRes, "fixture/pkg/models", "Query")
	if err != nil {
		t.Fatalf("schema build failed: %v", err)
	}

	modeField, ok := testNestedMap(schema, "properties", "mode")
	if !ok {
		t.Fatalf("expected mode field in schema, got %#v", schema)
	}
	enumValues, ok := modeField["enum"].([]any)
	if !ok {
		t.Fatalf("expected enum values for mode field, got %#v", modeField["enum"])
	}
	if len(enumValues) != 2 || enumValues[0] != "one" || enumValues[1] != "two" {
		t.Fatalf("unexpected enum values %#v", enumValues)
	}
	description, _ := modeField["description"].(string)
	if !strings.Contains(description, "Query execution mode.") {
		t.Fatalf("expected enum type description, got %#v", description)
	}
	if !strings.Contains(description, `Possible enum values:`) {
		t.Fatalf("expected enum values section, got %#v", description)
	}
	if !strings.Contains(description, `Use mode one.`) || !strings.Contains(description, `Use mode two.`) {
		t.Fatalf("expected enum value comments in description, got %#v", description)
	}
}

func TestBuildNamedTypeSchemaSupportsUnexportedRootTypes(t *testing.T) {
	dir := writeFixtureModule(t, map[string]string{
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
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	schema, err := BuildNamedTypeSchema(loadRes, "fixture/pkg/models", "settings")
	if err != nil {
		t.Fatalf("schema build failed: %v", err)
	}
	if _, ok := testNestedMap(schema, "properties", "name"); !ok {
		t.Fatalf("expected name property in schema, got %#v", schema)
	}
}

func TestBuildNamedTypeSchemaSuppressesSectionHeaderFieldComments(t *testing.T) {
	dir := writeFixtureModule(t, map[string]string{
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
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	schema, err := BuildNamedTypeSchema(loadRes, "fixture/pkg/models", "Settings")
	if err != nil {
		t.Fatalf("schema build failed: %v", err)
	}

	allowedHostsField, ok := testNestedMap(schema, "properties", "allowedHosts")
	if !ok {
		t.Fatalf("expected allowedHosts property, got %#v", schema)
	}
	if _, ok := allowedHostsField["description"]; ok {
		t.Fatalf("expected section header comment to be suppressed, got %#v", allowedHostsField["description"])
	}

	keepCookiesField, ok := testNestedMap(schema, "properties", "keepCookies")
	if !ok {
		t.Fatalf("expected keepCookies property, got %#v", schema)
	}
	if keepCookiesField["description"] != "Human readable explanation for cookies." {
		t.Fatalf("expected useful field description to remain, got %#v", keepCookiesField["description"])
	}
}

func TestBuildNamedTypeSchemaWithOptionsRequiresJSONTags(t *testing.T) {
	dir := writeFixtureModule(t, map[string]string{
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
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	schema, err := BuildNamedTypeSchemaWithOptions(loadRes, "fixture/pkg/models", "Settings", SchemaOptions{
		IncludeRequired: true,
		RequireJSONTags: true,
	})
	if err != nil {
		t.Fatalf("schema build failed: %v", err)
	}

	properties, ok := testNestedMap(schema, "properties")
	if !ok {
		t.Fatalf("expected properties object, got %#v", schema)
	}
	if _, ok := properties["name"]; !ok {
		t.Fatalf("expected tagged field in schema, got %#v", properties)
	}
	if _, ok := properties["Inputs"]; ok {
		t.Fatalf("did not expect untagged field in schema, got %#v", properties)
	}
}

func TestBuildNamedTypeSchemaSkipsDashedOmitEmptyTags(t *testing.T) {
	dir := writeFixtureModule(t, map[string]string{
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
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	schema, err := BuildNamedTypeSchema(loadRes, "fixture/pkg/models", "Settings")
	if err != nil {
		t.Fatalf("schema build failed: %v", err)
	}

	properties, ok := testNestedMap(schema, "properties")
	if !ok {
		t.Fatalf("expected properties object, got %#v", schema)
	}
	if _, ok := properties["-"]; ok {
		t.Fatalf("did not expect dashed property in schema, got %#v", properties)
	}
}

func TestBuildNamedTypeSchemaMapsUUIDToString(t *testing.T) {
	dir := writeFixtureModule(t, map[string]string{
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
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	schema, err := BuildNamedTypeSchema(loadRes, "fixture/pkg/models", "Query")
	if err != nil {
		t.Fatalf("schema build failed: %v", err)
	}

	idField, ok := testNestedMap(schema, "properties", "id")
	if !ok {
		t.Fatalf("expected id field in schema, got %#v", schema)
	}
	if idField["type"] != "string" {
		t.Fatalf("expected uuid field to render as string, got %#v", idField)
	}
	if idField["format"] != "uuid" {
		t.Fatalf("expected uuid format, got %#v", idField)
	}
}

func TestBuildNamedTypeSchemaWithOptionsCanEnableRequired(t *testing.T) {
	dir := writeFixtureModule(t, map[string]string{
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
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	schema, err := BuildNamedTypeSchemaWithOptions(loadRes, "fixture/pkg/models", "Query", SchemaOptions{
		IncludeRequired: true,
	})
	if err != nil {
		t.Fatalf("schema build failed: %v", err)
	}

	required, ok := testStringSlice(schema["required"])
	if !ok {
		t.Fatalf("expected required list, got %#v", schema["required"])
	}
	if strings.Join(required, ",") != "logs,queryType" {
		t.Fatalf("unexpected required list: %#v", required)
	}

	logsField, ok := testNestedMap(schema, "properties", "logs")
	if !ok {
		t.Fatalf("expected logs field in schema, got %#v", schema)
	}
	logsRequired, ok := testStringSlice(logsField["required"])
	if !ok {
		t.Fatalf("expected nested required list, got %#v", logsField["required"])
	}
	if strings.Join(logsRequired, ",") != "query" {
		t.Fatalf("unexpected nested required list: %#v", logsRequired)
	}
}

func TestBuildNamedTypeSchemaWithOptionsLowerCasesUntaggedQueryFields(t *testing.T) {
	dir := writeFixtureModule(t, map[string]string{
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
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	schema, err := BuildNamedTypeSchemaWithOptions(loadRes, "fixture/pkg/models", "Query", SchemaOptions{
		LowerCamelUntaggedFields: true,
	})
	if err != nil {
		t.Fatalf("schema build failed: %v", err)
	}

	properties, ok := testNestedMap(schema, "properties")
	if !ok {
		t.Fatalf("expected properties object, got %#v", schema)
	}
	if _, ok := properties["maxDataPoints"]; !ok {
		t.Fatalf("expected lower camel maxDataPoints property, got %#v", properties)
	}
	if _, ok := properties["query"]; !ok {
		t.Fatalf("expected lower camel query property, got %#v", properties)
	}
	if _, ok := properties["apiKey"]; !ok {
		t.Fatalf("expected acronym-aware apiKey property, got %#v", properties)
	}
}

func writeFixtureModule(t *testing.T, files map[string]string) string {
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

func testNestedMap(value map[string]any, keys ...string) (map[string]any, bool) {
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
