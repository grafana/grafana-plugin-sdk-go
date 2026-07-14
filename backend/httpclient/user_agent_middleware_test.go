package httpclient

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUserAgentMiddleware(t *testing.T) {
	ctx := &testContext{}
	finalRoundTripper := ctx.createRoundTripper("final")
	userAgent := newUserAgentMiddleware("test-plugin", "1.2.3")
	rt := userAgent.CreateMiddleware(Options{}, finalRoundTripper)
	require.NotNil(t, rt)
	middlewareName, ok := userAgent.(MiddlewareName)
	require.True(t, ok)
	require.Equal(t, UserAgentMiddlewareName, middlewareName.MiddlewareName())

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

	require.Equal(t, []string{"test-plugin/1.2.3"}, req.Header.Values("User-Agent"))
}
