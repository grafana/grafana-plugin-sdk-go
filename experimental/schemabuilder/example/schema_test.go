package example

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	data "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/datasource/v0alpha1"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/schemabuilder"
)

func TestPluginSchema(t *testing.T) {
	schema, err := schemabuilder.NewSchemaBuilder(schemabuilder.BuilderOptions{
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
	err = schema.AddQueries([]schemabuilder.QueryTypeInfo{{
		Discriminators: data.NewDiscriminators("queryType", QueryTypeMath),
		GoType:         reflect.TypeFor[*MathQuery](),
		Examples: []data.QueryExample{{
			Name: "constant addition",
			SaveModel: data.AsUnstructured(MathQuery{
				Expression: "$A + 11",
			}),
		}, {
			Name: "math with two queries",
			SaveModel: data.AsUnstructured(MathQuery{
				Expression: "$A - $B",
			}),
		}},
	}, {
		Discriminators: data.NewDiscriminators("queryType", QueryTypeReduce),
		GoType:         reflect.TypeFor[*ReduceQuery](),
		Examples: []data.QueryExample{{
			Name: "get max value",
			SaveModel: data.AsUnstructured(ReduceQuery{
				Expression: "$A",
				Reducer:    ReducerMax,
				Settings: ReduceSettings{
					Mode: ReduceModeDrop,
				},
			}),
		}},
	}, {
		Discriminators: data.NewDiscriminators("queryType", QueryTypeResample),
		GoType:         reflect.TypeFor[*ResampleQuery](),
	}})
	require.NoError(t, err)

	tmp := newSchema()
	err = schema.ConfigureSettings(tmp.SettingsSchema, tmp.SettingsExamples)
	require.NoError(t, err)

	err = schema.SetRoutes(tmp.Routes)
	require.NoError(t, err)

	schema.UpdateProviderFiles(t, "v0alpha1", "../testdata")
}
