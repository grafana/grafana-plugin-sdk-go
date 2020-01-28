package backend

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

func TestCallResource(t *testing.T) {
	t.Run("Test call resource using basic routes", func(t *testing.T) {
		anyHandler := &testResourceHandler{}
		getHandler := &testResourceHandler{}
		putHandler := &testResourceHandler{}
		postHandler := &testResourceHandler{}
		deleteHandler := &testResourceHandler{}
		patchHandler := &testResourceHandler{}
		adapter := &sdkAdapter{
			schema: Schema{
				Resources: ResourceMap{
					"test": NewResource("/").
						AddRoute("/", RouteMethodAny, anyHandler.handle).
						AddRoute("/", RouteMethodGet, getHandler.handle).
						AddRoute("/", RouteMethodPut, putHandler.handle).
						AddRoute("/", RouteMethodPost, postHandler.handle).
						AddRoute("/", RouteMethodDelete, deleteHandler.handle).
						AddRoute("/", RouteMethodPatch, patchHandler.handle),
				},
			},
		}

		t.Run("Call non-registered resource should return 404", func(t *testing.T) {
			res, err := adapter.CallResource(context.Background(), &pluginv2.CallResource_Request{
				Config:       &pluginv2.PluginConfig{},
				ResourceName: "non-existing",
			})
			assert.NoError(t, err)
			assert.Equal(t, http.StatusNotFound, int(res.Code))
			assert.Equal(t, 0, anyHandler.callerCount)
		})

		t.Run("Call test resource should call any handler and return 200", func(t *testing.T) {
			res, err := adapter.CallResource(context.Background(), &pluginv2.CallResource_Request{
				Config:       &pluginv2.PluginConfig{},
				ResourceName: "test",
				ResourcePath: "/",
				Method:       http.MethodTrace,
			})
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, int(res.Code))
			assert.Equal(t, 1, anyHandler.callerCount)
		})

		t.Run("Call test resource should call get handler and return 200", func(t *testing.T) {
			res, err := adapter.CallResource(context.Background(), &pluginv2.CallResource_Request{
				Config:       &pluginv2.PluginConfig{},
				ResourceName: "test",
				ResourcePath: "/",
				Method:       http.MethodGet,
			})
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, int(res.Code))
			assert.Equal(t, 1, getHandler.callerCount)
		})

		t.Run("Call test resource should call put handler and return 200", func(t *testing.T) {
			res, err := adapter.CallResource(context.Background(), &pluginv2.CallResource_Request{
				Config:       &pluginv2.PluginConfig{},
				ResourceName: "test",
				ResourcePath: "/",
				Method:       http.MethodPut,
			})
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, int(res.Code))
			assert.Equal(t, 1, putHandler.callerCount)
		})

		t.Run("Call test resource should call post handler and return 200", func(t *testing.T) {
			res, err := adapter.CallResource(context.Background(), &pluginv2.CallResource_Request{
				Config:       &pluginv2.PluginConfig{},
				ResourceName: "test",
				ResourcePath: "/",
				Method:       http.MethodPost,
			})
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, int(res.Code))
			assert.Equal(t, 1, postHandler.callerCount)
		})

		t.Run("Call test resource should call delete handler and return 200", func(t *testing.T) {
			res, err := adapter.CallResource(context.Background(), &pluginv2.CallResource_Request{
				Config:       &pluginv2.PluginConfig{},
				ResourceName: "test",
				ResourcePath: "/",
				Method:       http.MethodDelete,
			})
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, int(res.Code))
			assert.Equal(t, 1, deleteHandler.callerCount)
		})

		t.Run("Call test resource should call patch handler and return 200", func(t *testing.T) {
			res, err := adapter.CallResource(context.Background(), &pluginv2.CallResource_Request{
				Config:       &pluginv2.PluginConfig{},
				ResourceName: "test",
				ResourcePath: "/",
				Method:       http.MethodPatch,
			})
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, int(res.Code))
			assert.Equal(t, 1, patchHandler.callerCount)
		})
	})

	t.Run("Test call resource using advanced routes", func(t *testing.T) {
		anyHandler := &testResourceHandler{}
		getHandler := &testResourceHandler{}
		putHandler := &testResourceHandler{}
		postHandler := &testResourceHandler{}
		deleteHandler := &testResourceHandler{}
		patchHandler := &testResourceHandler{}
		adapter := &sdkAdapter{
			schema: Schema{
				Resources: ResourceMap{
					"test": NewResource("/test/:id").
						AddRoute("/", RouteMethodAny, anyHandler.handle).
						AddRoute("/get", RouteMethodGet, getHandler.handle).
						AddRoute("/put", RouteMethodPut, putHandler.handle).
						AddRoute("/post", RouteMethodPost, postHandler.handle).
						AddRoute("/delete", RouteMethodDelete, deleteHandler.handle).
						AddRoute("/patch", RouteMethodPatch, patchHandler.handle),
				},
			},
		}

		t.Run("Call non-registered resource should return 404", func(t *testing.T) {
			res, err := adapter.CallResource(context.Background(), &pluginv2.CallResource_Request{
				Config:       &pluginv2.PluginConfig{},
				ResourceName: "non-existing",
			})
			assert.NoError(t, err)
			assert.Equal(t, http.StatusNotFound, int(res.Code))
			assert.Equal(t, 0, anyHandler.callerCount)
		})

		t.Run("Call test resource should call any handler and return 200", func(t *testing.T) {
			res, err := adapter.CallResource(context.Background(), &pluginv2.CallResource_Request{
				Config:       &pluginv2.PluginConfig{},
				ResourceName: "test",
				ResourcePath: "/test/:id",
				Method:       http.MethodTrace,
			})
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, int(res.Code))
			assert.Equal(t, 1, anyHandler.callerCount)
		})

		t.Run("Call test resource should call get handler and return 200", func(t *testing.T) {
			res, err := adapter.CallResource(context.Background(), &pluginv2.CallResource_Request{
				Config:       &pluginv2.PluginConfig{},
				ResourceName: "test",
				ResourcePath: "/test/:id/get",
				Method:       http.MethodGet,
			})
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, int(res.Code))
			assert.Equal(t, 1, getHandler.callerCount)
		})

		t.Run("Call test resource should call put handler and return 200", func(t *testing.T) {
			res, err := adapter.CallResource(context.Background(), &pluginv2.CallResource_Request{
				Config:       &pluginv2.PluginConfig{},
				ResourceName: "test",
				ResourcePath: "/test/:id/put",
				Method:       http.MethodPut,
			})
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, int(res.Code))
			assert.Equal(t, 1, putHandler.callerCount)
		})

		t.Run("Call test resource should call post handler and return 200", func(t *testing.T) {
			res, err := adapter.CallResource(context.Background(), &pluginv2.CallResource_Request{
				Config:       &pluginv2.PluginConfig{},
				ResourceName: "test",
				ResourcePath: "/test/:id/post",
				Method:       http.MethodPost,
			})
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, int(res.Code))
			assert.Equal(t, 1, postHandler.callerCount)
		})

		t.Run("Call test resource should call delete handler and return 200", func(t *testing.T) {
			res, err := adapter.CallResource(context.Background(), &pluginv2.CallResource_Request{
				Config:       &pluginv2.PluginConfig{},
				ResourceName: "test",
				ResourcePath: "/test/:id/delete",
				Method:       http.MethodDelete,
			})
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, int(res.Code))
			assert.Equal(t, 1, deleteHandler.callerCount)
		})

		t.Run("Call test resource should call patch handler and return 200", func(t *testing.T) {
			res, err := adapter.CallResource(context.Background(), &pluginv2.CallResource_Request{
				Config:       &pluginv2.PluginConfig{},
				ResourceName: "test",
				ResourcePath: "/test/:id/patch",
				Method:       http.MethodPatch,
			})
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, int(res.Code))
			assert.Equal(t, 1, patchHandler.callerCount)
		})
	})

	t.Run("Call resource should provided expected request to route handler", func(t *testing.T) {
		handler := &testResourceHandler{}
		adapter := &sdkAdapter{
			schema: Schema{
				Resources: ResourceMap{
					"test": NewResource("/test").
						AddRoute("/", RouteMethodAny, handler.handle),
				},
			},
		}

		jsonMap := map[string]interface{}{
			"message": "hello",
		}
		body, _ := json.Marshal(&jsonMap)
		protoReq := &pluginv2.CallResource_Request{
			Config: &pluginv2.PluginConfig{
				Id:       2,
				OrgId:    3,
				Name:     "my-name",
				Type:     "my-type",
				Url:      "http://",
				JsonData: "{}",
			},
			ResourceName: "test",
			ResourcePath: "/test",
			Method:       http.MethodPost,
			Url:          "/api/plugins/test-plugin/resources/test",
			Headers: map[string]*pluginv2.CallResource_StringList{
				"X-Header-1": &pluginv2.CallResource_StringList{Values: []string{"A", "B"}},
				"X-Header-2": &pluginv2.CallResource_StringList{Values: []string{"C"}},
			},
			Params: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			Body: body,
		}
		_, err := adapter.CallResource(context.Background(), protoReq)
		require.NoError(t, err)
		require.Equal(t, 1, handler.callerCount)
		require.NotNil(t, handler.resourceCtx)
		require.NotNil(t, handler.resourceCtx.PluginConfig)
		require.Equal(t, protoReq.Config.Id, handler.resourceCtx.PluginConfig.ID)
		require.Equal(t, protoReq.Config.OrgId, handler.resourceCtx.PluginConfig.OrgID)
		require.Equal(t, protoReq.Config.Name, handler.resourceCtx.PluginConfig.Name)
		require.Equal(t, protoReq.Config.Type, handler.resourceCtx.PluginConfig.Type)
		require.Equal(t, protoReq.Config.Url, handler.resourceCtx.PluginConfig.URL)
		require.Equal(t, protoReq.Config.JsonData, string(handler.resourceCtx.PluginConfig.JSONData))
		require.Equal(t, "test", handler.resourceCtx.ResourceName)
		require.Equal(t, "/test", handler.resourceCtx.ResourcePath)
		require.Contains(t, handler.resourceCtx.params, "key1")
		require.Equal(t, "value1", handler.resourceCtx.params["key1"])
		require.Contains(t, handler.resourceCtx.params, "key2")
		require.Equal(t, "value2", handler.resourceCtx.params["key2"])
		require.NotNil(t, handler.req)
		require.Equal(t, protoReq.Url, handler.req.URL.String())
		require.Equal(t, protoReq.Method, handler.req.Method)
		require.Contains(t, handler.req.Header, "X-Header-1")
		require.Equal(t, []string{"A", "B"}, handler.req.Header["X-Header-1"])
		require.Contains(t, handler.req.Header, "X-Header-2")
		require.Equal(t, []string{"C"}, handler.req.Header["X-Header-2"])
		require.NotNil(t, handler.req.Body)
		defer handler.req.Body.Close()
		actualBodyBytes, err := ioutil.ReadAll(handler.req.Body)
		require.NoError(t, err)
		var actualJSONMap map[string]interface{}
		err = json.Unmarshal(actualBodyBytes, &actualJSONMap)
		require.NoError(t, err)
		require.Contains(t, actualJSONMap, "message")
		require.Equal(t, "hello", actualJSONMap["message"])
	})

	t.Run("Call resource should return expected response from route handler", func(t *testing.T) {
		handler := &testResourceHandler{
			responseHeaders: map[string][]string{
				"X-Header-1": []string{"A", "B"},
				"X-Header-2": []string{"C"},
			},
			responseData: map[string]interface{}{
				"message": "hello",
			},
		}
		adapter := &sdkAdapter{
			schema: Schema{
				Resources: ResourceMap{
					"test": NewResource("/test").
						AddRoute("/", RouteMethodAny, handler.handle),
				},
			},
		}

		actualRes, err := adapter.CallResource(context.Background(), &pluginv2.CallResource_Request{
			Config:       &pluginv2.PluginConfig{},
			ResourceName: "test",
			ResourcePath: "/test",
			Method:       http.MethodGet,
		})
		require.NoError(t, err)
		require.Equal(t, 1, handler.callerCount)
		require.NoError(t, handler.writeErr)
		require.NotNil(t, actualRes)
		require.Equal(t, http.StatusOK, int(actualRes.Code))
		require.Contains(t, actualRes.Headers, "X-Header-1")
		require.Equal(t, &pluginv2.CallResource_StringList{Values: []string{"A", "B"}}, actualRes.Headers["X-Header-1"])
		require.Contains(t, actualRes.Headers, "X-Header-2")
		require.Equal(t, &pluginv2.CallResource_StringList{Values: []string{"C"}}, actualRes.Headers["X-Header-2"])
		var actualJSONMap map[string]interface{}
		err = json.Unmarshal(actualRes.Body, &actualJSONMap)
		require.NoError(t, err)
		require.Contains(t, actualJSONMap, "message")
		require.Equal(t, "hello", actualJSONMap["message"])
	})
}

type testResourceHandler struct {
	responseStatus  int
	responseHeaders map[string][]string
	responseData    map[string]interface{}
	callerCount     int
	resourceCtx     *ResourceRequestContext
	req             *http.Request
	writeErr        error
}

func (h *testResourceHandler) handle(resourceCtx *ResourceRequestContext) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		h.callerCount++
		h.resourceCtx = resourceCtx
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
	})
}
