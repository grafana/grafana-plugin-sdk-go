package httplogger_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e/storage"
	httplogger "github.com/grafana/grafana-plugin-sdk-go/experimental/http_logger"
	"github.com/stretchr/testify/require"
)

func TestHTTPLogger(t *testing.T) {
	t.Run("saved file should match example", func(t *testing.T) {
		c, f := setup(true)
		res, err := c.Get("http://example.com")
		require.NoError(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusOK, res.StatusCode)
		b, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, "OK", string(b))
		expected := storage.NewHARStorage("testdata/example.har")
		actual := storage.NewHARStorage(f.Name())
		require.Equal(t, 1, len(actual.Entries()))
		require.Equal(t, expected.Entries()[0].Request, actual.Entries()[0].Request)
		require.Equal(t, expected.Entries()[0].Response, actual.Entries()[0].Response)
		har, err := os.ReadFile(f.Name())
		require.NoError(t, err)
		require.Greater(t, len(har), 0)
	})

	t.Run("file should not be created by storage if http logging is disabled", func(t *testing.T) {
		c, f := setup(false)
		res, err := c.Get("http://example.com")
		require.NoError(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusOK, res.StatusCode)
		b, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, "OK", string(b))
		actual := storage.NewHARStorage(f.Name())
		require.Equal(t, 0, len(actual.Entries()))
		_, err = os.Stat(f.Name())
		require.Equal(t, true, errors.Is(err, os.ErrNotExist))
	})

	t.Run("should set path and enabled overrides", func(t *testing.T) {
		// ensure env variables are not set
		os.Setenv(httplogger.PluginHARLogEnabledEnv, "false")
		os.Setenv(httplogger.PluginHARLogPathEnv, "")

		f, err := os.CreateTemp("", "test_*.har")
		defer os.Remove(f.Name())
		require.NoError(t, err)
		h := httplogger.NewHTTPLogger("example-plugin-id", &fakeRoundTripper{}, httplogger.Options{
			Path:      f.Name(),
			EnabledFn: func() bool { return true },
		})
		c := &http.Client{
			Transport: h,
			Timeout:   time.Second * 30,
		}
		res, err := c.Get("http://example.com")
		require.NoError(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusOK, res.StatusCode)
		b, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.Equal(t, "OK", string(b))
		expected := storage.NewHARStorage("testdata/example.har")
		actual := storage.NewHARStorage(f.Name())
		require.Equal(t, 1, len(actual.Entries()))
		require.Equal(t, expected.Entries()[0].Request, actual.Entries()[0].Request)
		require.Equal(t, expected.Entries()[0].Response, actual.Entries()[0].Response)
		har, err := os.ReadFile(f.Name())
		require.NoError(t, err)
		require.Greater(t, len(har), 0)
	})
}

func setup(enabled bool) (*http.Client, *os.File) {
	f, err := os.CreateTemp("", "example_*.har")
	defer os.Remove(f.Name())
	if err != nil {
		panic(err)
	}

	if enabled {
		os.Setenv(httplogger.PluginHARLogEnabledEnv, "true")
		os.Setenv(httplogger.PluginHARLogPathEnv, f.Name())
	} else {
		os.Setenv(httplogger.PluginHARLogEnabledEnv, "false")
	}

	h := httplogger.NewHTTPLogger("example-plugin-id", &fakeRoundTripper{})

	return &http.Client{
		Transport: h,
		Timeout:   time.Second * 30,
	}, f
}

type fakeRoundTripper struct{}

func (hl *fakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString("OK")),
	}, nil
}
