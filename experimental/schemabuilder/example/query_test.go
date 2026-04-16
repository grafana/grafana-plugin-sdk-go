package example

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	data "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/datasource/v0alpha1"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/schemabuilder"
)

func TestQueryTypeDefinitions(t *testing.T) {
	builder, err := schemabuilder.NewSchemaBuilder(schemabuilder.BuilderOptions{
		PluginID: []string{"__expr__"},
		ScanCode: []schemabuilder.CodePaths{{
			BasePackage: "github.com/grafana/grafana-plugin-sdk-go/experimental/schemabuilder/example",
			CodePath:    "./",
		}},
		Enums: []reflect.Type{
			reflect.TypeFor[ReducerID](),
			reflect.TypeFor[ReduceMode](),
		},
	})
	require.NoError(t, err)
	err = builder.AddQueries([]schemabuilder.QueryTypeInfo{{
		Discriminators: data.NewDiscriminators("queryType", QueryTypeMath),
		GoType:         reflect.TypeFor[*MathQuery](),
	}, {
		Discriminators: data.NewDiscriminators("queryType", QueryTypeReduce),
		GoType:         reflect.TypeFor[*ReduceQuery](),
	}, {
		Discriminators: data.NewDiscriminators("queryType", QueryTypeResample),
		GoType:         reflect.TypeFor[*ResampleQuery](),
	}})
	require.NoError(t, err)

	err = builder.AddExamples([]data.QueryExample{{
		Name:      "constant addition",
		QueryType: string(QueryTypeMath),
		SaveModel: data.AsUnstructured(MathQuery{
			Expression: "$A + 11",
		}),
	}, {
		Name:      "math with two queries",
		QueryType: string(QueryTypeMath),
		SaveModel: data.AsUnstructured(MathQuery{
			Expression: "$A - $B",
		}),
	}, {
		Name:      "get max value",
		QueryType: string(QueryTypeReduce),
		SaveModel: data.AsUnstructured(ReduceQuery{
			Expression: "$A",
			Reducer:    ReducerMax,
			Settings: ReduceSettings{
				Mode: ReduceModeDrop,
			},
		}),
	}})
	require.NoError(t, err)

	_ = builder.UpdateQueryTypes(t, "v0alpha1", "../testdata")
}
