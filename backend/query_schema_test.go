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

func TestMiddlewareHandlerGetQuerySchema(t *testing.T) {
	t.Run("Should delegate to underlying handler", func(t *testing.T) {
		called := false
		handler := &testHandlerWithQuerySchema{
			getQuerySchemaFunc: func(ctx context.Context, req *GetQuerySchemaRequest) (*GetQuerySchemaResponse, error) {
				called = true
				return &GetQuerySchemaResponse{
					Schema: json.RawMessage(`{"type": "object"}`),
				}, nil
			},
		}

		mh, err := HandlerFromMiddlewares(handler)
		require.NoError(t, err)

		resp, err := mh.GetQuerySchema(context.Background(), &GetQuerySchemaRequest{
			PluginContext: PluginContext{PluginID: "test"},
		})

		require.NoError(t, err)
		require.True(t, called)
		require.NotNil(t, resp)
		require.JSONEq(t, `{"type": "object"}`, string(resp.Schema))
	})

	t.Run("Should return error when request is nil", func(t *testing.T) {
		handler := &testHandlerWithQuerySchema{}

		mh, err := HandlerFromMiddlewares(handler)
		require.NoError(t, err)

		resp, err := mh.GetQuerySchema(context.Background(), nil)

		require.Error(t, err)
		require.Nil(t, resp)
	})
}

// testHandlerWithQuerySchema implements both Handler and QuerySchemaHandler
type testHandlerWithQuerySchema struct {
	getQuerySchemaFunc func(ctx context.Context, req *GetQuerySchemaRequest) (*GetQuerySchemaResponse, error)
}

func (h *testHandlerWithQuerySchema) QueryData(ctx context.Context, req *QueryDataRequest) (*QueryDataResponse, error) {
	return nil, nil
}

func (h *testHandlerWithQuerySchema) CallResource(ctx context.Context, req *CallResourceRequest, sender CallResourceResponseSender) error {
	return nil
}

func (h *testHandlerWithQuerySchema) CheckHealth(ctx context.Context, req *CheckHealthRequest) (*CheckHealthResult, error) {
	return nil, nil
}

func (h *testHandlerWithQuerySchema) CollectMetrics(ctx context.Context, req *CollectMetricsRequest) (*CollectMetricsResult, error) {
	return nil, nil
}

func (h *testHandlerWithQuerySchema) SubscribeStream(ctx context.Context, req *SubscribeStreamRequest) (*SubscribeStreamResponse, error) {
	return nil, nil
}

func (h *testHandlerWithQuerySchema) PublishStream(ctx context.Context, req *PublishStreamRequest) (*PublishStreamResponse, error) {
	return nil, nil
}

func (h *testHandlerWithQuerySchema) RunStream(ctx context.Context, req *RunStreamRequest, sender *StreamSender) error {
	return nil
}

func (h *testHandlerWithQuerySchema) ValidateAdmission(ctx context.Context, req *AdmissionRequest) (*ValidationResponse, error) {
	return nil, nil
}

func (h *testHandlerWithQuerySchema) MutateAdmission(ctx context.Context, req *AdmissionRequest) (*MutationResponse, error) {
	return nil, nil
}

func (h *testHandlerWithQuerySchema) ConvertObjects(ctx context.Context, req *ConversionRequest) (*ConversionResponse, error) {
	return nil, nil
}

func (h *testHandlerWithQuerySchema) GetQuerySchema(ctx context.Context, req *GetQuerySchemaRequest) (*GetQuerySchemaResponse, error) {
	if h.getQuerySchemaFunc != nil {
		return h.getQuerySchemaFunc(ctx, req)
	}
	return nil, nil
}
