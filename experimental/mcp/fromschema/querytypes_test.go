package fromschema_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	sdkapi "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/datasource/v0alpha1"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/mcp/fromschema"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/pluginschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type queryOnly struct{ lastReq *backend.QueryDataRequest }

func (q *queryOnly) QueryData(_ context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	q.lastReq = req
	return &backend.QueryDataResponse{}, nil
}

// jsonSchema is a helper that builds a sdkapi.JSONSchema from a JSON literal.
func jsonSchema(t *testing.T, body string) sdkapi.JSONSchema {
	t.Helper()
	var js sdkapi.JSONSchema
	require.NoError(t, json.Unmarshal([]byte(body), &js))
	return js
}

func TestRegisterQueryTools_addsOneToolPerQueryType(t *testing.T) {
	schema := &pluginschema.PluginSchema{
		QueryTypes: &sdkapi.QueryTypeDefinitionList{
			Items: []sdkapi.QueryTypeDefinition{
				{
					ObjectMeta: sdkapi.ObjectMeta{Name: "Pull_Requests"},
					Spec: sdkapi.QueryTypeDefinitionSpec{
						Discriminators: []sdkapi.DiscriminatorFieldValue{{Field: "queryType", Value: "Pull_Requests"}},
						Schema:         jsonSchema(t, `{"type":"object"}`),
					},
				},
				{
					ObjectMeta: sdkapi.ObjectMeta{Name: "Issues"},
					Spec: sdkapi.QueryTypeDefinitionSpec{
						Discriminators: []sdkapi.DiscriminatorFieldValue{{Field: "queryType", Value: "Issues"}},
						Schema:         jsonSchema(t, `{"type":"object"}`),
					},
				},
			},
		},
	}

	s := mcp.NewServer(mcp.ServerOpts{Name: "x", Version: "0"})
	q := &queryOnly{}
	s.BindQueryDataHandler(q)
	fromschema.RegisterQueryTools(s, schema)

	tools := s.Tools()
	require.Len(t, tools, 2)
	names := []string{tools[0].Name, tools[1].Name}
	assert.Contains(t, names, "query_Pull_Requests")
	assert.Contains(t, names, "query_Issues")
}

func TestRegisterQueryTools_handlerCallsBoundQueryData(t *testing.T) {
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
	}
	s := mcp.NewServer(mcp.ServerOpts{Name: "x", Version: "0"})
	q := &queryOnly{}
	s.BindQueryDataHandler(q)
	fromschema.RegisterQueryTools(s, schema)

	tool := s.Tools()[0]
	_, err := tool.Handler(context.Background(), map[string]any{"owner": "grafana"})
	require.NoError(t, err)
	require.NotNil(t, q.lastReq)
	require.Len(t, q.lastReq.Queries, 1)
	assert.Equal(t, "Pull_Requests", q.lastReq.Queries[0].QueryType)
}

func TestRegisterQueryTools_skipsWhenSchemaHasNoQueryTypes(t *testing.T) {
	s := mcp.NewServer(mcp.ServerOpts{Name: "x", Version: "0"})
	fromschema.RegisterQueryTools(s, &pluginschema.PluginSchema{})
	assert.Empty(t, s.Tools())
}
