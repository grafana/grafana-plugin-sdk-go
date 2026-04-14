package httpclient

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/config"
	"github.com/stretchr/testify/require"
)

func TestResponseLimitMiddleware(t *testing.T) {
	tcs := []struct {
		limit              int64
		expectedBodyLength int
		expectedBody       string
		err                error
		envLimit           string
	}{
		// Test that the limit is set from arguments
		{limit: 1, expectedBodyLength: 1, expectedBody: "d", err: errors.New("error: http: response body too large, response limit is set to: 1"), envLimit: ""},
		{limit: 1000000, expectedBodyLength: 5, expectedBody: "dummy", err: nil, envLimit: ""},
		{limit: 0, expectedBodyLength: 5, expectedBody: "dummy", err: nil, envLimit: ""},
		// Test that the limit is set from the environment variable
		{limit: 0, expectedBodyLength: 1, expectedBody: "d", err: errors.New("error: http: response body too large, response limit is set to: 1"), envLimit: "1"},
		{limit: 0, expectedBodyLength: 5, expectedBody: "dummy", err: nil, envLimit: "1000000"},
		{limit: 0, expectedBodyLength: 5, expectedBody: "dummy", err: nil, envLimit: "-1"},
		{limit: 0, expectedBodyLength: 5, expectedBody: "dummy", err: nil, envLimit: "0"},
	}
	for _, tc := range tcs {
		t.Run(fmt.Sprintf("Test ResponseLimitMiddleware with limit: %d and envLimit: %s", tc.limit, tc.envLimit), func(t *testing.T) {
			finalRoundTripper := RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Request: req, Body: io.NopCloser(strings.NewReader("dummy"))}, nil
			})

			if tc.envLimit != "" {
				t.Setenv(responseLimitEnvVar, tc.envLimit)
			}

			mw := ResponseLimitMiddleware(tc.limit)
			rt := mw.CreateMiddleware(Options{}, finalRoundTripper)
			require.NotNil(t, rt)
			middlewareName, ok := mw.(MiddlewareName)
			require.True(t, ok)
			require.Equal(t, ResponseLimitMiddlewareName, middlewareName.MiddlewareName())

		req, err := http.NewRequest(http.MethodGet, "http://", nil)
		require.NoError(t, err)

		res, err := rt.RoundTrip(req)
		require.NoError(t, err)
		require.NotNil(t, res)

		body, err := io.ReadAll(res.Body)
		require.Error(t, err)
		require.Equal(t, "error: http: response body too large, response limit is set to: 1", err.Error())
		require.Equal(t, "d", string(body))
		require.NoError(t, res.Body.Close())
	})

	t.Run("should prefer static even when context limit is set", func(t *testing.T) {
		next := &mockRoundTripper{
			response: &http.Response{
				Body: io.NopCloser(strings.NewReader("dummy")),
			},
		}

		middleware := ResponseLimitMiddleware(1000) // High static limit
		rt := middleware.CreateMiddleware(Options{}, next)

		req, err := http.NewRequest(http.MethodGet, "http://", nil)
		require.NoError(t, err)

		// Set a lower limit in the context
		ctx := config.WithGrafanaConfig(req.Context(), config.NewGrafanaCfg(map[string]string{
			config.ResponseLimit: "1",
		}))
		req = req.WithContext(ctx)

		res, err := rt.RoundTrip(req)
		require.NoError(t, err)
		require.NotNil(t, res)

		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, "dummy", string(body))
		require.NoError(t, res.Body.Close())
	})

	t.Run("should not limit response when limit is 0", func(t *testing.T) {
		next := &mockRoundTripper{
			response: &http.Response{
				Body: io.NopCloser(strings.NewReader("dummy")),
			},
		}

		middleware := ResponseLimitMiddleware(0)
		rt := middleware.CreateMiddleware(Options{}, next)

		req, err := http.NewRequest(http.MethodGet, "http://", nil)
		require.NoError(t, err)

		res, err := rt.RoundTrip(req)
		require.NoError(t, err)
		require.NotNil(t, res)

		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, "dummy", string(body))
		require.NoError(t, res.Body.Close())
	})

	t.Run("should not limit response when status is switching protocols", func(t *testing.T) {
		next := &mockRoundTripper{
			response: &http.Response{
				StatusCode: http.StatusSwitchingProtocols,
				Body:       io.NopCloser(strings.NewReader("dummy")),
			},
		}

		middleware := ResponseLimitMiddleware(1)
		rt := middleware.CreateMiddleware(Options{}, next)

		req, err := http.NewRequest(http.MethodGet, "http://", nil)
		require.NoError(t, err)

		res, err := rt.RoundTrip(req)
		require.NoError(t, err)
		require.NotNil(t, res)

		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, "dummy", string(body))
		require.NoError(t, res.Body.Close())
	})
}

type mockRoundTripper struct {
	response *http.Response
	err      error
}

func (m *mockRoundTripper) RoundTrip(_ *http.Request) (*http.Response, error) {
	return m.response, m.err
}

func newRoundTripperWithBody(body string) http.RoundTripper {
	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Request:    req,
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})
}

func newRequestWithCtx(t *testing.T, ctx context.Context) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://test.com/query", nil)
	require.NoError(t, err)
	return req
}

func TestResponseLimitMiddlewarePriority(t *testing.T) {
	t.Run("grafana config limit is applied first", func(t *testing.T) {
		// env var is loose, context (cfg) is tight — context wins
		t.Setenv(responseLimitEnvVar, "1000000")
		rt := ResponseLimitMiddleware(0).CreateMiddleware(Options{}, newRoundTripperWithBody("dummy"))

		ctx := WithResponseLimitContext(context.Background(), 3)
		res, err := rt.RoundTrip(newRequestWithCtx(t, ctx))
		require.NoError(t, err)

		_, err = io.ReadAll(res.Body)
		require.ErrorIs(t, err, ErrResponseBodyTooLarge)
	})

	t.Run("env var is used when grafana config is not set", func(t *testing.T) {
		t.Setenv(responseLimitEnvVar, "3")
		rt := ResponseLimitMiddleware(0).CreateMiddleware(Options{}, newRoundTripperWithBody("dummy"))

		res, err := rt.RoundTrip(newRequestWithCtx(t, context.Background()))
		require.NoError(t, err)

		_, err = io.ReadAll(res.Body)
		require.ErrorIs(t, err, ErrResponseBodyTooLarge)
	})

	t.Run("limit arg is used when grafana config and env var are not set", func(t *testing.T) {
		rt := ResponseLimitMiddleware(3).CreateMiddleware(Options{}, newRoundTripperWithBody("dummy"))

		res, err := rt.RoundTrip(newRequestWithCtx(t, context.Background()))
		require.NoError(t, err)

		_, err = io.ReadAll(res.Body)
		require.ErrorIs(t, err, ErrResponseBodyTooLarge)
	})

	t.Run("grafana config 0 disables limiting even when env var is set", func(t *testing.T) {
		t.Setenv(responseLimitEnvVar, "3")
		rt := ResponseLimitMiddleware(0).CreateMiddleware(Options{}, newRoundTripperWithBody("dummy"))

		ctx := WithResponseLimitContext(context.Background(), 0)
		res, err := rt.RoundTrip(newRequestWithCtx(t, ctx))
		require.NoError(t, err)

		bodyBytes, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, "dummy", string(bodyBytes))
	})

	t.Run("no limit when nothing is set", func(t *testing.T) {
		rt := ResponseLimitMiddleware(0).CreateMiddleware(Options{}, newRoundTripperWithBody("dummy"))

		res, err := rt.RoundTrip(newRequestWithCtx(t, context.Background()))
		require.NoError(t, err)

		bodyBytes, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, "dummy", string(bodyBytes))
	})
}
