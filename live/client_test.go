package live

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClient_Publish(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
		require.Equal(t, r.URL.Path, "/api/live/publish")
		_, _ = fmt.Fprintln(w, []byte(`{}`))
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL, WithAPIKey("test-api-key"))
	require.NoError(t, err)

	_, err = c.Publish(context.Background(), "test", json.RawMessage(`{}`))
	require.NoError(t, err)
}

func TestClient_Publish_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL, WithAPIKey("test-api-key"))
	require.NoError(t, err)

	_, err = c.Publish(context.Background(), "test", json.RawMessage(`{}`))
	require.Error(t, err)
	var statusCodeError *StatusCodeError
	require.ErrorAs(t, err, &statusCodeError)
	require.Equal(t, http.StatusInternalServerError, statusCodeError.Code)
}

func TestWithHTTPClient(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, r.URL.Path, "/api/live/publish")
		time.Sleep(50 * time.Millisecond)
		_, _ = fmt.Fprintln(w, []byte(`{}`))
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL, WithHTTPClient(&http.Client{Timeout: 10 * time.Microsecond}))
	require.NoError(t, err)

	_, err = c.Publish(context.Background(), "test", json.RawMessage(`{}`))
	require.Error(t, err)
	require.True(t, os.IsTimeout(err))
}
