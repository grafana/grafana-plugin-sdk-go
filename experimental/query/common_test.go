package query

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/invopop/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommonSupport(t *testing.T) {
	r := new(jsonschema.Reflector)
	err := r.AddGoComments("github.com/grafana/grafana-plugin-sdk-go/experimental/query", "./")
	require.NoError(t, err)

	query := r.Reflect(&CommonQueryProperties{})
	common, ok := query.Definitions["CommonQueryProperties"]
	require.True(t, ok)

	// Hide this old property
	common.Properties.Delete("datasourceId")
	out, err := json.MarshalIndent(query, "", "  ")
	require.NoError(t, err)

	update := false
	outfile := "common.jsonschema"
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
}
