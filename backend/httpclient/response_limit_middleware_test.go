package httpclient

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T { return &v }

func TestResponseLimitMiddleware(t *testing.T) {
	tcs := []struct {
		name               string
		limit              int64
		ctxLimit           *int64
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
		{name: "grafana config 0 disables even when env var is set", limit: 0, ctxLimit: ptr(int64(0)), envLimit: "3", expectedBodyLength: 5, expectedBody: "dummy"},
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
				ctx = WithResponseLimitContext(ctx, *tc.ctxLimit)
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
