package backend

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	"github.com/grafana/grafana-plugin-sdk-go/internal/tenant"
)

func TestCallCustomRoute(t *testing.T) {
	t.Run("When custom route handler not set should return http status not implemented", func(t *testing.T) {
		testSender := newTestCallCustomRouteServer()
		adapter := &customRouteSDKAdapter{}
		err := adapter.CallCustomRoute(&pluginv2.CallCustomRouteRequest{}, testSender)
		require.NoError(t, err)
		require.Len(t, testSender.respMessages, 1)
		resp := testSender.respMessages[0]
		require.NotNil(t, resp)
		require.Equal(t, http.StatusNotImplemented, int(resp.Code))
		require.Empty(t, resp.Headers)
		require.Empty(t, resp.Body)
	})

	t.Run("When custom route handler set should provide expected request and return expected response", func(t *testing.T) {
		body := []byte(`{"message":"hello"}`)
		handler := &testCallCustomRouteHandler{
			responseStatus: http.StatusOK,
			responseHeaders: map[string][]string{
				"X-Header-Out-1": {"D", "E"},
				"X-Header-Out-2": {"F"},
			},
			responseBody: body,
		}
		testSender := newTestCallCustomRouteServer()
		adapter := newCustomRouteSDKAdapter(handler)
		req := &pluginv2.CallCustomRouteRequest{
			PluginContext: &pluginv2.PluginContext{
				OrgId:    2,
				PluginId: "my-plugin",
			},
			Identifier: &pluginv2.ResourceFullIdentifier{
				Namespace: "default",
				Name:      "foo",
				Group:     "test.grafana.app",
				Version:   "v1",
				Kind:      "Foo",
				Plural:    "foos",
			},
			Path:   "bar",
			Method: http.MethodGet,
			Url:    "/apis/test.grafana.app/v1/namespaces/default/foos/foo/bar?test=1",
			Headers: map[string]*pluginv2.StringList{
				"X-Header-In-1": {Values: []string{"A", "B"}},
				"X-Header-In-2": {Values: []string{"C"}},
			},
			Body: body,
		}
		err := adapter.CallCustomRoute(req, testSender)
		require.NoError(t, err)

		// request
		require.NotNil(t, handler.actualReq)
		require.Equal(t, int64(2), handler.actualReq.PluginContext.OrgID) // nolint:staticcheck
		require.Equal(t, "my-plugin", handler.actualReq.PluginContext.PluginID)
		require.Equal(t, "test.grafana.app", handler.actualReq.Identifier.Group)
		require.Equal(t, "v1", handler.actualReq.Identifier.Version)
		require.Equal(t, "Foo", handler.actualReq.Identifier.Kind)
		require.Equal(t, "foos", handler.actualReq.Identifier.Plural)
		require.Equal(t, "foo", handler.actualReq.Identifier.Name)
		require.Equal(t, "default", handler.actualReq.Identifier.Namespace)
		require.Equal(t, "bar", handler.actualReq.Path)
		require.Equal(t, http.MethodGet, handler.actualReq.Method)
		require.Equal(t, "/apis/test.grafana.app/v1/namespaces/default/foos/foo/bar?test=1", handler.actualReq.URL)
		require.Equal(t, []string{"A", "B"}, handler.actualReq.Headers["X-Header-In-1"])
		require.Equal(t, []string{"C"}, handler.actualReq.Headers["X-Header-In-2"])
		require.Equal(t, body, handler.actualReq.Body)

		// response
		require.Len(t, testSender.respMessages, 1)
		resp := testSender.respMessages[0]
		require.NotNil(t, resp)
		require.Equal(t, http.StatusOK, int(resp.Code))
		require.Equal(t, []string{"D", "E"}, resp.Headers["X-Header-Out-1"].Values)
		require.Equal(t, []string{"F"}, resp.Headers["X-Header-Out-2"].Values)
		require.Equal(t, body, resp.Body)
	})

	t.Run("When custom route handler set should result in expected streaming response", func(t *testing.T) {
		handler := &testCallCustomRouteStreamHandler{
			responseStatus: http.StatusOK,
			responseMessages: [][]byte{
				[]byte("hello"),
				[]byte("world"),
				[]byte("over and out"),
			},
		}
		testSender := newTestCallCustomRouteServer()
		adapter := newCustomRouteSDKAdapter(handler)
		err := adapter.CallCustomRoute(&pluginv2.CallCustomRouteRequest{
			PluginContext: &pluginv2.PluginContext{},
		}, testSender)
		require.NoError(t, err)

		require.Len(t, testSender.respMessages, 3)
		require.Equal(t, http.StatusOK, int(testSender.respMessages[0].Code))
		require.Equal(t, "hello", string(testSender.respMessages[0].Body))
		require.Equal(t, "world", string(testSender.respMessages[1].Body))
		require.Equal(t, "over and out", string(testSender.respMessages[2].Body))
	})

	t.Run("When tenant information is attached to incoming context, it is propagated from adapter to handler", func(t *testing.T) {
		tid := "123456"
		handlers := Handlers{
			CustomRouteHandler: CallCustomRouteHandlerFunc(func(ctx context.Context, _ *CallCustomRouteRequest, _ CallCustomRouteResponseSender) error {
				require.Equal(t, tid, tenant.IDFromContext(ctx))
				return nil
			}),
		}
		handlerWithMw, err := HandlerFromMiddlewares(handlers, newTenantIDMiddleware())
		require.NoError(t, err)
		a := newCustomRouteSDKAdapter(handlerWithMw)

		testSender := newTestCallCustomRouteServer()
		testSender.WithContext(metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
			tenant.CtxKey: tid,
		})))

		err = a.CallCustomRoute(&pluginv2.CallCustomRouteRequest{
			PluginContext: &pluginv2.PluginContext{},
		}, testSender)
		require.NoError(t, err)
	})
}

type testCallCustomRouteHandler struct {
	responseStatus  int
	responseHeaders map[string][]string
	responseBody    []byte
	responseErr     error
	actualReq       *CallCustomRouteRequest
}

func (h *testCallCustomRouteHandler) CallCustomRoute(_ context.Context, req *CallCustomRouteRequest, sender CallCustomRouteResponseSender) error {
	h.actualReq = req
	err := sender.Send(&CallCustomRouteResponse{
		Status:  h.responseStatus,
		Headers: h.responseHeaders,
		Body:    h.responseBody,
	})
	if err != nil {
		return err
	}

	return h.responseErr
}

type testCallCustomRouteStreamHandler struct {
	responseStatus   int
	responseHeaders  map[string][]string
	responseMessages [][]byte
	responseErr      error
}

func (h *testCallCustomRouteStreamHandler) CallCustomRoute(_ context.Context, _ *CallCustomRouteRequest, sender CallCustomRouteResponseSender) error {
	err := sender.Send(&CallCustomRouteResponse{
		Status:  h.responseStatus,
		Headers: h.responseHeaders,
		Body:    h.responseMessages[0],
	})
	if err != nil {
		return err
	}

	for _, msg := range h.responseMessages[1:] {
		if err := sender.Send(&CallCustomRouteResponse{Body: msg}); err != nil {
			return err
		}
	}

	return h.responseErr
}

type testCallCustomRouteServer struct {
	ctx          context.Context
	respMessages []*pluginv2.CallCustomRouteResponse
}

func newTestCallCustomRouteServer() *testCallCustomRouteServer {
	return &testCallCustomRouteServer{
		respMessages: []*pluginv2.CallCustomRouteResponse{},
		ctx:          context.Background(),
	}
}

func (srv *testCallCustomRouteServer) Send(resp *pluginv2.CallCustomRouteResponse) error {
	srv.respMessages = append(srv.respMessages, resp)
	return nil
}

func (srv *testCallCustomRouteServer) SetHeader(metadata.MD) error {
	return nil
}

func (srv *testCallCustomRouteServer) SendHeader(metadata.MD) error {
	return nil
}

func (srv *testCallCustomRouteServer) SetTrailer(metadata.MD) {
}

func (srv *testCallCustomRouteServer) Context() context.Context {
	return srv.ctx
}

func (srv *testCallCustomRouteServer) SendMsg(_ interface{}) error {
	return nil
}

func (srv *testCallCustomRouteServer) RecvMsg(_ interface{}) error {
	return nil
}

func (srv *testCallCustomRouteServer) WithContext(ctx context.Context) {
	srv.ctx = ctx
}
