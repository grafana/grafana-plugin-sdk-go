package httpclient

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/config"
	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T { return &v }

func TestResponseLimitMiddleware(t *testing.T) {
	tcs := []struct {
		name               string
		limit              int64
		ctxLimit           *int64
		requestLimit       *int64
		envLimit           string
		expectedBodyLength int
		expectedBody       string
		expectErr          bool
	}{
		// limit arg
		{name: "limit arg enforced", limit: 1, expectedBodyLength: 1, expectedBody: "d", expectErr: true},
		{name: "limit arg not exceeded", limit: 1000000, expectedBodyLength: 5, expectedBody: "dummy"},
		{name: "limit arg 0 disables", limit: 0, expectedBodyLength: 5, expectedBody: "dummy"},
		// env var
		{name: "env var enforced when limit arg is 0", limit: 0, envLimit: "1", expectedBodyLength: 1, expectedBody: "d", expectErr: true},
		{name: "env var not exceeded", limit: 0, envLimit: "1000000", expectedBodyLength: 5, expectedBody: "dummy"},
		{name: "invalid env var ignored", limit: 0, envLimit: "-1", expectedBodyLength: 5, expectedBody: "dummy"},
		{name: "zero env var ignored", limit: 0, envLimit: "0", expectedBodyLength: 5, expectedBody: "dummy"},
		// grafana config (context) priority
		{name: "grafana config wins over env var", limit: 0, ctxLimit: ptr(int64(3)), envLimit: "1000000", expectedBodyLength: 3, expectedBody: "dum", expectErr: true},
		{name: "grafana config 0 falls back to env var", limit: 0, ctxLimit: ptr(int64(0)), envLimit: "3", expectedBodyLength: 3, expectedBody: "dum", expectErr: true},
		// explicit request override priority
		{name: "request override wins over grafana config and env var", limit: 1, ctxLimit: ptr(int64(2)), requestLimit: ptr(int64(3)), envLimit: "4", expectedBodyLength: 3, expectedBody: "dum", expectErr: true},
		{name: "zero request override leaves existing limit unchanged", limit: 1, requestLimit: ptr(int64(0)), expectedBodyLength: 1, expectedBody: "d", expectErr: true},
		{name: "negative request override leaves existing limit unchanged", limit: 1, requestLimit: ptr(int64(-1)), expectedBodyLength: 1, expectedBody: "d", expectErr: true},
		{name: "no limit when nothing is set", limit: 0, expectedBodyLength: 5, expectedBody: "dummy"},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envLimit != "" {
				t.Setenv(responseLimitEnvVar, tc.envLimit)
			}

			mw := ResponseLimitMiddleware(tc.limit)
			rt := mw.CreateMiddleware(Options{}, RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Request: req, Body: io.NopCloser(strings.NewReader("dummy"))}, nil
			}))

			middlewareName, ok := mw.(MiddlewareName)
			require.True(t, ok)
			require.Equal(t, ResponseLimitMiddlewareName, middlewareName.MiddlewareName())

			ctx := context.Background()
			if tc.ctxLimit != nil {
				ctx = config.WithGrafanaConfig(ctx, config.NewGrafanaCfg(map[string]string{
					config.ResponseLimit: strconv.FormatInt(*tc.ctxLimit, 10),
				}))
			}
			if tc.requestLimit != nil {
				ctx = WithResponseLimit(ctx, *tc.requestLimit)
			}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://test.com/query", nil)
			require.NoError(t, err)

			res, err := rt.RoundTrip(req)
			require.NoError(t, err)

			bodyBytes, err := io.ReadAll(res.Body)
			require.NoError(t, res.Body.Close())

			if tc.expectErr {
				require.ErrorIs(t, err, ErrResponseBodyTooLarge)
			} else {
				require.NoError(t, err)
			}
			require.Len(t, bodyBytes, tc.expectedBodyLength)
			require.Equal(t, tc.expectedBody, string(bodyBytes))
		})
	}
}

func TestResponseLimitMiddlewarePreservesStackedLimits(t *testing.T) {
	outer := ResponseLimitMiddleware(100)
	inner := ResponseLimitMiddleware(1)
	rt, err := roundTripperFromMiddlewares(Options{}, []Middleware{outer, inner}, RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Request: req, Body: io.NopCloser(strings.NewReader("dummy"))}, nil
	}))
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodGet, "http://test.com/query", nil)
	require.NoError(t, err)

	res, err := rt.RoundTrip(req)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, res.Body.Close()) })

	body, err := io.ReadAll(res.Body)
	require.ErrorIs(t, err, ErrResponseBodyTooLarge)
	require.Equal(t, "d", string(body))
}

func TestResponseLimitOverrideDoesNotAffectOtherRequests(t *testing.T) {
	rt := ResponseLimitMiddleware(2).CreateMiddleware(Options{}, RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Request: req, Body: io.NopCloser(strings.NewReader("dummy"))}, nil
	}))
	client := &http.Client{Transport: rt}

	overrideCtx := WithResponseLimit(context.Background(), 4)
	overrideReq, err := http.NewRequestWithContext(overrideCtx, http.MethodGet, "http://test.com/search", nil)
	require.NoError(t, err)
	overrideRes, err := client.Do(overrideReq)
	require.NoError(t, err)
	overrideBody, err := io.ReadAll(overrideRes.Body)
	require.NoError(t, overrideRes.Body.Close())
	require.ErrorIs(t, err, ErrResponseBodyTooLarge)
	require.Equal(t, "dumm", string(overrideBody))

	normalReq, err := http.NewRequest(http.MethodGet, "http://test.com/resource", nil)
	require.NoError(t, err)
	normalRes, err := client.Do(normalReq)
	require.NoError(t, err)
	normalBody, err := io.ReadAll(normalRes.Body)
	require.NoError(t, normalRes.Body.Close())
	require.ErrorIs(t, err, ErrResponseBodyTooLarge)
	require.Equal(t, "du", string(normalBody))
}
