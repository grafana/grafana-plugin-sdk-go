package httpclient

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCustomHeadersMiddleware(t *testing.T) {
	t.Run("Without custom headers set should return next http.RoundTripper", func(t *testing.T) {
		ctx := &testContext{}
		finalRoundTripper := ctx.createRoundTripper("finalrt")
		customHeaders := CustomHeadersMiddleware()
		rt := customHeaders.CreateMiddleware(Options{}, finalRoundTripper)
		require.NotNil(t, rt)
		middlewareName, ok := customHeaders.(MiddlewareName)
		require.True(t, ok)
		require.Equal(t, CustomHeadersMiddlewareName, middlewareName.MiddlewareName())

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
	})

	t.Run("With custom headers set should apply HTTP headers to the request", func(t *testing.T) {
		ctx := &testContext{}
		finalRoundTripper := ctx.createRoundTripper("final")
		customHeaders := CustomHeadersMiddleware()
		rt := customHeaders.CreateMiddleware(Options{Header: http.Header{
			"X-Headerone":   {"ValueOne"},
			"X-Headertwo":   {"ValueTwo"},
			"X-Headerthree": {"ValueThree", "ValueThreeAgain"},
		}}, finalRoundTripper)
		require.NotNil(t, rt)
		middlewareName, ok := customHeaders.(MiddlewareName)
		require.True(t, ok)
		require.Equal(t, CustomHeadersMiddlewareName, middlewareName.MiddlewareName())

		req, err := http.NewRequest(http.MethodGet, "http://", nil)
		require.NoError(t, err)
		res, err := rt.RoundTrip(req)
		require.NoError(t, err)
		require.NotNil(t, res)
		if res.Body != nil {
			require.NoError(t, res.Body.Close())
		}
		require.Len(t, ctx.callChain, 1)
		require.ElementsMatch(t, []string{"final"}, ctx.callChain)

		require.Equal(t, []string{"ValueOne"}, req.Header.Values("X-Headerone"))
		require.Equal(t, []string{"ValueTwo"}, req.Header.Values("X-Headertwo"))
		require.Equal(t, []string{"ValueThree", "ValueThreeAgain"}, req.Header.Values("X-Headerthree"))
	})

	t.Run("With custom Host header set should apply Host to the request", func(t *testing.T) {
		ctx := &testContext{}
		finalRoundTripper := ctx.createRoundTripper("final")
		customHeaders := CustomHeadersMiddleware()
		rt := customHeaders.CreateMiddleware(Options{Header: http.Header{
			"Host": {"example.com"},
		}}, finalRoundTripper)
		require.NotNil(t, rt)
		middlewareName, ok := customHeaders.(MiddlewareName)
		require.True(t, ok)
		require.Equal(t, CustomHeadersMiddlewareName, middlewareName.MiddlewareName())

		req, err := http.NewRequest(http.MethodGet, "http://", nil)
		require.NoError(t, err)
		res, err := rt.RoundTrip(req)
		require.NoError(t, err)
		require.NotNil(t, res)
		if res.Body != nil {
			require.NoError(t, res.Body.Close())
		}
		require.Len(t, ctx.callChain, 1)
		require.ElementsMatch(t, []string{"final"}, ctx.callChain)

		require.Equal(t, "example.com", req.Host)
	})

	t.Run("With custom headers set should previous HTTP headers to the request", func(t *testing.T) {
		ctx := &testContext{}
		finalRoundTripper := ctx.createRoundTripper("final")
		customHeaders := CustomHeadersMiddleware()
		rt := customHeaders.CreateMiddleware(Options{Header: http.Header{
			"X-Headerone": {"ValueOne"},
		}}, finalRoundTripper)
		require.NotNil(t, rt)
		middlewareName, ok := customHeaders.(MiddlewareName)
		require.True(t, ok)
		require.Equal(t, CustomHeadersMiddlewareName, middlewareName.MiddlewareName())

		req, err := http.NewRequest(http.MethodGet, "http://", nil)
		req.Header.Add("X-Headerone", "Other")
		require.NoError(t, err)
		res, err := rt.RoundTrip(req)
		require.NoError(t, err)
		require.NotNil(t, res)
		if res.Body != nil {
			require.NoError(t, res.Body.Close())
		}
		require.Len(t, ctx.callChain, 1)
		require.ElementsMatch(t, []string{"final"}, ctx.callChain)

		require.Equal(t, []string{"ValueOne"}, req.Header.Values("X-Headerone"))
	})
}
