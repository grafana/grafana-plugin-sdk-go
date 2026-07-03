package storedobjects

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/config"
)

type watchlistSpec struct {
	Title string `json:"title"`
}

type watchlistStatus struct {
	State   string `json:"state"`
	Message string `json:"message,omitempty"`
}

func TestPluralOf(t *testing.T) {
	require.Equal(t, "watchlists", PluralOf("Watchlist"))
	require.Equal(t, "clusterrules", PluralOf("ClusterRule"))
}

func TestNewClient(t *testing.T) {
	t.Run("requires AppURL", func(t *testing.T) {
		_, err := NewClient(ClientOpts{Token: "t", Group: "g", OrgNamespace: "default"})
		require.ErrorContains(t, err, "AppURL")
	})

	t.Run("requires Token", func(t *testing.T) {
		_, err := NewClient(ClientOpts{AppURL: "http://g", Group: "g", OrgNamespace: "default"})
		require.ErrorContains(t, err, "Token")
	})

	t.Run("requires Group", func(t *testing.T) {
		_, err := NewClient(ClientOpts{AppURL: "http://g", Token: "t", OrgNamespace: "default"})
		require.ErrorContains(t, err, "Group")
	})

	t.Run("requires OrgNamespace", func(t *testing.T) {
		_, err := NewClient(ClientOpts{AppURL: "http://g", Token: "t", Group: "g"})
		require.ErrorContains(t, err, "OrgNamespace")
	})

	t.Run("defaults version and trims trailing slash", func(t *testing.T) {
		c, err := NewClient(ClientOpts{AppURL: "http://grafana:3000/", Token: "t", Group: "my-app", OrgNamespace: "default"})
		require.NoError(t, err)
		require.Equal(t, "http://grafana:3000", c.baseURL)
		require.Equal(t, "v0alpha1", c.version)
		require.NotNil(t, c.httpClient)
	})
}

func TestNewClientFromContext(t *testing.T) {
	grafanaCtx := func(t *testing.T) context.Context {
		t.Helper()
		return config.WithGrafanaConfig(context.Background(), config.NewGrafanaCfg(map[string]string{
			config.AppURL:          "http://grafana:3000/",
			config.AppClientSecret: "sa-token",
		}))
	}

	t.Run("resolves group and namespace from the plugin context", func(t *testing.T) {
		ctx := backend.WithPluginContext(grafanaCtx(t), backend.PluginContext{
			PluginID:  "my-app",
			Namespace: "stacks-123",
		})
		c, err := NewClientFromContext(ctx)
		require.NoError(t, err)
		require.Equal(t, "http://grafana:3000", c.baseURL)
		require.Equal(t, "sa-token", c.token)
		require.Equal(t, "my-app", c.group)
		require.Equal(t, "v0alpha1", c.version)
		require.Equal(t, "stacks-123", c.orgNamespace)
	})

	t.Run("falls back to org-derived namespace", func(t *testing.T) {
		ctx := backend.WithPluginContext(grafanaCtx(t), backend.PluginContext{
			PluginID: "my-app",
			OrgID:    1,
		})
		c, err := NewClientFromContext(ctx)
		require.NoError(t, err)
		require.Equal(t, "default", c.orgNamespace)

		ctx = backend.WithPluginContext(grafanaCtx(t), backend.PluginContext{
			PluginID: "my-app",
			OrgID:    42,
		})
		c, err = NewClientFromContext(ctx)
		require.NoError(t, err)
		require.Equal(t, "org-42", c.orgNamespace)
	})

	t.Run("errors when token is missing", func(t *testing.T) {
		ctx := config.WithGrafanaConfig(context.Background(), config.NewGrafanaCfg(map[string]string{
			config.AppURL: "http://grafana:3000",
		}))
		ctx = backend.WithPluginContext(ctx, backend.PluginContext{PluginID: "my-app", OrgID: 1})
		_, err := NewClientFromContext(ctx)
		require.ErrorContains(t, err, "PluginAppClientSecret")
	})
}

// newTestCollection starts an httptest server with the given handler and
// returns a typed collection pointed at it.
func newTestCollection(t *testing.T, namespace string, handler http.HandlerFunc) *Collection[watchlistSpec, watchlistStatus] {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c, err := NewClient(ClientOpts{AppURL: srv.URL, Token: "test-token", Group: "my-app", OrgNamespace: namespace})
	require.NoError(t, err)
	return NewCollection[watchlistSpec, watchlistStatus](c, "Watchlist")
}

func requireCommonHeaders(t *testing.T, r *http.Request) {
	t.Helper()
	require.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
	require.Equal(t, "application/json", r.Header.Get("Accept"))
}

func TestCollectionList(t *testing.T) {
	coll := newTestCollection(t, "default", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/apis/my-app/v0alpha1/namespaces/default/watchlists", r.URL.Path)
		requireCommonHeaders(t, r)
		_, _ = w.Write([]byte(`{
			"apiVersion": "my-app/v0alpha1",
			"kind": "WatchlistList",
			"items": [
				{"apiVersion": "my-app/v0alpha1", "kind": "Watchlist", "metadata": {"name": "one", "labels": {"team": "a"}}, "spec": {"title": "One"}},
				{"apiVersion": "my-app/v0alpha1", "kind": "Watchlist", "metadata": {"name": "two"}, "spec": {"title": "Two"}, "status": {"state": "ok"}}
			]
		}`))
	})

	items, err := coll.List(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, "one", items[0].Name)
	require.Equal(t, map[string]string{"team": "a"}, items[0].Labels)
	require.Equal(t, "One", items[0].Spec.Title)
	require.Equal(t, "Two", items[1].Spec.Title)
	require.Equal(t, "ok", items[1].Status.State)
}

func TestCollectionGet(t *testing.T) {
	coll := newTestCollection(t, "org-2", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/apis/my-app/v0alpha1/namespaces/org-2/watchlists/my-list", r.URL.Path)
		requireCommonHeaders(t, r)
		_, _ = w.Write([]byte(`{"apiVersion": "my-app/v0alpha1", "kind": "Watchlist", "metadata": {"name": "my-list", "namespace": "org-2"}, "spec": {"title": "Mine"}}`))
	})

	item, err := coll.Get(context.Background(), "my-list")
	require.NoError(t, err)
	require.Equal(t, "my-list", item.Name)
	require.Equal(t, "Mine", item.Spec.Title)
}

func TestCollectionWriteStatus(t *testing.T) {
	coll := newTestCollection(t, "default", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPatch, r.Method)
		require.Equal(t, "/apis/my-app/v0alpha1/namespaces/default/watchlists/my-list/status", r.URL.Path)
		requireCommonHeaders(t, r)
		require.Equal(t, "application/merge-patch+json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.JSONEq(t, `{"status": {"state": "ok", "message": "reconciled"}}`, string(body))

		_, _ = w.Write([]byte(`{"apiVersion": "my-app/v0alpha1", "kind": "Watchlist", "metadata": {"name": "my-list"}, "status": {"state": "ok", "message": "reconciled"}}`))
	})

	err := coll.WriteStatus(context.Background(), "my-list", watchlistStatus{State: "ok", Message: "reconciled"})
	require.NoError(t, err)
}

func TestCollectionErrorResponse(t *testing.T) {
	coll := newTestCollection(t, "default", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"kind": "Status", "message": "watchlists is forbidden"}`))
	})

	_, err := coll.Get(context.Background(), "my-list")
	require.Error(t, err)
	require.ErrorContains(t, err, "403")
	require.ErrorContains(t, err, "watchlists is forbidden")

	_, err = coll.List(context.Background())
	require.ErrorContains(t, err, "403")

	err = coll.WriteStatus(context.Background(), "my-list", watchlistStatus{State: "ok"})
	require.ErrorContains(t, err, "403")
}

func TestCollectionErrorBodyTruncated(t *testing.T) {
	coll := newTestCollection(t, "default", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		body := make([]byte, 4096)
		for i := range body {
			body[i] = 'x'
		}
		_, _ = w.Write(body)
	})

	_, err := coll.Get(context.Background(), "my-list")
	require.Error(t, err)
	// The 512-byte cap plus the fixed prefix keeps the error well under the
	// full 4096-byte body.
	require.Less(t, len(err.Error()), 1024)
}
