package schemabuilder

import (
	"os"
	"reflect"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/resource"
	"github.com/invopop/jsonschema"
	"github.com/stretchr/testify/require"
)

func TestWriteQuerySchema(t *testing.T) {
	builder, err := NewSchemaBuilder(BuilderOptions{
		PluginID: []string{"dummy"},
		ScanCode: []CodePaths{
			{
				BasePackage: "github.com/grafana/grafana-plugin-sdk-go/experimental/resource/dotdothack",
				CodePath:    "../",
			},
			{
				BasePackage: "github.com/grafana/grafana-plugin-sdk-go/data",
				CodePath:    "../../../data",
			},
		},
		Enums: []reflect.Type{
			reflect.TypeOf(data.FrameTypeLogLines),
		},
	})
	require.NoError(t, err)

	query := builder.reflector.Reflect(&resource.CommonQueryProperties{})
	updateEnumDescriptions(query)
	query.ID = ""
	query.Version = draft04 // used by kube-openapi
	query.Description = "Generic query properties"
	query.AdditionalProperties = jsonschema.TrueSchema

	// // Hide this old property
	query.Properties.Delete("datasourceId")

	outfile := "../query.schema.json"
	old, _ := os.ReadFile(outfile)
	maybeUpdateFile(t, outfile, query, old)

	// Make sure the embedded schema is loadable
	schema, err := resource.GenericQuerySchema()
	require.NoError(t, err)
	require.Equal(t, 8, len(schema.Properties))

	// Add schema for query type definition
	query = builder.reflector.Reflect(&resource.QueryTypeDefinitionSpec{})
	updateEnumDescriptions(query)
	query.ID = ""
	query.Version = draft04 // used by kube-openapi
	outfile = "../query.definition.schema.json"
	old, _ = os.ReadFile(outfile)
	maybeUpdateFile(t, outfile, query, old)

	def := resource.QueryTypeDefinitionSpec{}.OpenAPIDefinition()
	require.Equal(t, query.Properties.Len(), len(def.Schema.Properties))
}
