package storedobjects

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/config"
)

func TestNewClient(t *testing.T) {
	t.Run("requires AppURL", func(t *testing.T) {
		_, err := NewClient(ClientOpts{Token: "t", Group: "g"})
		require.ErrorContains(t, err, "AppURL")
	})

	t.Run("requires Token", func(t *testing.T) {
		_, err := NewClient(ClientOpts{AppURL: "http://g", Group: "g"})
		require.ErrorContains(t, err, "Token")
	})

	t.Run("requires Group", func(t *testing.T) {
		_, err := NewClient(ClientOpts{AppURL: "http://g", Token: "t"})
		require.ErrorContains(t, err, "Group")
	})

	t.Run("defaults version and trims trailing slash", func(t *testing.T) {
		c, err := NewClient(ClientOpts{AppURL: "http://grafana:3000/", Token: "t", Group: "my-app"})
		require.NoError(t, err)
		require.Equal(t, "http://grafana:3000", c.baseURL)
		require.Equal(t, "v0alpha1", c.version)
		require.NotNil(t, c.httpClient)
	})
}

func TestNewClientFromContext(t *testing.T) {
	t.Run("reads app URL and token from Grafana config", func(t *testing.T) {
		ctx := config.WithGrafanaConfig(context.Background(), config.NewGrafanaCfg(map[string]string{
			config.AppURL:          "http://grafana:3000/",
			config.AppClientSecret: "sa-token",
		}))
		c, err := NewClientFromContext(ctx, "my-app")
		require.NoError(t, err)
		require.Equal(t, "http://grafana:3000", c.baseURL)
		require.Equal(t, "sa-token", c.token)
		require.Equal(t, "my-app", c.group)
		require.Equal(t, "v0alpha1", c.version)
	})

	t.Run("errors when token is missing", func(t *testing.T) {
		ctx := config.WithGrafanaConfig(context.Background(), config.NewGrafanaCfg(map[string]string{
			config.AppURL: "http://grafana:3000",
		}))
		_, err := NewClientFromContext(ctx, "my-app")
		require.ErrorContains(t, err, "PluginAppClientSecret")
	})
}

// newTestClient starts an httptest server with the given handler and returns
// a client pointed at it.
func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := NewClient(ClientOpts{AppURL: srv.URL, Token: "test-token", Group: "my-app"})
	require.NoError(t, err)
	return c
}

func requireCommonHeaders(t *testing.T, r *http.Request) {
	t.Helper()
	require.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
	require.Equal(t, "application/json", r.Header.Get("Accept"))
}

func TestClientList(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/apis/my-app/v0alpha1/namespaces/default/watchlists", r.URL.Path)
		requireCommonHeaders(t, r)
		_, _ = w.Write([]byte(`{
			"apiVersion": "my-app/v0alpha1",
			"kind": "WatchlistList",
			"items": [
				{"apiVersion": "my-app/v0alpha1", "kind": "Watchlist", "metadata": {"name": "one", "namespace": "default", "resourceVersion": "5", "generation": 2}, "spec": {"title": "One"}},
				{"apiVersion": "my-app/v0alpha1", "kind": "Watchlist", "metadata": {"name": "two"}, "spec": {"title": "Two"}, "status": {"state": "ok"}}
			]
		}`))
	})

	list, err := c.List(context.Background(), "default", "watchlists")
	require.NoError(t, err)
	require.Len(t, list.Items, 2)
	require.Equal(t, "one", list.Items[0].Metadata.Name)
	require.Equal(t, "default", list.Items[0].Metadata.Namespace)
	require.Equal(t, "5", list.Items[0].Metadata.ResourceVersion)
	require.Equal(t, int64(2), list.Items[0].Metadata.Generation)
	require.JSONEq(t, `{"title": "Two"}`, string(list.Items[1].Spec))
	require.JSONEq(t, `{"state": "ok"}`, string(list.Items[1].Status))
}

func TestClientGet(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/apis/my-app/v0alpha1/namespaces/org-2/watchlists/my-list", r.URL.Path)
		requireCommonHeaders(t, r)
		_, _ = w.Write([]byte(`{"apiVersion": "my-app/v0alpha1", "kind": "Watchlist", "metadata": {"name": "my-list", "namespace": "org-2"}, "spec": {"title": "Mine"}}`))
	})

	obj, err := c.Get(context.Background(), "org-2", "watchlists", "my-list")
	require.NoError(t, err)
	require.Equal(t, "Watchlist", obj.Kind)
	require.Equal(t, "my-list", obj.Metadata.Name)

	var spec struct {
		Title string `json:"title"`
	}
	require.NoError(t, obj.SpecInto(&spec))
	require.Equal(t, "Mine", spec.Title)
}

func TestClientUpdateStatus(t *testing.T) {
	type status struct {
		State   string `json:"state"`
		Message string `json:"message,omitempty"`
	}

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPatch, r.Method)
		require.Equal(t, "/apis/my-app/v0alpha1/namespaces/default/watchlists/my-list/status", r.URL.Path)
		requireCommonHeaders(t, r)
		require.Equal(t, "application/merge-patch+json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.JSONEq(t, `{"status": {"state": "ok", "message": "reconciled"}}`, string(body))

		_, _ = w.Write([]byte(`{"apiVersion": "my-app/v0alpha1", "kind": "Watchlist", "metadata": {"name": "my-list", "resourceVersion": "6"}, "status": {"state": "ok", "message": "reconciled"}}`))
	})

	obj, err := c.UpdateStatus(context.Background(), "default", "watchlists", "my-list", status{State: "ok", Message: "reconciled"})
	require.NoError(t, err)
	require.Equal(t, "6", obj.Metadata.ResourceVersion)

	var got status
	require.NoError(t, obj.StatusInto(&got))
	require.Equal(t, status{State: "ok", Message: "reconciled"}, got)
}

func TestClientErrorResponse(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"kind": "Status", "message": "watchlists is forbidden"}`))
	})

	_, err := c.Get(context.Background(), "default", "watchlists", "my-list")
	require.Error(t, err)
	require.ErrorContains(t, err, "403")
	require.ErrorContains(t, err, "watchlists is forbidden")

	_, err = c.List(context.Background(), "default", "watchlists")
	require.ErrorContains(t, err, "403")

	_, err = c.UpdateStatus(context.Background(), "default", "watchlists", "my-list", map[string]string{"state": "ok"})
	require.ErrorContains(t, err, "403")
}

func TestClientErrorBodyTruncated(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		body := make([]byte, 4096)
		for i := range body {
			body[i] = 'x'
		}
		_, _ = w.Write(body)
	})

	_, err := c.Get(context.Background(), "default", "watchlists", "my-list")
	require.Error(t, err)
	// The 512-byte cap plus the fixed prefix keeps the error well under the
	// full 4096-byte body.
	require.Less(t, len(err.Error()), 1024)
}

func TestObjectSpecStatusInto(t *testing.T) {
	obj := &Object{
		Spec:   json.RawMessage(`{"title": "hello", "count": 3}`),
		Status: json.RawMessage(`{"state": "ok"}`),
	}

	var spec struct {
		Title string `json:"title"`
		Count int    `json:"count"`
	}
	require.NoError(t, obj.SpecInto(&spec))
	require.Equal(t, "hello", spec.Title)
	require.Equal(t, 3, spec.Count)

	var st struct {
		State string `json:"state"`
	}
	require.NoError(t, obj.StatusInto(&st))
	require.Equal(t, "ok", st.State)

	empty := &Object{}
	require.ErrorContains(t, empty.SpecInto(&spec), "no spec")
	require.ErrorContains(t, empty.StatusInto(&st), "no status")
}

func TestNamespaceForOrgID(t *testing.T) {
	require.Equal(t, "default", NamespaceForOrgID(1))
	require.Equal(t, "org-2", NamespaceForOrgID(2))
	require.Equal(t, "org-42", NamespaceForOrgID(42))
}
