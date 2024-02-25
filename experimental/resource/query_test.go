package resource

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/invopop/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommonQueryProperties(t *testing.T) {
	r := new(jsonschema.Reflector)
	r.DoNotReference = true
	err := r.AddGoComments("github.com/grafana/grafana-plugin-sdk-go/experimental/resource", "./")
	require.NoError(t, err)

	query := r.Reflect(&CommonQueryProperties{})
	query.ID = ""
	query.Version = "https://json-schema.org/draft-04/schema" // used by kube-openapi
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

	// Make sure the embedded schema is loadable
	schema, err := CommonQueryPropertiesSchema()
	require.NoError(t, err)
	require.Equal(t, 8, len(schema.Properties))
}
