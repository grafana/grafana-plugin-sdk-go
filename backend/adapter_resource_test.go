package backend

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	"github.com/stretchr/testify/require"
)

func TestCallResource(t *testing.T) {
	t.Run("When call resource handler not set should return http status not implemented", func(t *testing.T) {
		adapter := &sdkAdapter{}
		res, err := adapter.CallResource(context.Background(), &pluginv2.CallResource_Request{})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, http.StatusNotImplemented, int(res.Code))
		require.Empty(t, res.Headers)
		require.Empty(t, res.Body)
	})

	t.Run("When call resource handler set should provide expected request and return expected response", func(t *testing.T) {
		data := map[string]interface{}{
			"message": "hello",
		}
		body, err := json.Marshal(&data)
		require.NoError(t, err)
		handler := &testCallResourceHandler{
			responseStatus: http.StatusOK,
			responseHeaders: map[string][]string{
				"X-Header-Out-1": []string{"D", "E"},
				"X-Header-Out-2": []string{"F"},
			},
			responseBody: body,
		}
		adapter := &sdkAdapter{
			CallResourceHandler: handler,
		}
		res, err := adapter.CallResource(context.Background(), &pluginv2.CallResource_Request{
			Config: &pluginv2.PluginConfig{
				OrgId:      2,
				PluginId:   "my-plugin",
				PluginType: "my-type",
			},
			Path:   "some/path",
			Method: http.MethodGet,
			Url:    "plugins/test-plugin/resources/some/path?test=1",
			Headers: map[string]*pluginv2.CallResource_StringList{
				"X-Header-In-1": &pluginv2.CallResource_StringList{Values: []string{"A", "B"}},
				"X-Header-In-2": &pluginv2.CallResource_StringList{Values: []string{"C"}},
			},
			Body: body,
		})

		require.NoError(t, err)
		// request
		require.NotNil(t, handler.actualReq)
		require.Equal(t, "some/path", handler.actualReq.Path)
		require.Equal(t, http.MethodGet, handler.actualReq.Method)
		require.Equal(t, "plugins/test-plugin/resources/some/path?test=1", handler.actualReq.URL)
		require.Contains(t, handler.actualReq.Headers, "X-Header-In-1")
		require.Equal(t, []string{"A", "B"}, handler.actualReq.Headers["X-Header-In-1"])
		require.Contains(t, handler.actualReq.Headers, "X-Header-In-2")
		require.Equal(t, []string{"C"}, handler.actualReq.Headers["X-Header-In-2"])
		var actualRequestData map[string]interface{}
		err = json.Unmarshal(res.Body, &actualRequestData)
		require.NoError(t, err)
		require.Equal(t, data, actualRequestData)
		require.Equal(t, int64(2), handler.actualReq.PluginConfig.OrgID)
		require.Equal(t, "my-plugin", handler.actualReq.PluginConfig.PluginID)
		require.Equal(t, "my-type", handler.actualReq.PluginConfig.PluginType)

		// response
		require.NotNil(t, res)
		require.Equal(t, http.StatusOK, int(res.Code))
		require.Contains(t, res.Headers, "X-Header-Out-1")
		require.Equal(t, []string{"D", "E"}, res.Headers["X-Header-Out-1"].Values)
		require.Contains(t, res.Headers, "X-Header-Out-2")
		require.Equal(t, []string{"F"}, res.Headers["X-Header-Out-2"].Values)
		var actualResponseData map[string]interface{}
		err = json.Unmarshal(res.Body, &actualResponseData)
		require.NoError(t, err)
		require.Equal(t, data, actualResponseData)
	})
}

type testCallResourceHandler struct {
	responseStatus  int
	responseHeaders map[string][]string
	responseBody    []byte
	responseErr     error
	actualReq       *CallResourceRequest
}

func (h *testCallResourceHandler) CallResource(ctx context.Context, req *CallResourceRequest) (*CallResourceResponse, error) {
	h.actualReq = req
	return &CallResourceResponse{
		Status:  h.responseStatus,
		Headers: h.responseHeaders,
		Body:    h.responseBody,
	}, h.responseErr
}
