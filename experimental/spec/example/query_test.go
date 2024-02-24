package example

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/spec"
	"github.com/stretchr/testify/require"
)

func TestQueryTypeDefinitions(t *testing.T) {
	builder, err := spec.NewSchemaBuilder(spec.BuilderOptions{
		BasePackage: "github.com/grafana/grafana-plugin-sdk-go/experimental/spec/example",
		CodePath:    "./",
		// We need to identify the enum fields explicitly :(
		// *AND* have the +enum common for this to work
		Enums: []reflect.Type{
			reflect.TypeOf(ReducerSum),     // pick an example value (not the root)
			reflect.TypeOf(ReduceModeDrop), // pick an example value (not the root)
		},
	})
	require.NoError(t, err)
	err = builder.AddQueries(spec.QueryTypeInfo{
		Discriminators: spec.NewDiscriminators("queryType", QueryTypeMath),
		GoType:         reflect.TypeOf(&MathQuery{}),
		Examples: []spec.QueryExample{
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
		spec.QueryTypeInfo{
			Discriminators: spec.NewDiscriminators("queryType", QueryTypeReduce),
			GoType:         reflect.TypeOf(&ReduceQuery{}),
			Examples: []spec.QueryExample{
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
		spec.QueryTypeInfo{
			Discriminators: spec.NewDiscriminators("queryType", QueryTypeResample),
			GoType:         reflect.TypeOf(&ResampleQuery{}),
		})
	require.NoError(t, err)

	defs := builder.UpdateQueryDefinition(t, "./")

	queries, err := spec.GetExampleQueries(defs)
	require.NoError(t, err)

	out, err := json.MarshalIndent(queries, "", "  ")
	require.NoError(t, err)

	fmt.Printf("%s", string(out))
}
