package httpresource

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/require"
)

func TestHttpResourceHandler(t *testing.T) {
	t.Run("Given http resource handler and calling CallResource", func(t *testing.T) {
		httpHandler := &testHTTPHandler{
			responseHeaders: map[string][]string{
				"X-Header-Out-1": []string{"A", "B"},
				"X-Header-Out-2": []string{"C"},
			},
			responseData: map[string]interface{}{
				"message": "hello client",
			},
			responseStatus: http.StatusCreated,
		}
		resourceHandler := New(httpHandler)

		jsonMap := map[string]interface{}{
			"message": "hello server",
		}
		reqBody, err := json.Marshal(&jsonMap)
		require.NoError(t, err)

		jsonData := map[string]interface{}{
			"prop": "value",
		}
		jsonDataBytes, err := json.Marshal(&jsonData)
		require.NoError(t, err)

		updated, err := time.Parse(time.RFC3339, "2020-02-14T00:00:00+00:00")
		require.NoError(t, err)

		req := &backend.CallResourceRequest{
			PluginConfig: backend.PluginConfig{
				OrgID:      3,
				PluginID:   "my-plugin",
				PluginType: "my-type",
				JSONData:   jsonDataBytes,
				DecryptedSecureJSONData: map[string]string{
					"secureProp": "secure value",
				},
				Updated: updated,
				DataSourceConfig: &backend.DataSourceConfig{
					ID:               2,
					Name:             "my-ds",
					URL:              "http://",
					Database:         "db123",
					User:             "usr",
					BasicAuthEnabled: true,
					BasicAuthUser:    "busr",
				},
			},
			Method: http.MethodPost,
			Path:   "path",
			URL:    "/api/plugins/plugin-abc/resources/path?query=1",
			Headers: map[string][]string{
				"X-Header-In-1": []string{"D", "E"},
				"X-Header-In-2": []string{"F"},
			},
			Body: reqBody,
			User: &backend.User{Name: "foobar", Email: "foo@bar.com", Login: "foo@bar.com"},
		}
		resp, err := resourceHandler.CallResource(context.Background(), req)
		require.NoError(t, err)
		require.Equal(t, 1, httpHandler.callerCount)

		t.Run("Should provide expected request to http handler", func(t *testing.T) {
			require.NotNil(t, httpHandler.req)
			require.Equal(t, "/path?query=1", httpHandler.req.URL.String())
			require.Equal(t, req.Method, httpHandler.req.Method)
			require.Contains(t, httpHandler.req.Header, "X-Header-In-1")
			require.Equal(t, []string{"D", "E"}, httpHandler.req.Header["X-Header-In-1"])
			require.Contains(t, httpHandler.req.Header, "X-Header-In-2")
			require.Equal(t, []string{"F"}, httpHandler.req.Header["X-Header-In-2"])
			require.NotNil(t, httpHandler.req.Body)
			defer httpHandler.req.Body.Close()
			actualBodyBytes, err := ioutil.ReadAll(httpHandler.req.Body)
			require.NoError(t, err)
			var actualJSONMap map[string]interface{}
			err = json.Unmarshal(actualBodyBytes, &actualJSONMap)
			require.NoError(t, err)
			require.Contains(t, actualJSONMap, "message")
			require.Equal(t, "hello server", actualJSONMap["message"])
		})

		t.Run("Should return expected response from http handler", func(t *testing.T) {
			require.NotNil(t, resp)
			require.NoError(t, httpHandler.writeErr)
			require.NotNil(t, resp)
			require.Equal(t, http.StatusCreated, resp.Status)
			require.Contains(t, resp.Headers, "X-Header-Out-1")
			require.Equal(t, []string{"A", "B"}, resp.Headers["X-Header-Out-1"])
			require.Contains(t, resp.Headers, "X-Header-Out-2")
			require.Equal(t, []string{"C"}, resp.Headers["X-Header-Out-2"])
			var actualJSONMap map[string]interface{}
			err = json.Unmarshal(resp.Body, &actualJSONMap)
			require.NoError(t, err)
			require.Contains(t, actualJSONMap, "message")
			require.Equal(t, "hello client", actualJSONMap["message"])
		})

		t.Run("Should be able to get plugin config from request context", func(t *testing.T) {
			require.NotNil(t, httpHandler.req)
			pluginCfg := PluginConfigFromContext(httpHandler.req.Context())
			require.NotNil(t, pluginCfg)
			require.Equal(t, req.PluginConfig.OrgID, pluginCfg.OrgID)
			require.Equal(t, req.PluginConfig.PluginID, pluginCfg.PluginID)
			require.Equal(t, req.PluginConfig.PluginType, pluginCfg.PluginType)
			require.Equal(t, req.PluginConfig.JSONData, pluginCfg.JSONData)
			require.Equal(t, req.PluginConfig.DecryptedSecureJSONData, pluginCfg.DecryptedSecureJSONData)
			require.Equal(t, req.PluginConfig.Updated, pluginCfg.Updated)
			require.NotNil(t, pluginCfg.DataSourceConfig)
			require.Equal(t, req.PluginConfig.DataSourceConfig.ID, pluginCfg.DataSourceConfig.ID)
			require.Equal(t, req.PluginConfig.DataSourceConfig.Name, pluginCfg.DataSourceConfig.Name)
			require.Equal(t, req.PluginConfig.DataSourceConfig.URL, pluginCfg.DataSourceConfig.URL)
			require.Equal(t, req.PluginConfig.DataSourceConfig.User, pluginCfg.DataSourceConfig.User)
			require.Equal(t, req.PluginConfig.DataSourceConfig.Database, pluginCfg.DataSourceConfig.Database)
			require.Equal(t, req.PluginConfig.DataSourceConfig.BasicAuthEnabled, pluginCfg.DataSourceConfig.BasicAuthEnabled)
			require.Equal(t, req.PluginConfig.DataSourceConfig.BasicAuthUser, pluginCfg.DataSourceConfig.BasicAuthUser)

			user := UserFromContext(httpHandler.req.Context())
			require.NotNil(t, user)
		})
	})
}

func TestServeMuxHandler(t *testing.T) {
	t.Run("Given http resource ServeMux handler and calling CallResource", func(t *testing.T) {
		mux := http.NewServeMux()
		handlerWasCalled := false
		mux.HandleFunc("/test", func(rw http.ResponseWriter, req *http.Request) {
			handlerWasCalled = true
		})
		resourceHandler := New(mux)

		req := &backend.CallResourceRequest{
			PluginConfig: backend.PluginConfig{
				OrgID:      3,
				PluginID:   "my-plugin",
				PluginType: "my-type",
			},
			Method: http.MethodGet,
			Path:   "test",
			URL:    "/test?query=1",
		}
		_, err := resourceHandler.CallResource(context.Background(), req)
		require.NoError(t, err)
		require.True(t, handlerWasCalled)
	})
}

type testHTTPHandler struct {
	responseStatus  int
	responseHeaders map[string][]string
	responseData    map[string]interface{}
	callerCount     int
	req             *http.Request
	writeErr        error
}

func (h *testHTTPHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	h.callerCount++
	h.req = req

	if h.responseHeaders != nil {
		for k, values := range h.responseHeaders {
			for _, v := range values {
				rw.Header().Add(k, v)
			}
		}
	}

	if h.responseStatus != 0 {
		rw.WriteHeader(h.responseStatus)
	} else {
		rw.WriteHeader(200)
	}

	if h.responseData != nil {
		body, _ := json.Marshal(&h.responseData)
		_, h.writeErr = rw.Write(body)
	}
}
