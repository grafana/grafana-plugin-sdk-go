package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/invopop/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommonSupport(t *testing.T) {
	r := new(jsonschema.Reflector)
	r.DoNotReference = true
	err := r.AddGoComments("github.com/grafana/grafana-plugin-sdk-go/experimental/spec", "./")
	require.NoError(t, err)

	query := r.Reflect(&CommonQueryProperties{})
	query.Version = draft04 // used by kube-openapi
	query.Description = "Query properties shared by all data sources"

	// Write the map of values ignored by the common parser
	fmt.Printf("var commonKeys = map[string]bool{\n")
	for pair := query.Properties.Oldest(); pair != nil; pair = pair.Next() {
		fmt.Printf("  \"%s\": true,\n", pair.Key)
	}
	fmt.Printf("}\n")

	// // Hide this old property
	query.Properties.Delete("datasourceId")
	out, err := json.MarshalIndent(query, "", "  ")
	require.NoError(t, err)

	update := false
	outfile := "query.schema.json"
	body, err := os.ReadFile(outfile)
	if err == nil {
		if !assert.JSONEq(t, string(out), string(body)) {
			update = true
		}
	} else {
		update = true
	}
	if update {
		err = os.WriteFile(outfile, out, 0600)
		require.NoError(t, err, "error writing file")
	}
}
