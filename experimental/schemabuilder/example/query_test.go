package example

import (
	"reflect"
	"testing"

	data "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/data/v0alpha1"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/schemabuilder"
	"github.com/stretchr/testify/require"
)

func TestQueryTypeDefinitions(t *testing.T) {
	builder, err := schemabuilder.NewSchemaBuilder(schemabuilder.BuilderOptions{
		PluginID: []string{"__expr__"},
		ScanCode: []schemabuilder.CodePaths{{
			BasePackage: "github.com/grafana/grafana-plugin-sdk-go/experimental/schemabuilder/example",
			CodePath:    "./",
		}},
		Enums: []reflect.Type{
			reflect.TypeOf(ReducerSum),     // pick an example value (not the root)
			reflect.TypeOf(ReduceModeDrop), // pick an example value (not the root)
		},
	})
	require.NoError(t, err)
	err = builder.AddQueries(schemabuilder.QueryTypeInfo{
		Discriminators: data.NewDiscriminators("queryType", QueryTypeMath),
		GoType:         reflect.TypeOf(&MathQuery{}),
		Examples: []data.QueryExample{
			{
				Name: "constant addition",
				SaveModel: data.AsUnstructured(MathQuery{
					Expression: "$A + 11",
				}),
			},
			{
				Name: "math with two queries",
				SaveModel: data.AsUnstructured(MathQuery{
					Expression: "$A - $B",
				}),
			},
		},
	},
		schemabuilder.QueryTypeInfo{
			Discriminators: data.NewDiscriminators("queryType", QueryTypeReduce),
			GoType:         reflect.TypeOf(&ReduceQuery{}),
			Examples: []data.QueryExample{
				{
					Name: "get max value",
					SaveModel: data.AsUnstructured(ReduceQuery{
						Expression: "$A",
						Reducer:    ReducerMax,
						Settings: ReduceSettings{
							Mode: ReduceModeDrop,
						},
					}),
				},
			},
		},
		schemabuilder.QueryTypeInfo{
			Discriminators: data.NewDiscriminators("queryType", QueryTypeResample),
			GoType:         reflect.TypeOf(&ResampleQuery{}),
			Examples:       []data.QueryExample{},
		})
	require.NoError(t, err)

	_ = builder.UpdateQueryDefinition(t, "./")
}
