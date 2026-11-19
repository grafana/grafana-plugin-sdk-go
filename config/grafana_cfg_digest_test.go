package config

import (
	"fmt"
	"maps"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend/proxy"
)

func TestGrafanaCfgDigestEqual(t *testing.T) {
	base := map[string]string{"a": "1", "b": "2"}
	tests := []struct {
		name  string
		other map[string]string
	}{
		{name: "same values", other: map[string]string{"a": "1", "b": "2"}},
		{name: "value changed", other: map[string]string{"a": "1", "b": "3"}},
		{name: "key added", other: map[string]string{"a": "1", "b": "2", "c": "3"}},
		{name: "key removed", other: map[string]string{"a": "1"}},
		{name: "values swapped", other: map[string]string{"a": "2", "b": "1"}},
		{name: "key and value boundary moved", other: map[string]string{"a1": "", "b": "2"}},
		{name: "empty value", other: map[string]string{"a": "", "b": "2"}},
		{name: "empty map", other: map[string]string{}},
		{name: "nil map", other: nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			requireDigestMatchesMaps(t, base, tc.other)
		})
	}

	t.Run("nil and empty maps", func(t *testing.T) {
		requireDigestMatchesMaps(t, nil, map[string]string{})
	})

	t.Run("nil config", func(t *testing.T) {
		var c *GrafanaCfg
		require.Nil(t, c.Digest())
		require.False(t, NewGrafanaCfg(base).Digest().Equal(nil))
		require.False(t, c.Equal(NewGrafanaCfg(base).Digest()))
		require.ElementsMatch(t, []string{"a", "b"}, c.Diff(NewGrafanaCfg(base).Digest()))
		require.ElementsMatch(t, []string{"a", "b"}, NewGrafanaCfg(base).Digest().Diff(nil))
	})
}

func TestGrafanaCfgDigestMatchesFullConfig(t *testing.T) {
	keys := []string{
		"", "a", "b", "ab", AppURL,
		proxy.PluginSecureSocksProxyRootCAsContents,
		proxy.PluginSecureSocksProxyRootCAs,    // nolint:staticcheck
		proxy.PluginSecureSocksProxyClientCert, // nolint:staticcheck
	}
	values := []string{"", "1", "2", "12", "https://grafana.example.com"}
	rng := rand.New(rand.NewPCG(3, 4)) //nolint:gosec // G404: deterministic test data
	randomMap := func() map[string]string {
		m := map[string]string{}
		for range rng.IntN(len(keys) + 1) {
			m[keys[rng.IntN(len(keys))]] = values[rng.IntN(len(values))]
		}
		return m
	}
	for i := range 500 {
		m1 := randomMap()
		m2 := maps.Clone(m1)
		if rng.IntN(2) == 0 {
			m2 = randomMap()
		}
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			requireDigestMatchesMaps(t, m1, m2)
		})
	}
}

func requireDigestMatchesMaps(t *testing.T, m1, m2 map[string]string) {
	t.Helper()
	c1, c2 := NewGrafanaCfg(m1), NewGrafanaCfg(maps.Clone(m2))
	want := c1.Equal(c2)
	require.Equal(t, want, c2.Equal(c1), "full and full, reversed")
	require.Equal(t, want, c1.Digest().Equal(c2), "digest and full")
	require.Equal(t, want, c2.Equal(c1.Digest()), "full and digest")
	require.Equal(t, want, c1.Digest().Equal(c2.Digest()), "digest and digest")

	wantDiff := c1.Diff(c2)
	require.ElementsMatch(t, wantDiff, c1.Digest().Diff(c2), "digest and full")
	require.ElementsMatch(t, wantDiff, c2.Diff(c1.Digest()), "full and digest")
	require.ElementsMatch(t, wantDiff, c1.Digest().Diff(c2.Digest()), "digest and digest")
}

func TestGrafanaCfgDigestKeepsNoValues(t *testing.T) {
	c := NewGrafanaCfg(map[string]string{AppURL: "https://grafana.example.com"})
	d := c.Digest()
	require.Empty(t, d.Config())
	require.Empty(t, d.Get(AppURL))
	require.True(t, d.Equal(c))
	require.Same(t, d, d.Digest())
}

func TestGrafanaCfgDigestEqualAllocs(t *testing.T) {
	m := map[string]string{AppURL: "https://grafana.example.com", ConcurrentQueryCount: "10", SQLRowLimit: "1000000"}
	c := NewGrafanaCfg(m)
	d := c.Digest()
	allocs := testing.AllocsPerRun(100, func() {
		d.Equal(c)
	})
	require.Zero(t, allocs)
}
