package httpclient

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBearerAuthMiddleware(t *testing.T) {
	t.Run("Without bearer auth options should return next http.RoundTripper", func(t *testing.T) {
		ctx := &testContext{}
		finalRoundTripper := ctx.createRoundTripper("final")
		bearerAuth := BearerAuthenticationMiddleware()
		rt := bearerAuth.CreateMiddleware(Options{}, finalRoundTripper)
		require.NotNil(t, rt)
		middlewareName, ok := bearerAuth.(MiddlewareName)
		require.True(t, ok)
		require.Equal(t, BearerAuthenticationMiddlewareName, middlewareName.MiddlewareName())

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
	})

	t.Run("With bearer auth options should apply bearer auth authentication HTTP header to the request", func(t *testing.T) {
		ctx := &testContext{}
		finalRoundTripper := ctx.createRoundTripper("final")
		bearerAuth := BearerAuthenticationMiddleware()
		rt := bearerAuth.CreateMiddleware(Options{BearerAuth: &BearerAuthOptions{Token: "gf_token"}}, finalRoundTripper)
		require.NotNil(t, rt)
		middlewareName, ok := bearerAuth.(MiddlewareName)
		require.True(t, ok)
		require.Equal(t, BearerAuthenticationMiddlewareName, middlewareName.MiddlewareName())

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

		authHeader := req.Header.Get("Authorization")
		require.NotEmpty(t, authHeader)
		require.True(t, strings.HasPrefix(authHeader, "Bearer"))
		require.Equal(t, "gf_token", strings.TrimPrefix(authHeader, "Bearer "))
	})

	t.Run("With bearer auth options should not apply bearer auth authentication HTTP header to the request if header already set", func(t *testing.T) {
		ctx := &testContext{}
		finalRoundTripper := ctx.createRoundTripper("final")
		bearerAuth := BearerAuthenticationMiddleware()
		rt := bearerAuth.CreateMiddleware(Options{BearerAuth: &BearerAuthOptions{Token: "gf_token"}}, finalRoundTripper)
		require.NotNil(t, rt)
		middlewareName, ok := bearerAuth.(MiddlewareName)
		require.True(t, ok)
		require.Equal(t, BearerAuthenticationMiddlewareName, middlewareName.MiddlewareName())

		req, err := http.NewRequest(http.MethodGet, "http://", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "test")
		res, err := rt.RoundTrip(req)
		require.NoError(t, err)
		require.NotNil(t, res)
		if res.Body != nil {
			require.NoError(t, res.Body.Close())
		}
		require.Len(t, ctx.callChain, 1)
		require.ElementsMatch(t, []string{"final"}, ctx.callChain)

		authHeader := req.Header.Get("Authorization")
		require.Equal(t, "test", authHeader)
	})
}
