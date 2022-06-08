package backend

import (
	"net/http"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/stretchr/testify/require"
)

func TestForwardedCookiesMiddleware(t *testing.T) {
	tcs := []struct {
		desc                 string
		headers              map[string]string
		httpSettings         *HTTPSettings
		expectedCookieHeader string
	}{
		{
			desc:                 "With nil headers and forward cookies not enabled should not populate Cookie header",
			headers:              nil,
			httpSettings:         &HTTPSettings{},
			expectedCookieHeader: "",
		},
		{
			desc:                 "With empty headers and forward cookies not enabled should not populate Cookie header",
			headers:              map[string]string{},
			httpSettings:         &HTTPSettings{},
			expectedCookieHeader: "",
		},
		{
			desc:                 "With nil headers and forward cookies enabled should not populate Cookie header",
			headers:              nil,
			httpSettings:         &HTTPSettings{ForwardCookies: []string{"ci"}},
			expectedCookieHeader: "",
		},
		{
			desc:                 "With empty headers and forward cookies enabled should not populate Cookie header",
			headers:              map[string]string{},
			httpSettings:         &HTTPSettings{ForwardCookies: []string{"ci"}},
			expectedCookieHeader: "",
		},
		{
			desc:                 "With Cookie header set and forward cookies enabled should populate Cookie header",
			headers:              map[string]string{"Cookie": "c1; c2"},
			httpSettings:         &HTTPSettings{ForwardCookies: []string{"ci", "c2"}},
			expectedCookieHeader: "c1; c2",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := &testContext{}
			finalRoundTripper := ctx.createRoundTripper()
			mw := forwardedCookiesMiddleware(tc.headers)
			opts := httpclient.Options{}
			setCustomOptionsFromHTTPSettings(&opts, tc.httpSettings)
			rt := mw.CreateMiddleware(opts, finalRoundTripper)
			require.NotNil(t, rt)
			middlewareName, ok := mw.(httpclient.MiddlewareName)
			require.True(t, ok)
			require.Equal(t, "forwarded-cookies", middlewareName.MiddlewareName())

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
			require.Equal(t, tc.expectedCookieHeader, ctx.req.Header.Get("Cookie"))
		})
	}
}

func TestForwardedOAuthIdentityMiddleware(t *testing.T) {
	tcs := []struct {
		desc                        string
		headers                     map[string]string
		httpSettings                *HTTPSettings
		expectedAuthorizationHeader string
		expectedIDTokenHeader       string
	}{
		{
			desc:                        "With nil headers and forward OAuth identity not enabled should not populate Cookie header",
			headers:                     nil,
			httpSettings:                &HTTPSettings{},
			expectedAuthorizationHeader: "",
			expectedIDTokenHeader:       "",
		},
		{
			desc:                        "With empty headers and forward OAuth identity not enabled should not populate Cookie header",
			headers:                     map[string]string{},
			httpSettings:                &HTTPSettings{},
			expectedAuthorizationHeader: "",
			expectedIDTokenHeader:       "",
		},
		{
			desc:                        "With nil headers and forward OAuth identity enabled should not populate Cookie header",
			headers:                     nil,
			httpSettings:                &HTTPSettings{ForwardOAauthIdentity: true},
			expectedAuthorizationHeader: "",
			expectedIDTokenHeader:       "",
		},
		{
			desc:                        "With empty headers and forward OAuth identity enabled should not populate Cookie header",
			headers:                     map[string]string{},
			httpSettings:                &HTTPSettings{ForwardOAauthIdentity: true},
			expectedAuthorizationHeader: "",
			expectedIDTokenHeader:       "",
		},
		{
			desc:                        "With Authorization header set and forward OAuth identity enabled should populate Authorization header",
			headers:                     map[string]string{"Authorization": "bearer something"},
			httpSettings:                &HTTPSettings{ForwardOAauthIdentity: true},
			expectedAuthorizationHeader: "bearer something",
			expectedIDTokenHeader:       "",
		},
		{
			desc:                        "With X-ID-Token header set and forward OAuth identity enabled should populate Authorization header",
			headers:                     map[string]string{"X-ID-Token": "token payload"},
			httpSettings:                &HTTPSettings{ForwardOAauthIdentity: true},
			expectedAuthorizationHeader: "",
			expectedIDTokenHeader:       "token payload",
		},
		{
			desc: "With Authorization and X-ID-Token header set and forward OAuth identity enabled should populate Authorization and X-Id-Token header",
			headers: map[string]string{
				"Authorization": "bearer something",
				"X-ID-Token":    "token payload",
			},
			httpSettings:                &HTTPSettings{ForwardOAauthIdentity: true},
			expectedAuthorizationHeader: "bearer something",
			expectedIDTokenHeader:       "token payload",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := &testContext{}
			finalRoundTripper := ctx.createRoundTripper()
			mw := forwardedOAuthIdentityMiddleware(tc.headers)
			opts := httpclient.Options{}
			setCustomOptionsFromHTTPSettings(&opts, tc.httpSettings)
			rt := mw.CreateMiddleware(opts, finalRoundTripper)
			require.NotNil(t, rt)
			middlewareName, ok := mw.(httpclient.MiddlewareName)
			require.True(t, ok)
			require.Equal(t, "forwarded-oauth-identity", middlewareName.MiddlewareName())

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
			require.Equal(t, tc.expectedAuthorizationHeader, ctx.req.Header.Get("Authorization"))
			require.Equal(t, tc.expectedIDTokenHeader, ctx.req.Header.Get("X-ID-Token"))
		})
	}
}

type testContext struct {
	callChain []string
	req       *http.Request
}

func (c *testContext) createRoundTripper() http.RoundTripper {
	return httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		c.callChain = append(c.callChain, "final")
		c.req = req
		return &http.Response{StatusCode: http.StatusOK}, nil
	})
}
