package httpclient

import (
	"net/http"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend/proxy"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestDataSourceMetricsMiddleware(t *testing.T) {
	t.Run("Without label options set should return next http.RoundTripper", func(t *testing.T) {
		origExecuteMiddlewareFunc := executeMiddlewareFunc
		executeMiddlewareCalled := false
		middlewareCalled := false
		executeMiddlewareFunc = func(next http.RoundTripper, _ string, _ string) http.RoundTripper {
			executeMiddlewareCalled = true
			return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
				middlewareCalled = true
				return next.RoundTrip(r)
			})
		}
		t.Cleanup(func() {
			executeMiddlewareFunc = origExecuteMiddlewareFunc
		})

		ctx := &testContext{}
		finalRoundTripper := ctx.createRoundTripper("finalrt")
		mw := DataSourceMetricsMiddleware()
		rt := mw.CreateMiddleware(Options{}, finalRoundTripper)
		require.NotNil(t, rt)
		middlewareName, ok := mw.(MiddlewareName)
		require.True(t, ok)
		require.Equal(t, DataSourceMetricsMiddlewareName, middlewareName.MiddlewareName())

		req, err := http.NewRequest(http.MethodGet, "http://", nil)
		require.NoError(t, err)
		res, err := rt.RoundTrip(req)
		require.NoError(t, err)
		require.NotNil(t, res)
		if res.Body != nil {
			require.NoError(t, res.Body.Close())
		}
		require.Len(t, ctx.callChain, 1)
		require.ElementsMatch(t, []string{"finalrt"}, ctx.callChain)
		require.False(t, executeMiddlewareCalled)
		require.False(t, middlewareCalled)
	})

	t.Run("Without data source type label options set should return next http.RoundTripper", func(t *testing.T) {
		origExecuteMiddlewareFunc := executeMiddlewareFunc
		executeMiddlewareCalled := false
		middlewareCalled := false
		executeMiddlewareFunc = func(next http.RoundTripper, _ string, _ string) http.RoundTripper {
			executeMiddlewareCalled = true
			return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
				middlewareCalled = true
				return next.RoundTrip(r)
			})
		}
		t.Cleanup(func() {
			executeMiddlewareFunc = origExecuteMiddlewareFunc
		})

		ctx := &testContext{}
		finalRoundTripper := ctx.createRoundTripper("finalrt")
		mw := DataSourceMetricsMiddleware()
		rt := mw.CreateMiddleware(Options{Labels: map[string]string{"test": "test"}}, finalRoundTripper)
		require.NotNil(t, rt)
		middlewareName, ok := mw.(MiddlewareName)
		require.True(t, ok)
		require.Equal(t, DataSourceMetricsMiddlewareName, middlewareName.MiddlewareName())

		req, err := http.NewRequest(http.MethodGet, "http://", nil)
		require.NoError(t, err)
		res, err := rt.RoundTrip(req)
		require.NoError(t, err)
		require.NotNil(t, res)
		if res.Body != nil {
			require.NoError(t, res.Body.Close())
		}
		require.Len(t, ctx.callChain, 1)
		require.ElementsMatch(t, []string{"finalrt"}, ctx.callChain)
		require.False(t, executeMiddlewareCalled)
		require.False(t, middlewareCalled)
	})

	t.Run("With datasource type label options set should execute middleware", func(t *testing.T) {
		origExecuteMiddlewareFunc := executeMiddlewareFunc
		executeMiddlewareCalled := false
		labels := prometheus.Labels{}
		middlewareCalled := false
		executeMiddlewareFunc = func(next http.RoundTripper, datasourceLabel string, secureSocksProxyEnabled string) http.RoundTripper {
			executeMiddlewareCalled = true
			labels = prometheus.Labels{"datasource_type": datasourceLabel, "secure_socks_ds_proxy_enabled": secureSocksProxyEnabled}
			return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
				middlewareCalled = true
				return next.RoundTrip(r)
			})
		}
		t.Cleanup(func() {
			executeMiddlewareFunc = origExecuteMiddlewareFunc
		})

		testCases := []struct {
			description                       string
			httpClientOptions                 Options
			expectedSecureSocksDSProxyEnabled string
		}{
			{
				description: "secure socks ds proxy is disabled",
				httpClientOptions: Options{
					Labels: map[string]string{"datasource_type": "prometheus"},
				},
				expectedSecureSocksDSProxyEnabled: "false",
			},
			{
				description: "secure socks ds proxy is enabled",
				httpClientOptions: Options{
					Labels:       map[string]string{"datasource_type": "prometheus"},
					ProxyOptions: &proxy.Options{Enabled: true},
				},
				expectedSecureSocksDSProxyEnabled: "true",
			},
		}

		for _, tt := range testCases {
			t.Run(tt.description, func(t *testing.T) {
				ctx := &testContext{}
				finalRoundTripper := ctx.createRoundTripper("finalrt")
				mw := DataSourceMetricsMiddleware()
				rt := mw.CreateMiddleware(tt.httpClientOptions, finalRoundTripper)
				require.NotNil(t, rt)
				middlewareName, ok := mw.(MiddlewareName)
				require.True(t, ok)
				require.Equal(t, DataSourceMetricsMiddlewareName, middlewareName.MiddlewareName())

				req, err := http.NewRequest(http.MethodGet, "http://", nil)
				require.NoError(t, err)
				res, err := rt.RoundTrip(req)
				require.NoError(t, err)
				require.NotNil(t, res)
				if res.Body != nil {
					require.NoError(t, res.Body.Close())
				}
				require.Len(t, ctx.callChain, 1)
				require.ElementsMatch(t, []string{"finalrt"}, ctx.callChain)
				require.True(t, executeMiddlewareCalled)
				require.Len(t, labels, 2)
				require.Equal(t, "prometheus", labels["datasource_type"])
				require.Equal(t, tt.expectedSecureSocksDSProxyEnabled, labels["secure_socks_ds_proxy_enabled"])
				require.True(t, middlewareCalled)
			})
		}
	})
}
