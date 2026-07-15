package httpclient

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserAgentMiddleware(t *testing.T) {
	runTestCase := func(t *testing.T, requestHeaders http.Header) http.Header {
		t.Helper()

		ctx := &testContext{}
		finalRoundTripper := ctx.createRoundTripper("final")
		userAgent := newUserAgentMiddleware("test-plugin", "1.2.3", true)
		rt := userAgent.CreateMiddleware(Options{}, finalRoundTripper)
		require.NotNil(t, rt)
		middlewareName, ok := userAgent.(MiddlewareName)
		require.True(t, ok)
		require.Equal(t, UserAgentMiddlewareName, middlewareName.MiddlewareName())

		req, err := http.NewRequest(http.MethodGet, "http://", nil)
		require.NoError(t, err)
		req.Header = requestHeaders

		res, err := rt.RoundTrip(req)
		require.NoError(t, err)
		require.NotNil(t, res)
		if res.Body != nil {
			require.NoError(t, res.Body.Close())
		}
		require.Len(t, ctx.callChain, 1)
		require.ElementsMatch(t, []string{"final"}, ctx.callChain)

		return req.Header
	}

	t.Run("when no headers are present on the request", func(t *testing.T) {
		headers := http.Header{}
		finalHeaders := runTestCase(t, headers)
		expectedHeaders := http.Header{
			"User-Agent": []string{"test-plugin/1.2.3"},
		}

		require.Equal(t, expectedHeaders, finalHeaders)
	})

	t.Run("when other headers are present on the request, but no User-Agent", func(t *testing.T) {
		headers := http.Header{
			"X-Foo": []string{"bar"},
		}
		finalHeaders := runTestCase(t, headers)
		expectedHeaders := http.Header{
			"User-Agent": []string{"test-plugin/1.2.3"},
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
}
