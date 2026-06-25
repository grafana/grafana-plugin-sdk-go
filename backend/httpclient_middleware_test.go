package backend_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/handlertest"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/stretchr/testify/require"
)

// forwardPluginRequestHTTPHeaders mirrors the unexported middleware name in
// backend/httpclient_middleware.go (this is an external test package).
const forwardPluginRequestHTTPHeaders = "forward-plugin-request-http-headers"

func TestHTTPClientMiddleware(t *testing.T) {
	const otherHeader = "test"

	pluginCtx := backend.PluginContext{
		DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{},
	}

	newTest := func(t *testing.T) *handlertest.HandlerMiddlewareTest {
		t.Helper()
		return handlertest.NewHandlerMiddlewareTest(t,
			handlertest.WithMiddlewares(backend.NewHTTPClientMiddleware()),
		)
	}

	t.Run("When no http headers in plugin request", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/some/thing", nil)
		require.NoError(t, err)

		t.Run("Should not forward arbitrary headers when calling QueryData", func(t *testing.T) {
			cdt := newTest(t)
			_, err = cdt.MiddlewareHandler.QueryData(req.Context(), &backend.QueryDataRequest{
				PluginContext: pluginCtx,
				Headers:       map[string]string{otherHeader: "val"},
			})
			require.NoError(t, err)

			reqClone := applyContextualMiddleware(t, cdt.QueryDataCtx, req)
			require.Len(t, reqClone.Header, 0)
		})

		t.Run("Should not forward arbitrary headers when calling CheckHealth", func(t *testing.T) {
			cdt := newTest(t)
			_, err = cdt.MiddlewareHandler.CheckHealth(req.Context(), &backend.CheckHealthRequest{
				PluginContext: pluginCtx,
				Headers:       map[string]string{otherHeader: "val"},
			})
			require.NoError(t, err)

			reqClone := applyContextualMiddleware(t, cdt.CheckHealthCtx, req)
			require.Len(t, reqClone.Header, 0)
		})
	})

	t.Run("When HTTP headers in plugin request", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/some/thing", nil)
		require.NoError(t, err)

		headers := map[string]string{
			backend.FromAlertHeaderName:            "true",
			backend.OAuthIdentityTokenHeaderName:   "bearer token",
			backend.OAuthIdentityIDTokenHeaderName: "id-token",
			"http_X-Custom":                        "custom-value",
			backend.CookiesHeaderName:              "cookie1=; cookie2=; cookie3=",
			otherHeader:                            "val",
		}

		crHeaders := map[string][]string{}
		for k, v := range headers {
			crHeaders[k] = []string{v}
		}

		// QueryData and CheckHealth use getHTTPHeadersFromStringMap, which only
		// exposes OAuth/cookie/http_-prefixed headers; the middleware additionally
		// forwards the plain FromAlert. So otherHeader ("test") is dropped and
		// http_X-Custom is exposed as X-Custom.
		assertFiltered := func(t *testing.T, reqClone *http.Request) {
			t.Helper()
			require.Len(t, reqClone.Header, 5)
			require.Equal(t, "true", reqClone.Header.Get(backend.FromAlertHeaderName))
			require.Equal(t, "bearer token", reqClone.Header.Get(backend.OAuthIdentityTokenHeaderName))
			require.Equal(t, "id-token", reqClone.Header.Get(backend.OAuthIdentityIDTokenHeaderName))
			require.Equal(t, "custom-value", reqClone.Header.Get("X-Custom"))
			require.Len(t, reqClone.Cookies(), 3)
			require.Equal(t, "cookie1", reqClone.Cookies()[0].Name)
			require.Equal(t, "cookie2", reqClone.Cookies()[1].Name)
			require.Equal(t, "cookie3", reqClone.Cookies()[2].Name)
		}

		t.Run("Should forward headers when calling QueryData", func(t *testing.T) {
			cdt := newTest(t)
			_, err = cdt.MiddlewareHandler.QueryData(req.Context(), &backend.QueryDataRequest{
				PluginContext: pluginCtx,
				Headers:       headers,
			})
			require.NoError(t, err)
			assertFiltered(t, applyContextualMiddleware(t, cdt.QueryDataCtx, req))
		})

		t.Run("Should forward headers when calling CheckHealth", func(t *testing.T) {
			cdt := newTest(t)
			_, err = cdt.MiddlewareHandler.CheckHealth(req.Context(), &backend.CheckHealthRequest{
				PluginContext: pluginCtx,
				Headers:       headers,
			})
			require.NoError(t, err)
			assertFiltered(t, applyContextualMiddleware(t, cdt.CheckHealthCtx, req))
		})

		// CallResourceRequest.GetHTTPHeaders() returns all headers unfiltered, so
		// every header is forwarded (FromAlert is also set via the special-case).
		t.Run("Should forward headers when calling CallResource", func(t *testing.T) {
			cdt := newTest(t)
			err = cdt.MiddlewareHandler.CallResource(req.Context(), &backend.CallResourceRequest{
				PluginContext: pluginCtx,
				Headers:       crHeaders,
			}, nopCallResourceSender{})
			require.NoError(t, err)

			reqClone := applyContextualMiddleware(t, cdt.CallResourceCtx, req)
			require.Equal(t, "true", reqClone.Header.Get(backend.FromAlertHeaderName))
			require.Equal(t, "bearer token", reqClone.Header.Get(backend.OAuthIdentityTokenHeaderName))
			require.Equal(t, "id-token", reqClone.Header.Get(backend.OAuthIdentityIDTokenHeaderName))
			require.Len(t, reqClone.Cookies(), 3)
		})

		t.Run("Should not overwrite an existing header", func(t *testing.T) {
			cdt := newTest(t)
			_, err = cdt.MiddlewareHandler.CheckHealth(req.Context(), &backend.CheckHealthRequest{
				PluginContext: pluginCtx,
				Headers:       headers,
			})
			require.NoError(t, err)

			middlewares := httpclient.ContextualMiddlewareFromContext(cdt.CheckHealthCtx)
			require.Len(t, middlewares, 1)

			reqClone := req.Clone(req.Context())
			// Pretend a preceding middleware already set this header.
			reqClone.Header.Set(backend.OAuthIdentityTokenHeaderName, "bearer test-token")
			res, err := middlewares[0].CreateMiddleware(httpclient.Options{}, finalRoundTripper).RoundTrip(reqClone)
			require.NoError(t, err)
			require.NoError(t, res.Body.Close())
			require.Len(t, reqClone.Header, 5)
			require.Equal(t, "true", reqClone.Header.Get(backend.FromAlertHeaderName))
			require.Equal(t, "bearer test-token", reqClone.Header.Get(backend.OAuthIdentityTokenHeaderName))
			require.Equal(t, "id-token", reqClone.Header.Get(backend.OAuthIdentityIDTokenHeaderName))
			require.Equal(t, "custom-value", reqClone.Header.Get("X-Custom"))
			require.Len(t, reqClone.Cookies(), 3)
		})
	})
}

// applyContextualMiddleware runs the single contextual middleware installed by
// NewHTTPClientMiddleware against a fresh clone of req and returns the clone, so
// the resulting outgoing headers can be inspected.
func applyContextualMiddleware(t *testing.T, ctx context.Context, req *http.Request) *http.Request {
	t.Helper()
	middlewares := httpclient.ContextualMiddlewareFromContext(ctx)
	require.Len(t, middlewares, 1)
	require.Equal(t, forwardPluginRequestHTTPHeaders, middlewares[0].(httpclient.MiddlewareName).MiddlewareName())

	reqClone := req.Clone(req.Context())
	res, err := middlewares[0].CreateMiddleware(httpclient.Options{}, finalRoundTripper).RoundTrip(reqClone)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())
	return reqClone
}

type nopCallResourceSender struct{}

func (nopCallResourceSender) Send(*backend.CallResourceResponse) error { return nil }

var finalRoundTripper = httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Request:    req,
		Body:       io.NopCloser(bytes.NewBufferString("")),
	}, nil
})
