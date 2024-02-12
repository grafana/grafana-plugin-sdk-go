package expr

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/query/schema"
	"github.com/stretchr/testify/require"
)

func TestQueryTypeDefinitions(t *testing.T) {
	builder, err := schema.NewBuilder(
		schema.BuilderOptions{
			BasePackage: "github.com/grafana/grafana-plugin-sdk-go/experimental/query/expr",
			CodePath:    "./",
			PluginIDs:   []string{"expr"},
		},
		schema.QueryTypeInfo{
			QueryType: string(QueryTypeMath),
			GoType:    reflect.TypeOf(&MathQuery{}),
		},
		schema.QueryTypeInfo{
			QueryType: string(QueryTypeReduce),
			GoType:    reflect.TypeOf(&ReduceQuery{}),
		},
		schema.QueryTypeInfo{
			QueryType: string(QueryTypeResample),
			GoType:    reflect.TypeOf(&ResampleQuery{}),
		})
	require.NoError(t, err)

	// The full schema
	s, err := builder.GetFullQuerySchema()
	require.NoError(t, err)

	out, err := json.MarshalIndent(s, "", "  ")
	require.NoError(t, err)

	fmt.Printf("%s\n", out)
}
