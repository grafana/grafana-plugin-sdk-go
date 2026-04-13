package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLogger captures Warn calls for assertion in tests.
type mockLogger struct {
	log.Logger
	warns []mockLogCall
}

type mockLogCall struct {
	msg  string
	args []interface{}
}

func (m *mockLogger) Warn(msg string, args ...interface{}) {
	m.warns = append(m.warns, mockLogCall{msg, args})
}

func (m *mockLogger) FromContext(_ context.Context) log.Logger {
	return m
}

func newRoundTripper(body string) http.RoundTripper {
	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Request:    req,
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})
}

func newRequest(t *testing.T) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://test.com/query", nil)
	require.NoError(t, err)
	return req
}

func newRequestWithContext(t *testing.T, ctx context.Context) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://test.com/query", nil)
	require.NoError(t, err)
	return req
}

func TestResponseLimitMiddleware(t *testing.T) {
	tcs := []struct {
		limit              int64
		expectedBodyLength int
		expectedBody       string
		expectErr          bool
	}{
		{limit: 1, expectedBodyLength: 1, expectedBody: "d", expectErr: true},
		{limit: 1000000, expectedBodyLength: 5, expectedBody: "dummy", expectErr: false},
	}
	for _, tc := range tcs {
		t.Run(fmt.Sprintf("limit %d", tc.limit), func(t *testing.T) {
			mw := ResponseLimitMiddleware(tc.limit)
			rt := mw.CreateMiddleware(Options{}, newRoundTripper("dummy"))

			middlewareName, ok := mw.(MiddlewareName)
			require.True(t, ok)
			require.Equal(t, ResponseLimitMiddlewareName, middlewareName.MiddlewareName())

			res, err := rt.RoundTrip(newRequest(t))
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

func TestResponseLimitMiddlewareFallback(t *testing.T) {
	t.Run("uses env var when limit arg is 0", func(t *testing.T) {
		t.Setenv(responseLimitEnvVar, "3")

		rt := ResponseLimitMiddleware(0).CreateMiddleware(Options{}, newRoundTripper("dummy"))
		res, err := rt.RoundTrip(newRequest(t))
		require.NoError(t, err)

		_, err = io.ReadAll(res.Body)
		require.ErrorIs(t, err, ErrResponseBodyTooLarge)
	})

	t.Run("no limit when limit arg is 0 and env var is unset", func(t *testing.T) {
		rt := ResponseLimitMiddleware(0).CreateMiddleware(Options{}, newRoundTripper("dummy"))
		res, err := rt.RoundTrip(newRequest(t))
		require.NoError(t, err)

		bodyBytes, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, "dummy", string(bodyBytes))
	})

	t.Run("env var takes priority over explicit limit arg", func(t *testing.T) {
		t.Setenv(responseLimitEnvVar, "3")

		// limit arg would allow the body; env var is tighter and wins
		rt := ResponseLimitMiddleware(1000000).CreateMiddleware(Options{}, newRoundTripper("dummy"))
		res, err := rt.RoundTrip(newRequest(t))
		require.NoError(t, err)

		_, err = io.ReadAll(res.Body)
		require.ErrorIs(t, err, ErrResponseBodyTooLarge)
	})

	t.Run("explicit limit arg used when env var is unset", func(t *testing.T) {
		rt := ResponseLimitMiddleware(3).CreateMiddleware(Options{}, newRoundTripper("dummy"))
		res, err := rt.RoundTrip(newRequest(t))
		require.NoError(t, err)

		_, err = io.ReadAll(res.Body)
		require.ErrorIs(t, err, ErrResponseBodyTooLarge)
	})
}

func TestResponseLimitMiddlewareContextPriority(t *testing.T) {
	t.Run("context limit overrides explicit limit arg", func(t *testing.T) {
		// explicit limit would allow the body; context limit is tighter
		rt := ResponseLimitMiddleware(1000000).CreateMiddleware(Options{}, newRoundTripper("dummy"))

		ctx := WithResponseLimitContext(context.Background(), 3)
		res, err := rt.RoundTrip(newRequestWithContext(t, ctx))
		require.NoError(t, err)

		_, err = io.ReadAll(res.Body)
		require.ErrorIs(t, err, ErrResponseBodyTooLarge)
	})

	t.Run("context limit takes priority over env var", func(t *testing.T) {
		t.Setenv(responseLimitEnvVar, "1000000")

		// env var would allow the body; context limit is tighter and wins
		rt := ResponseLimitMiddleware(0).CreateMiddleware(Options{}, newRoundTripper("dummy"))

		ctx := WithResponseLimitContext(context.Background(), 3)
		res, err := rt.RoundTrip(newRequestWithContext(t, ctx))
		require.NoError(t, err)

		_, err = io.ReadAll(res.Body)
		require.ErrorIs(t, err, ErrResponseBodyTooLarge)
	})

	t.Run("context limit 0 disables limiting even when env var is set", func(t *testing.T) {
		t.Setenv(responseLimitEnvVar, "3")

		rt := ResponseLimitMiddleware(0).CreateMiddleware(Options{}, newRoundTripper("dummy"))

		ctx := WithResponseLimitContext(context.Background(), 0)
		res, err := rt.RoundTrip(newRequestWithContext(t, ctx))
		require.NoError(t, err)

		bodyBytes, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, "dummy", string(bodyBytes))
	})
}

func TestResponseLimitMiddlewareLogging(t *testing.T) {
	t.Run("logs warning with datasource labels when limit exceeded", func(t *testing.T) {
		logger := &mockLogger{}
		log.DefaultLogger = logger
		t.Cleanup(func() { log.DefaultLogger = log.New() })

		opts := Options{
			Labels: map[string]string{
				"datasource_uid":  "abc-123",
				"datasource_name": "My DS",
			},
		}
		rt := ResponseLimitMiddleware(3).CreateMiddleware(opts, newRoundTripper("dummy"))
		res, err := rt.RoundTrip(newRequest(t))
		require.NoError(t, err)

		_, _ = io.ReadAll(res.Body)

		require.Len(t, logger.warns, 1)
		call := logger.warns[0]
		assert.Equal(t, "downstream response body exceeded limit", call.msg)
		assert.Contains(t, call.args, "datasource_uid")
		assert.Contains(t, call.args, "abc-123")
		assert.Contains(t, call.args, "datasource_name")
		assert.Contains(t, call.args, "My DS")
		assert.Contains(t, call.args, "limit_bytes")
		assert.Contains(t, call.args, int64(3))
	})

	t.Run("logs only once across multiple reads", func(t *testing.T) {
		logger := &mockLogger{}
		log.DefaultLogger = logger
		t.Cleanup(func() { log.DefaultLogger = log.New() })

		rt := ResponseLimitMiddleware(3).CreateMiddleware(Options{}, newRoundTripper("dummy"))
		res, err := rt.RoundTrip(newRequest(t))
		require.NoError(t, err)

		// read until we hit the limit, then keep reading
		buf := make([]byte, 1)
		for i := 0; i < 10; i++ {
			_, _ = res.Body.Read(buf)
		}

		require.Len(t, logger.warns, 1)
	})

	t.Run("does not log when body is within limit", func(t *testing.T) {
		logger := &mockLogger{}
		log.DefaultLogger = logger
		t.Cleanup(func() { log.DefaultLogger = log.New() })

		rt := ResponseLimitMiddleware(1000000).CreateMiddleware(Options{}, newRoundTripper("dummy"))
		res, err := rt.RoundTrip(newRequest(t))
		require.NoError(t, err)

		_, err = io.ReadAll(res.Body)
		require.NoError(t, err)
		require.Empty(t, logger.warns)
	})

	t.Run("does not log when limiting is disabled via context", func(t *testing.T) {
		logger := &mockLogger{}
		log.DefaultLogger = logger
		t.Cleanup(func() { log.DefaultLogger = log.New() })

		rt := ResponseLimitMiddleware(0).CreateMiddleware(Options{}, newRoundTripper("dummy"))

		ctx := WithResponseLimitContext(context.Background(), 0)
		res, err := rt.RoundTrip(newRequestWithContext(t, ctx))
		require.NoError(t, err)

		_, err = io.ReadAll(res.Body)
		require.NoError(t, err)
		require.Empty(t, logger.warns)
	})
}

func TestResponseLimitMiddlewareStatusCodes(t *testing.T) {
	t.Run("does not wrap body for 101 Switching Protocols", func(t *testing.T) {
		rt := ResponseLimitMiddleware(1).CreateMiddleware(Options{}, RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusSwitchingProtocols,
				Request:    req,
				Body:       io.NopCloser(strings.NewReader("dummy")),
			}, nil
		}))

		res, err := rt.RoundTrip(newRequest(t))
		require.NoError(t, err)
		require.Equal(t, http.StatusSwitchingProtocols, res.StatusCode)

		// body should not be wrapped — no limit error despite limit=1
		bodyBytes, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, "dummy", string(bodyBytes))
	})

	t.Run("wraps body for normal 200 response", func(t *testing.T) {
		rt := ResponseLimitMiddleware(3).CreateMiddleware(Options{}, newRoundTripper("dummy"))
		res, err := rt.RoundTrip(newRequest(t))
		require.NoError(t, err)

		_, err = io.ReadAll(res.Body)
		require.ErrorIs(t, err, ErrResponseBodyTooLarge)
	})
}

func TestParseEnvResponseLimit(t *testing.T) {
	tcs := []struct {
		name     string
		envVar   string
		expected int64
	}{
		{name: "parses valid positive value", envVar: "1024", expected: 1024},
		{name: "returns 0 when unset", expected: 0},
		{name: "returns 0 for invalid value", envVar: "notanumber", expected: 0},
		{name: "returns 0 when env var is 0", envVar: "0", expected: 0},
		{name: "returns 0 when env var is negative", envVar: "-1", expected: 0},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envVar != "" {
				t.Setenv(responseLimitEnvVar, tc.envVar)
			}
			require.Equal(t, tc.expected, parseEnvResponseLimit())
		})
	}
}

func TestResolveResponseLimit(t *testing.T) {
	tcs := []struct {
		name     string
		envLimit int64
		limit    int64
		ctxLimit *int64
		expected int64
	}{
		{name: "context wins over env var and limit arg", envLimit: 100, limit: 999, ctxLimit: ptr(int64(50)), expected: 50},
		{name: "context wins over env var", envLimit: 100, ctxLimit: ptr(int64(50)), expected: 50},
		{name: "env var wins over limit arg", envLimit: 100, limit: 999, expected: 100},
		{name: "context used when env var set", envLimit: 100, limit: 999, ctxLimit: ptr(int64(50)), expected: 50},
		{name: "context 0 disables even when env var is set", envLimit: 100, ctxLimit: ptr(int64(0)), expected: 0},
		{name: "limit arg used when env var and context unset", limit: 500, expected: 500},
		{name: "no limit when all unset", expected: 0},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			if tc.ctxLimit != nil {
				ctx = WithResponseLimitContext(ctx, *tc.ctxLimit)
			}
			require.Equal(t, tc.expected, resolveResponseLimit(tc.envLimit, tc.limit, ctx))
		})
	}
}

func ptr[T any](v T) *T { return &v }

func TestResponseLimitMiddlewareErrors(t *testing.T) {
	t.Run("propagates round trip error without wrapping body", func(t *testing.T) {
		expectedErr := errors.New("connection refused")
		rt := ResponseLimitMiddleware(100).CreateMiddleware(Options{}, RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			return nil, expectedErr
		}))

		res, err := rt.RoundTrip(newRequest(t))
		require.ErrorIs(t, err, expectedErr)
		require.Nil(t, res)
	})
}
