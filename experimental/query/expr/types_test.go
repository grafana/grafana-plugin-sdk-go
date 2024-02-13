package expr

import (
	"reflect"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/query"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/query/schema"
	"github.com/stretchr/testify/require"
)

func TestQueryTypeDefinitions(t *testing.T) {
	builder, err := schema.NewBuilder(t,
		schema.BuilderOptions{
			BasePackage: "github.com/grafana/grafana-plugin-sdk-go/experimental/query/expr",
			CodePath:    "./",
			// We need to identify the enum fields explicitly :(
			// *AND* have the +enum common for this to work
			Enums: []reflect.Type{
				reflect.TypeOf(ReducerSum),     // pick an example value (not the root)
				reflect.TypeOf(ReduceModeDrop), // pick an example value (not the root)
			},
		},
		schema.QueryTypeInfo{
			QueryType: string(QueryTypeMath),
			GoType:    reflect.TypeOf(&MathQuery{}),
			Examples: []query.QueryExample{
				{
					Name: "constant addition",
					Query: MathQuery{
						Expression: "$A + 10",
					},
				},
				{
					Name: "math with two queries",
					Query: MathQuery{
						Expression: "$A - $B",
					},
				},
			},
		},
		schema.QueryTypeInfo{
			QueryType: string(QueryTypeReduce),
			GoType:    reflect.TypeOf(&ReduceQuery{}),
			Examples: []query.QueryExample{
				{
					Name: "get max value",
					Query: ReduceQuery{
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
			QueryType: string(QueryTypeResample),
			GoType:    reflect.TypeOf(&ResampleQuery{}),
		})
	require.NoError(t, err)

	_ = builder.Write("types.json")
}
