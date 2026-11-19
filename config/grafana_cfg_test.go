package config

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/featuretoggles"
)

func splitSetIsEnabled(cfg map[string]string, f string) bool {
	features, exists := cfg[featuretoggles.EnabledFeatures]
	if !exists || features == "" {
		return false
	}
	enabled := map[string]struct{}{}
	for name := range strings.SplitSeq(features, ",") {
		enabled[name] = struct{}{}
	}
	_, ok := enabled[f]
	return ok
}

func TestFeatureTogglesIsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		cfg      map[string]string
		flag     string
		expected bool
	}{
		{name: "nil config", cfg: nil, flag: "a", expected: false},
		{name: "key missing", cfg: map[string]string{"other": "a"}, flag: "a", expected: false},
		{name: "empty list", cfg: map[string]string{featuretoggles.EnabledFeatures: ""}, flag: "", expected: false},
		{name: "single match", cfg: map[string]string{featuretoggles.EnabledFeatures: "a"}, flag: "a", expected: true},
		{name: "empty flag against single", cfg: map[string]string{featuretoggles.EnabledFeatures: "a"}, flag: "", expected: false},
		{name: "last of several", cfg: map[string]string{featuretoggles.EnabledFeatures: "a,b,c"}, flag: "c", expected: true},
		{name: "prefix is not a match", cfg: map[string]string{featuretoggles.EnabledFeatures: "ab"}, flag: "a", expected: false},
		{name: "suffix is not a match", cfg: map[string]string{featuretoggles.EnabledFeatures: "ba"}, flag: "a", expected: false},
		{name: "flag with comma", cfg: map[string]string{featuretoggles.EnabledFeatures: "a,b"}, flag: "a,b", expected: false},
		{name: "trailing comma enables empty flag", cfg: map[string]string{featuretoggles.EnabledFeatures: "a,"}, flag: "", expected: true},
		{name: "leading comma enables empty flag", cfg: map[string]string{featuretoggles.EnabledFeatures: ",a"}, flag: "", expected: true},
		{name: "double comma enables empty flag", cfg: map[string]string{featuretoggles.EnabledFeatures: "a,,b"}, flag: "", expected: true},
		{name: "spaces are kept", cfg: map[string]string{featuretoggles.EnabledFeatures: " a"}, flag: "a", expected: false},
		{name: "duplicates", cfg: map[string]string{featuretoggles.EnabledFeatures: "a,a"}, flag: "a", expected: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NewGrafanaCfg(tc.cfg).FeatureToggles().IsEnabled(tc.flag)
			require.Equal(t, tc.expected, got)
			require.Equal(t, splitSetIsEnabled(tc.cfg, tc.flag), got)
		})
	}

	t.Run("zero value", func(t *testing.T) {
		require.False(t, FeatureToggles{}.IsEnabled(""))
		require.False(t, FeatureToggles{}.IsEnabled("a"))
	})
}

func TestFeatureTogglesIsEnabledMatchesSplitSet(t *testing.T) {
	names := []string{"", "a", "ab", "b", "ba", " a", "a ", featuretoggles.TTLInstanceManager, featuretoggles.PluginsForceTLS13}
	flags := append([]string{"a,b", ",", "a,"}, names...)
	rng := rand.New(rand.NewPCG(1, 2)) //nolint:gosec // G404: deterministic test data
	for i := range 500 {
		list := make([]string, rng.IntN(6))
		for j := range list {
			list[j] = names[rng.IntN(len(names))]
		}
		cfg := map[string]string{featuretoggles.EnabledFeatures: strings.Join(list, ",")}
		toggles := NewGrafanaCfg(cfg).FeatureToggles()
		for _, f := range flags {
			require.Equal(t, splitSetIsEnabled(cfg, f), toggles.IsEnabled(f), "input %d: list %q, flag %q", i, cfg[featuretoggles.EnabledFeatures], f)
		}
	}
}

func TestFeatureTogglesAllocs(t *testing.T) {
	cfg := NewGrafanaCfg(map[string]string{featuretoggles.EnabledFeatures: toggleList(300)})
	allocs := testing.AllocsPerRun(100, func() {
		cfg.FeatureToggles().IsEnabled(featuretoggles.TTLInstanceManager)
	})
	require.Zero(t, allocs)
}

func toggleList(n int) string {
	names := make([]string, n)
	for i := range names {
		names[i] = fmt.Sprintf("featureToggleName%03d", i)
	}
	return strings.Join(names, ",")
}

var isEnabledSink bool

func BenchmarkFeatureToggles(b *testing.B) {
	for _, n := range []int{5, 300} {
		cfg := NewGrafanaCfg(map[string]string{featuretoggles.EnabledFeatures: toggleList(n)})
		b.Run(fmt.Sprintf("per-call/toggles=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				isEnabledSink = cfg.FeatureToggles().IsEnabled(featuretoggles.TTLInstanceManager)
			}
		})
		toggles := cfg.FeatureToggles()
		b.Run(fmt.Sprintf("held/toggles=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				isEnabledSink = toggles.IsEnabled(featuretoggles.TTLInstanceManager)
			}
		})
	}
}
