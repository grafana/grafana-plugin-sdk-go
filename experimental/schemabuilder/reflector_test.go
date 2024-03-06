package schemabuilder

import (
	"os"
	"reflect"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	apisdata "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/data/v0alpha1"
	"github.com/invopop/jsonschema"
	"github.com/stretchr/testify/require"
)

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
	query.ID = ""
	query.Version = draft04 // used by kube-openapi
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

	// Add schema for query type definition
	query = builder.reflector.Reflect(&apisdata.QueryTypeDefinitionSpec{})
	updateEnumDescriptions(query)
	query.ID = ""
	query.Version = draft04 // used by kube-openapi
	outfile = "../apis/data/v0alpha1/query.definition.schema.json"
	old, _ = os.ReadFile(outfile)
	maybeUpdateFile(t, outfile, query, old)

	def := apisdata.GetOpenAPIDefinitions(nil)["github.com/grafana/grafana-plugin-sdk-go/experimental/apis/data/v0alpha1.QueryTypeDefinitionSpec"]
	require.Equal(t, query.Properties.Len(), len(def.Schema.Properties))
}
