package fromschema_test

import (
	"context"
	"testing"

	sdkapi "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/datasource/v0alpha1"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp/fromschema"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/pluginschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterQueryExamples_publishesResourceAndAttachesExamples(t *testing.T) {
	schema := &pluginschema.PluginSchema{
		QueryTypes: &sdkapi.QueryTypeDefinitionList{
			Items: []sdkapi.QueryTypeDefinition{{
				ObjectMeta: sdkapi.ObjectMeta{Name: "Pull_Requests"},
				Spec: sdkapi.QueryTypeDefinitionSpec{
					Discriminators: []sdkapi.DiscriminatorFieldValue{{Field: "queryType", Value: "Pull_Requests"}},
					Schema:         jsonSchema(t, `{"type":"object"}`),
				},
			}},
		},
		QueryExamples: &sdkapi.QueryExamples{Examples: []sdkapi.QueryExample{
			{Name: "Simple", QueryType: "Pull_Requests", SaveModel: sdkapi.AsUnstructured(map[string]any{"owner": "grafana"})},
		}},
	}
	s := mcp.NewServer(mcp.ServerOpts{Name: "x", Version: "0"})
	fromschema.RegisterQueryTools(s, schema)
	fromschema.RegisterQueryExamples(s, schema)

	// resource registered
	resources := s.Resources()
	require.Len(t, resources, 1)
	assert.Equal(t, "examples://query", resources[0].URI)
	body, _, err := resources[0].Reader(context.Background())
	require.NoError(t, err)
	assert.Contains(t, string(body), "Pull_Requests")

	// example attached to the matching tool
	for _, t2 := range s.Tools() {
		if t2.Name == "query_Pull_Requests" {
			require.Len(t, t2.Examples, 1)
			return
		}
	}
	t.Fatalf("query_Pull_Requests tool not found")
}
