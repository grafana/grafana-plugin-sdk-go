package expr

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/query/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryTypeDefinitions(t *testing.T) {
	builder, err := schema.NewBuilder(
		schema.BuilderOptions{
			BasePackage: "github.com/grafana/grafana-plugin-sdk-go/experimental/query/expr",
			CodePath:    "./",
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
	defs, err := builder.QueryTypeDefinitions()
	require.NoError(t, err)
	out, err := json.MarshalIndent(defs, "", "  ")
	require.NoError(t, err)

	update := false
	outfile := "types.json"
	body, err := os.ReadFile(outfile)
	if err == nil {
		if !assert.JSONEq(t, string(out), string(body)) {
			update = true
		}
	} else {
		update = true
	}
	if update {
		err = os.WriteFile(outfile, out, 0644)
		require.NoError(t, err, "error writing file")
	}

	// // The full schema
	// s, err := builder.FullQuerySchema()
	// require.NoError(t, err)

	// out, err := json.MarshalIndent(s, "", "  ")
	// require.NoError(t, err)

	// fmt.Printf("%s\n", out)
}
