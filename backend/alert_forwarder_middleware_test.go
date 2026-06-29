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

func TestAlertForwarderMiddleware(t *testing.T) {
	const otherHeader = "test"

	pluginCtx := backend.PluginContext{
		DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{},
	}

	newTest := func(t *testing.T) *handlertest.HandlerMiddlewareTest {
		t.Helper()
		return handlertest.NewHandlerMiddlewareTest(t,
			handlertest.WithMiddlewares(backend.NewAlertForwarderMiddleware()),
		)
	}

	t.Run("When no FromAlert header in plugin request", func(t *testing.T) {
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

	t.Run("When FromAlert header is present in plugin request", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/some/thing", nil)
		require.NoError(t, err)

		// Only FromAlertHeaderName must be forwarded; other headers (OAuth, cookies,
		// http_-prefixed, etc.) are the responsibility of headerMiddleware.
		headers := map[string]string{
			backend.FromAlertHeaderName:          "true",
			backend.OAuthIdentityTokenHeaderName: "bearer token",
			otherHeader:                          "val",
		}

		crHeaders := map[string][]string{}
		for k, v := range headers {
			crHeaders[k] = []string{v}
		}

		t.Run("Should forward only FromAlert header when calling QueryData", func(t *testing.T) {
			cdt := newTest(t)
			_, err = cdt.MiddlewareHandler.QueryData(req.Context(), &backend.QueryDataRequest{
				PluginContext: pluginCtx,
				Headers:       headers,
			})
			require.NoError(t, err)

			reqClone := applyContextualMiddleware(t, cdt.QueryDataCtx, req)
			require.Len(t, reqClone.Header, 1)
			require.Equal(t, "true", reqClone.Header.Get(backend.FromAlertHeaderName))
		})

		t.Run("Should forward only FromAlert header when calling CheckHealth", func(t *testing.T) {
			cdt := newTest(t)
			_, err = cdt.MiddlewareHandler.CheckHealth(req.Context(), &backend.CheckHealthRequest{
				PluginContext: pluginCtx,
				Headers:       headers,
			})
			require.NoError(t, err)

			reqClone := applyContextualMiddleware(t, cdt.CheckHealthCtx, req)
			require.Len(t, reqClone.Header, 1)
			require.Equal(t, "true", reqClone.Header.Get(backend.FromAlertHeaderName))
		})

		t.Run("Should forward only FromAlert header when calling CallResource", func(t *testing.T) {
			cdt := newTest(t)
			err = cdt.MiddlewareHandler.CallResource(req.Context(), &backend.CallResourceRequest{
				PluginContext: pluginCtx,
				Headers:       crHeaders,
			}, nopCallResourceSender{})
			require.NoError(t, err)

			reqClone := applyContextualMiddleware(t, cdt.CallResourceCtx, req)
			require.Len(t, reqClone.Header, 1)
			require.Equal(t, "true", reqClone.Header.Get(backend.FromAlertHeaderName))
		})
	})
}

// applyContextualMiddleware runs the single contextual middleware installed by
// NewAlertForwarderMiddleware against a fresh clone of req and returns the clone, so
// the resulting outgoing headers can be inspected.
func applyContextualMiddleware(t *testing.T, ctx context.Context, req *http.Request) *http.Request {
	t.Helper()
	middlewares := httpclient.ContextualMiddlewareFromContext(ctx)
	require.Len(t, middlewares, 1)

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
