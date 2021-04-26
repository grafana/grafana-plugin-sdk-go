package httpclient

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBasicAuthMiddleware(t *testing.T) {
	t.Run("Without basic auth options should return next http.RoundTripper", func(t *testing.T) {
		ctx := &testContext{}
		finalRoundTripper := ctx.createRoundTripper("final")
		basicAuth := BasicAuthenticationMiddleware()
		rt := basicAuth.CreateMiddleware(&Options{}, finalRoundTripper)
		require.NotNil(t, rt)
		middlewareName, ok := basicAuth.(MiddlewareName)
		require.True(t, ok)
		require.Equal(t, BasicAuthenticationMiddlewareName, middlewareName.MiddlewareName())

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

	t.Run("With basic auth options should apply basic auth authentication HTTP header to the request", func(t *testing.T) {
		ctx := &testContext{}
		finalRoundTripper := ctx.createRoundTripper("final")
		basicAuth := BasicAuthenticationMiddleware()
		rt := basicAuth.CreateMiddleware(&Options{BasicAuth: &BasicAuthOptions{User: "user1", Password: "pwd"}}, finalRoundTripper)
		require.NotNil(t, rt)
		middlewareName, ok := basicAuth.(MiddlewareName)
		require.True(t, ok)
		require.Equal(t, BasicAuthenticationMiddlewareName, middlewareName.MiddlewareName())

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

		authHeader := req.Header.Get("Authentication")
		require.NotEmpty(t, authHeader)
		require.True(t, strings.HasPrefix(authHeader, "Basic"))
		user, password, err := decodeBasicAuthHeader(authHeader)
		require.NoError(t, err)
		require.Equal(t, "user1", user)
		require.Equal(t, "pwd", password)
	})
}

func decodeBasicAuthHeader(header string) (string, string, error) {
	var code string
	parts := strings.SplitN(header, " ", 2)
	if len(parts) == 2 && parts[0] == "Basic" {
		code = parts[1]
	}

	decoded, err := base64.StdEncoding.DecodeString(code)
	if err != nil {
		return "", "", err
	}

	userAndPass := strings.SplitN(string(decoded), ":", 2)
	if len(userAndPass) != 2 {
		return "", "", errors.New("invalid basic auth header")
	}

	return userAndPass[0], userAndPass[1], nil
}
