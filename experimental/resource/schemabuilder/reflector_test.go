package schemabuilder

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/resource"
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
	query.Version = "https://json-schema.org/draft-04/schema" // used by kube-openapi
	query.Description = "Query properties shared by all data sources"

	// Write the map of values ignored by the common parser
	fmt.Printf("var commonKeys = map[string]bool{\n")
	for pair := query.Properties.Oldest(); pair != nil; pair = pair.Next() {
		fmt.Printf("  \"%s\": true,\n", pair.Key)
	}
	fmt.Printf("}\n")

	// // Hide this old property
	query.Properties.Delete("datasourceId")

	outfile := "../query.schema.json"
	old, _ := os.ReadFile(outfile)
	maybeUpdateFile(t, outfile, query, old)

	// Make sure the embedded schema is loadable
	schema, err := resource.CommonQueryPropertiesSchema()
	require.NoError(t, err)
	require.Equal(t, 8, len(schema.Properties))
}
