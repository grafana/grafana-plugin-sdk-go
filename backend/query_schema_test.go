package backend

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

func TestQuerySchemaHandlerFunc(t *testing.T) {
	t.Run("QuerySchemaHandlerFunc should call the underlying function", func(t *testing.T) {
		called := false
		handler := QuerySchemaHandlerFunc(func(ctx context.Context, req *GetQuerySchemaRequest) (*GetQuerySchemaResponse, error) {
			called = true
			require.Equal(t, "metrics", req.QueryType)
			return &GetQuerySchemaResponse{
				Schema: json.RawMessage(`{"type": "object"}`),
				QueryTypes: []QueryTypeInfo{
					{Type: "metrics", Name: "Metrics", Description: "Query metrics"},
				},
			}, nil
		})

		resp, err := handler.GetQuerySchema(context.Background(), &GetQuerySchemaRequest{
			QueryType: "metrics",
		})

		require.NoError(t, err)
		require.True(t, called)
		require.NotNil(t, resp)
		require.Len(t, resp.QueryTypes, 1)
		require.Equal(t, "metrics", resp.QueryTypes[0].Type)
		require.Equal(t, "Metrics", resp.QueryTypes[0].Name)
		require.JSONEq(t, `{"type": "object"}`, string(resp.Schema))
	})
}

func TestQuerySchemaAdapter(t *testing.T) {
	t.Run("Adapter should convert between protobuf and SDK types", func(t *testing.T) {
		handler := QuerySchemaHandlerFunc(func(ctx context.Context, req *GetQuerySchemaRequest) (*GetQuerySchemaResponse, error) {
			require.Equal(t, "test-query-type", req.QueryType)
			require.Equal(t, "test-plugin", req.PluginContext.PluginID)
			return &GetQuerySchemaResponse{
				Schema: json.RawMessage(`{"$schema": "http://json-schema.org/draft-07/schema#", "type": "object"}`),
				QueryTypes: []QueryTypeInfo{
					{Type: "default", Name: "Default Query", Description: "The default query type"},
					{Type: "advanced", Name: "Advanced Query", Description: "Advanced query options"},
				},
			}, nil
		})

		adapter := newQuerySchemaSDKAdapter(handler)

		resp, err := adapter.GetQuerySchema(context.Background(), &pluginv2.GetQuerySchemaRequest{
			PluginContext: &pluginv2.PluginContext{
				PluginId: "test-plugin",
			},
			QueryType: "test-query-type",
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.JSONEq(t, `{"$schema": "http://json-schema.org/draft-07/schema#", "type": "object"}`, string(resp.Schema))
		require.Len(t, resp.QueryTypes, 2)
		require.Equal(t, "default", resp.QueryTypes[0].Type)
		require.Equal(t, "Default Query", resp.QueryTypes[0].Name)
		require.Equal(t, "The default query type", resp.QueryTypes[0].Description)
		require.Equal(t, "advanced", resp.QueryTypes[1].Type)
	})

	t.Run("Adapter should propagate errors", func(t *testing.T) {
		expectedErr := DownstreamError(context.DeadlineExceeded)
		handler := QuerySchemaHandlerFunc(func(ctx context.Context, req *GetQuerySchemaRequest) (*GetQuerySchemaResponse, error) {
			return nil, expectedErr
		})

		adapter := newQuerySchemaSDKAdapter(handler)

		resp, err := adapter.GetQuerySchema(context.Background(), &pluginv2.GetQuerySchemaRequest{
			PluginContext: &pluginv2.PluginContext{},
		})

		require.Error(t, err)
		require.Nil(t, resp)
	})
}
