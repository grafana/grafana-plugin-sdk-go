package datasourceschema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateOpenAPIPreservesLoadOptions(t *testing.T) {
	dir := writeFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1

require github.com/grafana/grafana-plugin-sdk-go v0.0.0

replace github.com/grafana/grafana-plugin-sdk-go => ./stubs/grafana-plugin-sdk-go
`,
		"stubs/grafana-plugin-sdk-go/go.mod": `
module github.com/grafana/grafana-plugin-sdk-go

go 1.26.1
`,
		"stubs/grafana-plugin-sdk-go/backend/backend.go": `
package backend

type DataSourceInstanceSettings struct {
	JSONData []byte
}
`,
		"plugin/plugin.go": `
package plugin

import (
	"encoding/json"

	"fixture/plugin/models"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func LoadSettings(cfg backend.DataSourceInstanceSettings) error {
	var settings models.Settings
	return json.Unmarshal(cfg.JSONData, &settings)
}
`,
		"plugin/models/models_tagged.go": `
//go:build custom

package models

type Settings struct {
	Name string ` + "`json:\"name\"`" + `
}
`,
		"broken/broken.go": `
package broken

var _ = doesNotExist
`,
	})

	result, err := GenerateOpenAPI(OpenAPIOptions{
		Dir:        dir,
		Patterns:   []string{"./plugin/..."},
		BuildFlags: []string{"-tags=custom"},
	})
	if err != nil {
		t.Fatalf("generate openapi failed: %v", err)
	}

	if !strings.Contains(string(result.Body), `"name"`) {
		t.Fatalf("expected generated schema to include tagged settings fields, got %s", result.Body)
	}
	if !strings.Contains(string(result.Body), `"settings"`) {
		t.Fatalf("expected pluginspec settings wrapper, got %s", result.Body)
	}
}

func TestGenerateOpenAPIReturnsWarningsForAmbiguousSettingsTargets(t *testing.T) {
	dir := writeFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1

require github.com/grafana/grafana-plugin-sdk-go v0.0.0

replace github.com/grafana/grafana-plugin-sdk-go => ./stubs/grafana-plugin-sdk-go
`,
		"stubs/grafana-plugin-sdk-go/go.mod": `
module github.com/grafana/grafana-plugin-sdk-go

go 1.26.1
`,
		"stubs/grafana-plugin-sdk-go/backend/backend.go": `
package backend

type DataSourceInstanceSettings struct {
	JSONData []byte
}
`,
		"plugin/plugin.go": `
package plugin

import (
	"encoding/json"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type Settings struct {
	Name string ` + "`json:\"name\"`" + `
}

type OtherSettings struct {
	Enabled bool ` + "`json:\"enabled\"`" + `
}

func DecodePrimary(cfg backend.DataSourceInstanceSettings) error {
	var settings Settings
	return json.Unmarshal(cfg.JSONData, &settings)
}

func DecodeSecondary(cfg backend.DataSourceInstanceSettings) error {
	var settings OtherSettings
	return json.Unmarshal(cfg.JSONData, &settings)
}
`,
	})

	result, err := GenerateOpenAPI(OpenAPIOptions{
		Dir: dir,
	})
	if err != nil {
		t.Fatalf("generate openapi failed: %v", err)
	}

	if len(result.Warnings) == 0 {
		t.Fatalf("expected extractor warnings, got none")
	}

	found := false
	for _, warning := range result.Warnings {
		if warning.Code == "datasource_multiple_types" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected datasource_multiple_types warning, got %#v", result.Warnings)
	}

	if len(result.Body) == 0 {
		t.Fatalf("expected generated output body")
	}
}

func TestGenerateOpenAPIHandlesAnonymousSettingsTargetsAndAliasedSecureMaps(t *testing.T) {
	dir := writeFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1

require github.com/grafana/grafana-plugin-sdk-go v0.0.0

replace github.com/grafana/grafana-plugin-sdk-go => ./stubs/grafana-plugin-sdk-go
`,
		"stubs/grafana-plugin-sdk-go/go.mod": `
module github.com/grafana/grafana-plugin-sdk-go

go 1.26.1
`,
		"stubs/grafana-plugin-sdk-go/backend/backend.go": `
package backend

type DataSourceInstanceSettings struct {
	JSONData []byte
	DecryptedSecureJSONData map[string]string
}

type DataQuery struct {
	JSON []byte
}
`,
		"plugin/plugin.go": `
package plugin

import (
	"encoding/json"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type Query struct {
	QueryType string ` + "`json:\"queryType\"`" + `
	LogSearch struct {
		Query string ` + "`json:\"query\"`" + `
	} ` + "`json:\"logSearch\"`" + `
}

func LoadSettings(cfg backend.DataSourceInstanceSettings) error {
	jsonData := struct {
		Name string ` + "`json:\"name\"`" + `
		// runtime-only field should not show up
	}{}
	if err := json.Unmarshal(cfg.JSONData, &jsonData); err != nil {
		return err
	}

	secureSettings := cfg.DecryptedSecureJSONData
	_ = secureSettings["apiKey"]
	return nil
}

func LoadQuery(q backend.DataQuery) error {
	var query Query
	return json.Unmarshal(q.JSON, &query)
}
`,
	})

	result, err := GenerateOpenAPI(OpenAPIOptions{
		Dir: dir,
	})
	if err != nil {
		t.Fatalf("generate openapi failed: %v", err)
	}

	body := string(result.Body)
	if !strings.Contains(body, `"name"`) {
		t.Fatalf("expected config schema from anonymous target, got %s", body)
	}
	if strings.Contains(body, `"Inputs"`) {
		t.Fatalf("did not expect runtime-only untagged config fields, got %s", body)
	}
	if !strings.Contains(body, `"apiKey"`) {
		t.Fatalf("expected aliased secure key in output, got %s", body)
	}
	if strings.Contains(body, `"queries"`) {
		t.Fatalf("did not expect query definitions embedded in pluginspec output, got %s", body)
	}
}

func TestGenerateQueryTypesHandlesLooseQueries(t *testing.T) {
	dir := writeFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1

require github.com/grafana/grafana-plugin-sdk-go v0.0.0

replace github.com/grafana/grafana-plugin-sdk-go => ./stubs/grafana-plugin-sdk-go
`,
		"stubs/grafana-plugin-sdk-go/go.mod": `
module github.com/grafana/grafana-plugin-sdk-go

go 1.26.1
`,
		"stubs/grafana-plugin-sdk-go/backend/backend.go": `
package backend

type DataQuery struct {
	JSON []byte
}
`,
		"plugin/plugin.go": `
package plugin

import (
	"encoding/json"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type Query struct {
	QueryType string ` + "`json:\"queryType\"`" + `
	LogSearch struct {
		Query string ` + "`json:\"query\"`" + `
	} ` + "`json:\"logSearch\"`" + `
}

func LoadQuery(q backend.DataQuery) error {
	var query Query
	return json.Unmarshal(q.JSON, &query)
}
`,
	})

	result, err := GenerateQueryTypes(OpenAPIOptions{
		Dir: dir,
	})
	if err != nil {
		t.Fatalf("generate query types failed: %v", err)
	}

	var queries map[string]any
	if err := json.Unmarshal(result.Body, &queries); err != nil {
		t.Fatalf("unmarshal generated output failed: %v", err)
	}

	items := queries["items"].([]any)
	item := items[0].(map[string]any)
	spec := item["spec"].(map[string]any)
	schema := spec["schema"].(map[string]any)
	if _, ok := schema["required"]; ok {
		t.Fatalf("did not expect auto-discovered query schema required list, got %#v", schema["required"])
	}
}

func TestGenerateQueryTypesReturnsTypedEmptyListWhenNoQueriesAreFound(t *testing.T) {
	dir := writeFixtureModule(t, map[string]string{
		"go.mod": `
module fixture

go 1.26.1

require github.com/grafana/grafana-plugin-sdk-go v0.0.0

replace github.com/grafana/grafana-plugin-sdk-go => ./stubs/grafana-plugin-sdk-go
`,
		"stubs/grafana-plugin-sdk-go/go.mod": `
module github.com/grafana/grafana-plugin-sdk-go

go 1.26.1
`,
		"stubs/grafana-plugin-sdk-go/backend/backend.go": `
package backend

type DataSourceInstanceSettings struct {
	JSONData []byte
}
`,
		"plugin/plugin.go": `
package plugin

import "github.com/grafana/grafana-plugin-sdk-go/backend"

func LoadSettings(cfg backend.DataSourceInstanceSettings) error {
	return nil
}
`,
	})

	result, err := GenerateQueryTypes(OpenAPIOptions{
		Dir: dir,
	})
	if err != nil {
		t.Fatalf("generate query types failed: %v", err)
	}

	var queries map[string]any
	if err := json.Unmarshal(result.Body, &queries); err != nil {
		t.Fatalf("unmarshal generated output failed: %v", err)
	}

	if queries["kind"] != "QueryTypeDefinitionList" {
		t.Fatalf("expected typed empty list kind, got %#v", queries["kind"])
	}
	if queries["apiVersion"] != "datasource.grafana.app/v0alpha1" {
		t.Fatalf("expected typed empty list apiVersion, got %#v", queries["apiVersion"])
	}
	items, ok := queries["items"].([]any)
	if !ok {
		t.Fatalf("expected items array, got %#v", queries["items"])
	}
	if len(items) != 0 {
		t.Fatalf("expected empty query type items, got %#v", items)
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
