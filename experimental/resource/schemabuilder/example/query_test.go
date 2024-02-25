package example

import (
	"reflect"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/resource"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/resource/schemabuilder"
	"github.com/stretchr/testify/require"
)

func TestQueryTypeDefinitions(t *testing.T) {
	builder, err := schemabuilder.NewSchemaBuilder(schemabuilder.BuilderOptions{
		BasePackage: "github.com/grafana/grafana-plugin-sdk-go/experimental/resource/schemabuilder/example",
		CodePath:    "./",
		// We need to identify the enum fields explicitly :(
		// *AND* have the +enum common for this to work
		Enums: []reflect.Type{
			reflect.TypeOf(ReducerSum),     // pick an example value (not the root)
			reflect.TypeOf(ReduceModeDrop), // pick an example value (not the root)
		},
	})
	require.NoError(t, err)
	err = builder.AddQueries(schemabuilder.QueryTypeInfo{
		Discriminators: resource.NewDiscriminators("queryType", QueryTypeMath),
		GoType:         reflect.TypeOf(&MathQuery{}),
		Examples: []resource.QueryExample{
			{
				Name: "constant addition",
				SaveModel: MathQuery{
					Expression: "$A + 10",
				},
			},
			{
				Name: "math with two queries",
				SaveModel: MathQuery{
					Expression: "$A - $B",
				},
			},
		},
	},
		schemabuilder.QueryTypeInfo{
			Discriminators: resource.NewDiscriminators("queryType", QueryTypeReduce),
			GoType:         reflect.TypeOf(&ReduceQuery{}),
			Examples: []resource.QueryExample{
				{
					Name: "get max value",
					SaveModel: ReduceQuery{
						Expression: "$A",
						Reducer:    ReducerMax,
						Settings: ReduceSettings{
							Mode: ReduceModeDrop,
						},
					},
				},
			},
		},
		schemabuilder.QueryTypeInfo{
			Discriminators: resource.NewDiscriminators("queryType", QueryTypeResample),
			GoType:         reflect.TypeOf(&ResampleQuery{}),
		})
	require.NoError(t, err)

	_ = builder.UpdateQueryDefinition(t, "./")
}
