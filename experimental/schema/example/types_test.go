package example

import (
	"reflect"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/schema"
	"github.com/stretchr/testify/require"
)

func TestQueryTypeDefinitions(t *testing.T) {
	builder, err := schema.NewSchemaBuilder(schema.BuilderOptions{
		BasePackage: "github.com/grafana/grafana-plugin-sdk-go/experimental/schema/example",
		CodePath:    "./",
		// We need to identify the enum fields explicitly :(
		// *AND* have the +enum common for this to work
		Enums: []reflect.Type{
			reflect.TypeOf(ReducerSum),     // pick an example value (not the root)
			reflect.TypeOf(ReduceModeDrop), // pick an example value (not the root)
		},
	})
	require.NoError(t, err)
	err = builder.AddQueries(schema.QueryTypeInfo{
		Discriminators: schema.NewDiscriminators("queryType", QueryTypeMath),
		GoType:         reflect.TypeOf(&MathQuery{}),
		Examples: []schema.QueryExample{
			{
				Name: "constant addition",
				QueryPayload: MathQuery{
					Expression: "$A + 10",
				},
			},
			{
				Name: "math with two queries",
				QueryPayload: MathQuery{
					Expression: "$A - $B",
				},
			},
		},
	},
		schema.QueryTypeInfo{
			Discriminators: schema.NewDiscriminators("queryType", QueryTypeReduce),
			GoType:         reflect.TypeOf(&ReduceQuery{}),
			Examples: []schema.QueryExample{
				{
					Name: "get max value",
					QueryPayload: ReduceQuery{
						Expression: "$A",
						Reducer:    ReducerMax,
						Settings: ReduceSettings{
							Mode: ReduceModeDrop,
						},
					},
				},
			},
		},
		schema.QueryTypeInfo{
			Discriminators: schema.NewDiscriminators("queryType", QueryTypeResample),
			GoType:         reflect.TypeOf(&ResampleQuery{}),
		})
	require.NoError(t, err)

	builder.UpdateQueryDefinition(t, "types.json")
}
