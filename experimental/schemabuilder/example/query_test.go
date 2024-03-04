package example

import (
	"reflect"
	"testing"

	sdkapi "github.com/grafana/grafana-plugin-sdk-go/apis/sdkapi/v0alpha1"
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
		Discriminators: sdkapi.NewDiscriminators("queryType", QueryTypeMath),
		GoType:         reflect.TypeOf(&MathQuery{}),
		Examples: []sdkapi.QueryExample{
			{
				Name: "constant addition",
				SaveModel: sdkapi.AsUnstructured(MathQuery{
					Expression: "$A + 11",
				}),
			},
			{
				Name: "math with two queries",
				SaveModel: sdkapi.AsUnstructured(MathQuery{
					Expression: "$A - $B",
				}),
			},
		},
	},
		schemabuilder.QueryTypeInfo{
			Discriminators: sdkapi.NewDiscriminators("queryType", QueryTypeReduce),
			GoType:         reflect.TypeOf(&ReduceQuery{}),
			Examples: []sdkapi.QueryExample{
				{
					Name: "get max value",
					SaveModel: sdkapi.AsUnstructured(ReduceQuery{
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
			Discriminators: sdkapi.NewDiscriminators("queryType", QueryTypeResample),
			GoType:         reflect.TypeOf(&ResampleQuery{}),
			Examples:       []sdkapi.QueryExample{},
		})
	require.NoError(t, err)

	_ = builder.UpdateQueryDefinition(t, "./")
}
