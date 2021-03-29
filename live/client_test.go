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
		require.Equal(t, r.URL.Path, "/api/live/publish")
		_, _ = fmt.Fprintln(w, []byte(`{}`))
	}))
	defer ts.Close()

	c, err := NewClient(ts.URL)
	require.NoError(t, err)

	_, err = c.Publish(context.Background(), "test", json.RawMessage(`{}`))
	require.NoError(t, err)
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
