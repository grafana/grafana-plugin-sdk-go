package datasourceschema

import (
	"encoding/json"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestGenerateOpenAPIPreservesLoadOptions(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
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
	require.NoError(t, err, "generate openapi failed")
	require.Contains(t, string(result.Body), `"name"`, "expected generated schema to include tagged settings fields")
	require.Contains(t, string(result.Body), `"settings"`, "expected pluginspec settings wrapper")
}

func TestGenerateOpenAPIReturnsWarningsForAmbiguousSettingsTargets(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
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
	require.NoError(t, err, "generate openapi failed")
	require.NotEmpty(t, result.Warnings, "expected extractor warnings")

	found := false
	for _, warning := range result.Warnings {
		if warning.Code == "datasource_multiple_types" {
			found = true
			break
		}
	}
	require.True(t, found, "expected datasource_multiple_types warning, got %#v", result.Warnings)
	require.NotEmpty(t, result.Body, "expected generated output body")
}

func TestGenerateOpenAPIHandlesAnonymousSettingsTargetsAndAliasedSecureMaps(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
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
	require.NoError(t, err, "generate openapi failed")

	body := string(result.Body)
	require.Contains(t, body, `"name"`, "expected config schema from anonymous target")
	require.NotContains(t, body, `"Inputs"`, "did not expect runtime-only untagged config fields")
	require.Contains(t, body, `"apiKey"`, "expected aliased secure key in output")
	require.NotContains(t, body, `"queries"`, "did not expect query definitions embedded in pluginspec output")
}

func TestGenerateQueryTypesHandlesLooseQueries(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
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
	require.NoError(t, err, "generate query types failed")

	var queries map[string]any
	require.NoError(t, json.Unmarshal(result.Body, &queries), "unmarshal generated output failed")

	items, ok := queries["items"].([]any)
	require.True(t, ok, "expected items array, got %#v", queries["items"])
	require.NotEmpty(t, items, "expected at least one query item")
	item, ok := items[0].(map[string]any)
	require.True(t, ok, "expected query item object, got %#v", items[0])
	spec, ok := item["spec"].(map[string]any)
	require.True(t, ok, "expected query spec object, got %#v", item["spec"])
	schema, ok := spec["schema"].(map[string]any)
	require.True(t, ok, "expected query schema object, got %#v", spec["schema"])
	require.NotContains(t, schema, "required", "did not expect auto-discovered query schema required list")
}

func TestGenerateQueryTypesReturnsTypedEmptyListWhenNoQueriesAreFound(t *testing.T) {
	dir := testutil.WriteFixtureModule(t, map[string]string{
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
	require.NoError(t, err, "generate query types failed")

	var queries map[string]any
	require.NoError(t, json.Unmarshal(result.Body, &queries), "unmarshal generated output failed")

	require.Equal(t, "QueryTypeDefinitionList", queries["kind"], "expected typed empty list kind")
	require.Equal(t, "datasource.grafana.app/v0alpha1", queries["apiVersion"], "expected typed empty list apiVersion")
	items, ok := queries["items"].([]any)
	require.True(t, ok, "expected items array, got %#v", queries["items"])
	require.Empty(t, items, "expected empty query type items")
}
