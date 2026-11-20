package httpclient

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend/useragent"
	"github.com/grafana/grafana-plugin-sdk-go/config"
)

func TestUserAgentMiddleware(t *testing.T) {
	runTestCase := func(t *testing.T, requestHeaders http.Header) http.Header {
		t.Helper()

		testCtx := &testContext{}
		finalRoundTripper := testCtx.createRoundTripper("final")
		userAgent := newUserAgentMiddleware("test-plugin", "1.2.3", true)
		rt := userAgent.CreateMiddleware(Options{}, finalRoundTripper)
		require.NotNil(t, rt)
		middlewareName, ok := userAgent.(MiddlewareName)
		require.True(t, ok)
		require.Equal(t, UserAgentMiddlewareName, middlewareName.MiddlewareName())

		ua, err := useragent.New("4.5.6", "SomeOS", "x64")
		require.NoError(t, err)
		ctx := useragent.WithUserAgent(context.Background(), ua)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://", nil)
		require.NoError(t, err)
		req.Header = requestHeaders

		res, err := rt.RoundTrip(req)
		require.NoError(t, err)
		require.NotNil(t, res)
		if res.Body != nil {
			require.NoError(t, res.Body.Close())
		}
		require.Len(t, testCtx.callChain, 1)
		require.ElementsMatch(t, []string{"final"}, testCtx.callChain)

		return req.Header
	}

	t.Run("when no headers are present on the request", func(t *testing.T) {
		headers := http.Header{}
		finalHeaders := runTestCase(t, headers)
		expectedHeaders := http.Header{
			"User-Agent": []string{"Grafana/4.5.6 (SomeOS; x64) test-plugin/1.2.3"},
		}

		require.Equal(t, expectedHeaders, finalHeaders)
	})

	t.Run("when other headers are present on the request, but no User-Agent", func(t *testing.T) {
		headers := http.Header{
			"X-Foo": []string{"bar"},
		}
		finalHeaders := runTestCase(t, headers)
		expectedHeaders := http.Header{
			"User-Agent": []string{"Grafana/4.5.6 (SomeOS; x64) test-plugin/1.2.3"},
			"X-Foo":      []string{"bar"},
		}

		require.Equal(t, expectedHeaders, finalHeaders)
	})

	t.Run("when a User-Agent header is already present", func(t *testing.T) {
		headers := http.Header{
			"User-Agent": []string{"foo"},
			"X-Foo":      []string{"bar"},
		}
		finalHeaders := runTestCase(t, headers)
		expectedHeaders := http.Header{
			"User-Agent": []string{"foo"},
			"X-Foo":      []string{"bar"},
		}

		require.Equal(t, expectedHeaders, finalHeaders)
	})

	t.Run("when no user agent is set on the request context", func(t *testing.T) {
		testCtx := &testContext{}
		finalRoundTripper := testCtx.createRoundTripper("final")
		userAgent := newUserAgentMiddleware("test-plugin", "1.2.3", true)
		rt := userAgent.CreateMiddleware(Options{}, finalRoundTripper)

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://", nil)
		require.NoError(t, err)

		res, err := rt.RoundTrip(req)
		require.NoError(t, err)
		require.NotNil(t, res)
		if res.Body != nil {
			require.NoError(t, res.Body.Close())
		}

		expectedHeaders := http.Header{
			"User-Agent": []string{"Grafana test-plugin/1.2.3"},
		}
		require.Equal(t, expectedHeaders, req.Header)
	})
}

func TestUserAgentMiddleware_ConfiguredUserAgent(t *testing.T) {
	roundTrip := func(t *testing.T, haveVersionInfo bool, configured string, requestHeaders http.Header) http.Header {
		t.Helper()

		testCtx := &testContext{}
		rt := newUserAgentMiddleware("test-plugin", "1.2.3", haveVersionInfo).CreateMiddleware(Options{}, testCtx.createRoundTripper("final"))

		ua, err := useragent.New("4.5.6", "SomeOS", "x64")
		require.NoError(t, err)
		ctx := useragent.WithUserAgent(context.Background(), ua)
		ctx = config.WithGrafanaConfig(ctx, config.NewGrafanaCfg(map[string]string{
			config.PluginsUserAgent: configured,
		}))
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://", nil)
		require.NoError(t, err)
		if requestHeaders != nil {
			req.Header = requestHeaders
		}

		res, err := rt.RoundTrip(req)
		require.NoError(t, err)
		if res.Body != nil {
			require.NoError(t, res.Body.Close())
		}
		return req.Header
	}

	t.Run("uses the configured user agent as the prefix", func(t *testing.T) {
		headers := roundTrip(t, true, "Grafana/4.5.6 my-fleet/1", nil)
		require.Equal(t, "Grafana/4.5.6 my-fleet/1 test-plugin/1.2.3", headers.Get("User-Agent"))
	})

	t.Run("sets the configured user agent when the plugin has no build info", func(t *testing.T) {
		headers := roundTrip(t, false, "Grafana/4.5.6 my-fleet/1", nil)
		require.Equal(t, "Grafana/4.5.6 my-fleet/1", headers.Get("User-Agent"))
	})

	t.Run("does not override a User-Agent already on the request", func(t *testing.T) {
		headers := roundTrip(t, true, "Grafana/4.5.6 my-fleet/1", http.Header{"User-Agent": []string{"foo"}})
		require.Equal(t, "foo", headers.Get("User-Agent"))
	})

	t.Run("falls back to the context user agent when the value is empty", func(t *testing.T) {
		headers := roundTrip(t, true, "", nil)
		require.Equal(t, "Grafana/4.5.6 (SomeOS; x64) test-plugin/1.2.3", headers.Get("User-Agent"))
	})

	t.Run("ignores a value with control characters", func(t *testing.T) {
		headers := roundTrip(t, true, "Grafana/4.5.6\r\nX-Injected: 1", nil)
		require.Equal(t, "Grafana/4.5.6 (SomeOS; x64) test-plugin/1.2.3", headers.Get("User-Agent"))
	})

	t.Run("leaves the header unset with no build info and nothing configured", func(t *testing.T) {
		headers := roundTrip(t, false, "", nil)
		require.Empty(t, headers.Get("User-Agent"))
	})
}
