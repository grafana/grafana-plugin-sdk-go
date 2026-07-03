package schemabuilder

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/pluginschema"
)

type watchlistSpec struct {
	// Title is the human readable name.
	Title string `json:"title"`

	// Patterns are the match patterns this watchlist enforces.
	Patterns []string `json:"patterns"`

	// Severity is one of "info", "warn", "crit".
	Severity string `json:"severity"`
}

type clusterRuleSpec struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type watchlistStatus struct {
	// MatchCount is how many objects currently match the patterns.
	MatchCount int `json:"matchCount"`

	// LastEvaluated is when the patterns were last evaluated.
	LastEvaluated string `json:"lastEvaluated"`
}

func newTestBuilder(t *testing.T) *Builder {
	t.Helper()
	b, err := NewSchemaBuilder(BuilderOptions{
		PluginID: []string{"test"},
	})
	require.NoError(t, err)
	return b
}

func TestAddStoredObjects_Defaults(t *testing.T) {
	b := newTestBuilder(t)

	err := b.AddStoredObjects([]StoredObjectInfo{{
		Name:     "Watchlist",
		SpecType: reflect.TypeOf(watchlistSpec{}),
	}})
	require.NoError(t, err)

	require.NotNil(t, b.storedObjects)
	require.Len(t, b.storedObjects.Items, 1)

	got := b.storedObjects.Items[0]
	require.Equal(t, "Watchlist", got.Name)
	require.Equal(t, "watchlists", got.Plural)
	require.Equal(t, "watchlist", got.Singular)
	require.Equal(t, pluginschema.StoredObjectScope(""), got.Scope)
	require.NotNil(t, got.Spec)
	require.Contains(t, got.Spec.Properties, "title")
	require.Contains(t, got.Spec.Properties, "patterns")
	require.Contains(t, got.Spec.Properties, "severity")
}

func TestAddStoredObjects_RespectsExplicitFields(t *testing.T) {
	b := newTestBuilder(t)

	err := b.AddStoredObjects([]StoredObjectInfo{{
		Name:       "ClusterRule",
		Plural:     "clusterrules",
		Singular:   "clusterrule",
		Scope:      pluginschema.ScopeCluster,
		SpecType:   reflect.TypeOf(clusterRuleSpec{}),
		Validation: []pluginschema.Operation{pluginschema.OperationCreate, pluginschema.OperationUpdate},
		Mutation:   []pluginschema.Operation{pluginschema.OperationDelete},
	}})
	require.NoError(t, err)
	require.Len(t, b.storedObjects.Items, 1)

	got := b.storedObjects.Items[0]
	require.Equal(t, "ClusterRule", got.Name)
	require.Equal(t, "clusterrules", got.Plural)
	require.Equal(t, "clusterrule", got.Singular)
	require.Equal(t, pluginschema.ScopeCluster, got.Scope)
	require.Equal(t, []pluginschema.Operation{pluginschema.OperationCreate, pluginschema.OperationUpdate}, got.Validation)
	require.Equal(t, []pluginschema.Operation{pluginschema.OperationDelete}, got.Mutation)
}

func TestAddStoredObjects_StatusSchema(t *testing.T) {
	b := newTestBuilder(t)

	err := b.AddStoredObjects([]StoredObjectInfo{
		{
			Name:       "Watchlist",
			SpecType:   reflect.TypeOf(watchlistSpec{}),
			StatusType: reflect.TypeOf(watchlistStatus{}),
		},
		{
			Name:     "ClusterRule",
			SpecType: reflect.TypeOf(clusterRuleSpec{}),
		},
	})
	require.NoError(t, err)
	require.Len(t, b.storedObjects.Items, 2)

	withStatus := b.storedObjects.Items[0]
	require.NotNil(t, withStatus.Status)
	require.Contains(t, withStatus.Status.Properties, "matchCount")
	require.Contains(t, withStatus.Status.Properties, "lastEvaluated")

	withoutStatus := b.storedObjects.Items[1]
	require.Nil(t, withoutStatus.Status)

	raw, err := json.Marshal(b.storedObjects)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"matchCount"`)

	var items []map[string]json.RawMessage
	var list map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &list))
	require.NoError(t, json.Unmarshal(list["items"], &items))
	require.Contains(t, items[0], "status")
	require.NotContains(t, items[1], "status")
}

func TestAddStoredObjects_Append(t *testing.T) {
	b := newTestBuilder(t)

	require.NoError(t, b.AddStoredObjects([]StoredObjectInfo{{
		Name:     "Watchlist",
		SpecType: reflect.TypeOf(watchlistSpec{}),
	}}))
	require.NoError(t, b.AddStoredObjects([]StoredObjectInfo{{
		Name:     "ClusterRule",
		Scope:    pluginschema.ScopeCluster,
		SpecType: reflect.TypeOf(clusterRuleSpec{}),
	}}))

	require.Len(t, b.storedObjects.Items, 2)
	require.Equal(t, "Watchlist", b.storedObjects.Items[0].Name)
	require.Equal(t, "ClusterRule", b.storedObjects.Items[1].Name)
}

func TestAddStoredObjects_ValidationErrors(t *testing.T) {
	t.Run("missing name", func(t *testing.T) {
		b := newTestBuilder(t)
		err := b.AddStoredObjects([]StoredObjectInfo{{
			SpecType: reflect.TypeOf(watchlistSpec{}),
		}})
		require.ErrorContains(t, err, "missing name")
	})

	t.Run("missing spec type", func(t *testing.T) {
		b := newTestBuilder(t)
		err := b.AddStoredObjects([]StoredObjectInfo{{Name: "Watchlist"}})
		require.ErrorContains(t, err, "missing SpecType")
	})
}

func TestPluginSchema_RoundTrip_WithStoredObjects(t *testing.T) {
	b := newTestBuilder(t)
	require.NoError(t, b.AddStoredObjects([]StoredObjectInfo{{
		Name:     "Watchlist",
		SpecType: reflect.TypeOf(watchlistSpec{}),
	}}))

	src := &pluginschema.PluginSchema{
		TargetAPIVersion: "v0alpha1",
		StoredObjects:    b.storedObjects,
	}

	raw, err := json.Marshal(src)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"storedObjects"`)
	require.Contains(t, string(raw), `"Watchlist"`)
	require.Contains(t, string(raw), `"watchlists"`)

	var dst pluginschema.PluginSchema
	require.NoError(t, pluginschema.Load(raw, &dst))
	require.NotNil(t, dst.StoredObjects)
	require.Len(t, dst.StoredObjects.Items, 1)
	require.Equal(t, "Watchlist", dst.StoredObjects.Items[0].Name)
	require.NotNil(t, dst.StoredObjects.Items[0].Spec)
	require.Contains(t, dst.StoredObjects.Items[0].Spec.Properties, "title")

	require.False(t, src.IsZero())

	empty := &pluginschema.PluginSchema{TargetAPIVersion: "v0alpha1"}
	require.True(t, empty.IsZero())
}
