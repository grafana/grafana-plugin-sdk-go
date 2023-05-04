package backend

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	"github.com/stretchr/testify/require"
)

type fakeDataHandlerWithOAuth struct {
	cli     *http.Client
	svr     *httptest.Server
	lastReq *http.Request
}

func newFakeDataHandlerWithOAuth() *fakeDataHandlerWithOAuth {
	settings := DataSourceInstanceSettings{}
	opts, err := settings.HTTPClientOptions()
	if err != nil {
		panic("http client options: " + err.Error())
	}
	cli, err := httpclient.New(opts)
	if err != nil {
		panic("httpclient new: " + err.Error())
	}

	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	return &fakeDataHandlerWithOAuth{
		cli: cli,
		svr: svr,
	}
}

func (f *fakeDataHandlerWithOAuth) QueryData(ctx context.Context, req *QueryDataRequest) (*QueryDataResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", f.svr.URL, nil)
	if err != nil {
		return nil, err
	}
	f.lastReq = httpReq

	res, err := f.cli.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	return &QueryDataResponse{}, nil
}

func TestQueryData(t *testing.T) {
	handler := newFakeDataHandlerWithOAuth()
	adapter := newDataSDKAdapter(handler)

	t.Run("When forward HTTP headers enabled should forward headers", func(t *testing.T) {
		ctx := context.Background()
		_, err := adapter.QueryData(ctx, &pluginv2.QueryDataRequest{
			Headers: map[string]string{
				"Authorization": "Bearer 123",
			},
			PluginContext: &pluginv2.PluginContext{},
		})
		require.NoError(t, err)

		middlewares := httpclient.ContextualMiddlewareFromContext(handler.lastReq.Context())
		require.Len(t, middlewares, 1)

		reqClone := handler.lastReq.Clone(handler.lastReq.Context())
		// clean up headers to be sure they are injected
		reqClone.Header = http.Header{}

		res, err := middlewares[0].CreateMiddleware(httpclient.Options{ForwardHTTPHeaders: true}, finalRoundTripper).RoundTrip(reqClone)
		require.NoError(t, err)
		require.NoError(t, res.Body.Close())
		require.Len(t, reqClone.Header, 1)
		require.Equal(t, "Bearer 123", reqClone.Header.Get("Authorization"))
	})

	t.Run("When forward HTTP headers disable should not forward headers", func(t *testing.T) {
		ctx := context.Background()
		_, err := adapter.QueryData(ctx, &pluginv2.QueryDataRequest{
			Headers: map[string]string{
				"Authorization": "Bearer 123",
			},
			PluginContext: &pluginv2.PluginContext{},
		})
		require.NoError(t, err)

		middlewares := httpclient.ContextualMiddlewareFromContext(handler.lastReq.Context())
		require.Len(t, middlewares, 1)

		reqClone := handler.lastReq.Clone(handler.lastReq.Context())
		// clean up headers to be sure they are injected
		reqClone.Header = http.Header{}

		res, err := middlewares[0].CreateMiddleware(httpclient.Options{ForwardHTTPHeaders: false}, finalRoundTripper).RoundTrip(reqClone)
		require.NoError(t, err)
		require.NoError(t, res.Body.Close())
		require.Empty(t, reqClone.Header)
	})
}

var finalRoundTripper = httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Request:    req,
		Body:       io.NopCloser(bytes.NewBufferString("")),
	}, nil
})
