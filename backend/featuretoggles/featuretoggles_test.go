package featuretoggles

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnvFeatureToggles(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		t.Setenv(envFeatureTogglesEnable, "")
		flags := newFeatureTogglesFromEnv()
		require.False(t, flags.IsEnabled("abc"))
	})

	t.Run("single", func(t *testing.T) {
		t.Setenv(envFeatureTogglesEnable, "abc")
		flags := newFeatureTogglesFromEnv()
		require.True(t, flags.IsEnabled("abc"))
		require.False(t, flags.IsEnabled("def"))
	})

	t.Run("multiple", func(t *testing.T) {
		t.Setenv(envFeatureTogglesEnable, "abc def")
		flags := newFeatureTogglesFromEnv()
		require.True(t, flags.IsEnabled("abc"))
		require.True(t, flags.IsEnabled("def"))
		require.False(t, flags.IsEnabled("ghi"))
	})
}
