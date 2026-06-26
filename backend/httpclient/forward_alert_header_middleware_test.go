package httpclient

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestForwardedFromAlertHeaderMiddleware(t *testing.T) {
	t.Run("Without a FromAlert header in opts should return next http.RoundTripper", func(t *testing.T) {
		ctx := &testContext{}
		finalRoundTripper := ctx.createRoundTripper("final")
		mw := ForwardFromAlertHeaderMiddleware()
		rt := mw.CreateMiddleware(Options{}, finalRoundTripper)
		require.NotNil(t, rt)
		middlewareName, ok := mw.(MiddlewareName)
		require.True(t, ok)
		require.Equal(t, ForwardFromAlertHeaderMiddlewareName, middlewareName.MiddlewareName())

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
		require.Empty(t, req.Header.Get(FromAlertHeaderName))
	})

	t.Run("With a FromAlert header in opts should forward it to the request", func(t *testing.T) {
		ctx := &testContext{}
		finalRoundTripper := ctx.createRoundTripper("final")
		mw := ForwardFromAlertHeaderMiddleware()
		optsHeader := http.Header{}
		optsHeader.Set(FromAlertHeaderName, "true")
		rt := mw.CreateMiddleware(Options{Header: optsHeader}, finalRoundTripper)
		require.NotNil(t, rt)

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
		require.Equal(t, "true", req.Header.Get(FromAlertHeaderName))
	})

	t.Run("With a FromAlert header in opts should not overwrite an existing header on the request", func(t *testing.T) {
		ctx := &testContext{}
		finalRoundTripper := ctx.createRoundTripper("final")
		mw := ForwardFromAlertHeaderMiddleware()
		optsHeader := http.Header{}
		optsHeader.Set(FromAlertHeaderName, "true")
		rt := mw.CreateMiddleware(Options{Header: optsHeader}, finalRoundTripper)
		require.NotNil(t, rt)

		req, err := http.NewRequest(http.MethodGet, "http://", nil)
		require.NoError(t, err)
		req.Header.Set(FromAlertHeaderName, "existing")
		res, err := rt.RoundTrip(req)
		require.NoError(t, err)
		require.NotNil(t, res)
		if res.Body != nil {
			require.NoError(t, res.Body.Close())
		}
		require.Len(t, ctx.callChain, 1)
		require.ElementsMatch(t, []string{"final"}, ctx.callChain)
		require.Equal(t, "existing", req.Header.Get(FromAlertHeaderName))
	})
}
