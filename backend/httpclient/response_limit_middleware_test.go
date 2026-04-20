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
	t.Run("should use static limit when no context limit is set", func(t *testing.T) {
		next := &mockRoundTripper{
			response: &http.Response{
				Body: io.NopCloser(strings.NewReader("dummy")),
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
