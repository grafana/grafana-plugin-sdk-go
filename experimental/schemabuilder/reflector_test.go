package schemabuilder

import (
	_ "embed"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/go-openapi/loads"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag/loading"
	"github.com/go-openapi/validate"
	"github.com/invopop/jsonschema"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	apisdata "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/data/v0alpha1"
)

//go:embed testdata/draft-04-schema.json
var draft04SchemaJSON []byte

func TestWriteQuerySchema(t *testing.T) {
	builder, err := NewSchemaBuilder(BuilderOptions{
		PluginID: []string{"dummy"},
		ScanCode: []CodePaths{
			{
				BasePackage: "github.com/grafana/grafana-plugin-sdk-go/experimental/apis",
				CodePath:    "../apis/data/v0alpha1",
			},
			{
				BasePackage: "github.com/grafana/grafana-plugin-sdk-go/data",
				CodePath:    "../../data",
			},
		},
		Enums: []reflect.Type{
			reflect.TypeOf(data.FrameTypeLogLines),
		},
	})
	require.NoError(t, err)

	query := builder.reflector.Reflect(&apisdata.CommonQueryProperties{})
	updateEnumDescriptions(query)

	query.Version = "" // $schema is not allowed in openapi v2's SchemaObject
	query.ID = ""
	query.Description = "Generic query properties"
	query.AdditionalProperties = jsonschema.TrueSchema

	// // Hide this old property
	query.Properties.Delete("datasourceId")

	outfile := "../apis/data/v0alpha1/query.schema.json"
	old, _ := os.ReadFile(outfile)
	maybeUpdateFile(t, outfile, query, old)

	// Make sure the embedded schema is loadable
	schema, err := apisdata.DataQuerySchema()
	require.NoError(t, err)
	require.Equal(t, 8, len(schema.Properties))

	bytes, err := os.ReadFile(outfile)
	require.NoError(t, err)
	validateOpenAPIv2Schema(t, bytes, outfile)

	// Add schema for query type definition
	query = builder.reflector.Reflect(&apisdata.QueryTypeDefinitionSpec{})
	updateEnumDescriptions(query)
	query.ID = ""
	query.Version = "" // $schema is not allowed in openapi v2's SchemaObject
	outfile = "../apis/data/v0alpha1/query.definition.schema.json"
	old, _ = os.ReadFile(outfile)
	maybeUpdateFile(t, outfile, query, old)

	bytes, err = os.ReadFile(outfile)
	require.NoError(t, err)
	validateOpenAPIv2Schema(t, bytes, outfile)

	def := apisdata.GetOpenAPIDefinitions(nil)["github.com/grafana/grafana-plugin-sdk-go/experimental/apis/data/v0alpha1.QueryTypeDefinitionSpec"]
	require.Equal(t, query.Properties.Len(), len(def.Schema.Properties))
}

func validateOpenAPIv2Schema(t *testing.T, data []byte, file string) {
	t.Helper()
	// --- Stage 1: Check for disallowed top-level keys ---
	// https://github.com/go-openapi/spec/blob/0201d0c/schema.go#L622 json.Unmarshal on `spec.Schema` gets rid of $schema - so need to unmarshall into a generic map
	var genericMap map[string]interface{}
	if err := json.Unmarshal(data, &genericMap); err != nil {
		require.NoError(t, err, file)
	}

	// https://github.com/OAI/OpenAPI-Specification/blob/main/versions/2.0.md#schemaObject doesn't contain $schema
	if _, found := genericMap["$schema"]; found {
		require.Fail(t, "$schema not allowed", file)
	}

	// --- Stage 2: Validate against OpenAPI v2 structure using go-openapi ---

	// 2a. Try unmarshal the input data into a spec.Schema.
	var schema spec.Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		require.NoError(t, err, file)
	}

	// 2b. Create minimal valid swagger spec & marshal it to bytes.
	swaggerBytes, err := json.Marshal(&spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Swagger: "2.0",
			Info: &spec.Info{
				InfoProps: spec.InfoProps{
					Title:   "example", // Placeholder, required
					Version: "1.0",     // Placeholder, required
				},
			},
			Paths: &spec.Paths{ // Required, can be empty
				Paths: map[string]spec.PathItem{},
			},
			Definitions: spec.Definitions{
				"SchemaToValidate": schema,
			},
		},
	})
	require.NoError(t, err, file)

	localDocLoader := func(path string, opts ...loading.Option) (json.RawMessage, error) {
		// Use vendored draft-04 schema to avoid network calls
		if path == "https://json-schema.org/draft-04/schema#" || path == "https://json-schema.org/draft-04/schema" {
			return draft04SchemaJSON, nil
		}

		// For other paths (local files or HTTP URLs), use the default loader
		return loading.JSONDoc(path, opts...)
	}

	tmpFile, err := os.CreateTemp("", "swagger-*.json")
	require.NoError(t, err, file)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	_, err = tmpFile.Write(swaggerBytes)
	require.NoError(t, err, file)
	err = tmpFile.Close()
	require.NoError(t, err, file)

	doc, err := loads.Spec(tmpFile.Name(), loads.WithDocLoader(localDocLoader))
	require.NoError(t, err, file)

	err = validate.Spec(doc, strfmt.Default)
	require.NoError(t, err, file)
}
