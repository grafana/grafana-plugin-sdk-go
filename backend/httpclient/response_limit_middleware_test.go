package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResponseLimitMiddleware(t *testing.T) {
	tcs := []struct {
		limit              int64
		expectedBodyLength int
		expectedBody       string
		err                error
	}{
		{limit: 1, expectedBodyLength: 1, expectedBody: "d", err: errors.New("error: http: response body too large, response limit is set to: 1")},
		{limit: 1000000, expectedBodyLength: 5, expectedBody: "dummy", err: nil},
	}
	for _, tc := range tcs {
		t.Run(fmt.Sprintf("Test ResponseLimitMiddleware with limit: %d", tc.limit), func(t *testing.T) {
			finalRoundTripper := RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Request: req, Body: io.NopCloser(strings.NewReader("dummy"))}, nil
			})

			mw := ResponseLimitMiddleware(tc.limit)
			rt := mw.CreateMiddleware(Options{}, finalRoundTripper)
			require.NotNil(t, rt)
			middlewareName, ok := mw.(MiddlewareName)
			require.True(t, ok)
			require.Equal(t, ResponseLimitMiddlewareName, middlewareName.MiddlewareName())

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://test.com/query", nil)
			require.NoError(t, err)
			res, err := rt.RoundTrip(req)
			require.NoError(t, err)
			require.NotNil(t, res)
			require.NotNil(t, res.Body)

			bodyBytes, err := io.ReadAll(res.Body)
			if err != nil {
				require.EqualError(t, tc.err, err.Error())
			} else {
				require.NoError(t, tc.err)
			}
			require.NoError(t, res.Body.Close())

			require.Len(t, bodyBytes, tc.expectedBodyLength)
			require.Equal(t, string(bodyBytes), tc.expectedBody)
		})
	}
}

func TestResponseLimitMiddlewareFallback(t *testing.T) {
	t.Run("uses env var when limit is 0", func(t *testing.T) {
		t.Setenv(responseLimitEnvVar, "3")

		finalRoundTripper := RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Request: req, Body: io.NopCloser(strings.NewReader("dummy"))}, nil
		})

		mw := ResponseLimitMiddleware(0)
		rt := mw.CreateMiddleware(Options{}, finalRoundTripper)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://test.com/query", nil)
		require.NoError(t, err)
		res, err := rt.RoundTrip(req)
		require.NoError(t, err)

		_, err = io.ReadAll(res.Body)
		require.ErrorIs(t, err, ErrResponseBodyTooLarge)
	})

	t.Run("uses 200MB default when limit is 0 and env var is not set", func(t *testing.T) {
		finalRoundTripper := RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Request: req, Body: io.NopCloser(strings.NewReader("dummy"))}, nil
		})

		mw := ResponseLimitMiddleware(0)
		rt := mw.CreateMiddleware(Options{}, finalRoundTripper)
		require.Equal(t, int64(defaultResponseLimit), resolveResponseLimit(0))

		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://test.com/query", nil)
		require.NoError(t, err)
		res, err := rt.RoundTrip(req)
		require.NoError(t, err)

		bodyBytes, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, "dummy", string(bodyBytes))
	})

	t.Run("explicit limit takes priority over env var", func(t *testing.T) {
		t.Setenv(responseLimitEnvVar, "1000000")

		finalRoundTripper := RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Request: req, Body: io.NopCloser(strings.NewReader("dummy"))}, nil
		})

		mw := ResponseLimitMiddleware(3)
		rt := mw.CreateMiddleware(Options{}, finalRoundTripper)
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://test.com/query", nil)
		require.NoError(t, err)
		res, err := rt.RoundTrip(req)
		require.NoError(t, err)

		_, err = io.ReadAll(res.Body)
		require.ErrorIs(t, err, ErrResponseBodyTooLarge)
	})
}
