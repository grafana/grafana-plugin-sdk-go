package standalone

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const pluginID = "grafana-test-datasource"

func TestServerModeEnabled(t *testing.T) {
	t.Run("Disabled by default", func(t *testing.T) {
		settings, enabled := ServerModeEnabled(pluginID)
		require.False(t, enabled)
		require.Empty(t, settings)
	})

	t.Run("Enabled by env var", func(t *testing.T) {
		t.Setenv("GF_PLUGIN_GRPC_STANDALONE_GRAFANA_TEST_DATASOURCE", "true")

		settings, enabled := ServerModeEnabled(pluginID)
		require.True(t, enabled)
		require.False(t, settings.Debugger)
		require.NotEmpty(t, settings.Address)
		require.NotEmpty(t, settings.Dir)
	})

	t.Run("Enabled by flag", func(t *testing.T) {
		before := standaloneEnabled
		t.Cleanup(func() {
			standaloneEnabled = before
		})
		truthy := true
		standaloneEnabled = &truthy

		settings, enabled := ServerModeEnabled(pluginID)
		require.True(t, enabled)
		require.False(t, settings.Debugger)
		require.NotEmpty(t, settings.Address)
		require.NotEmpty(t, settings.Dir)
	})

	t.Run("Debug enabled by flag, but only when standalone is enabled and process has access to a plugin.json file",
		func(t *testing.T) {
			before := debugEnabled
			t.Cleanup(func() {
				debugEnabled = before
			})
			truthy := true
			debugEnabled = &truthy

			settings, enabled := ServerModeEnabled(pluginID)
			require.False(t, enabled)
			require.Empty(t, settings)

			curProcPath, err := os.Executable()
			require.NoError(t, err)

			dir := filepath.Dir(curProcPath)

			file, err := os.Create(filepath.Join(dir, "plugin.json"))
			require.NoError(t, err)
			t.Cleanup(func() {
				err = os.Remove(file.Name())
				require.NoError(t, err)
			})

			t.Setenv("GF_PLUGIN_GRPC_STANDALONE_GRAFANA_TEST_DATASOURCE", "true")
			settings, enabled = ServerModeEnabled(pluginID)
			require.True(t, enabled)
			require.True(t, settings.Debugger)
			require.NotEmpty(t, settings.Address)
			require.Equal(t, dir, settings.Dir)
		})
}

func Test_debuggerEnabled(t *testing.T) {
	t.Run("debug paths", func(t *testing.T) {
		for _, processPaths := range []string{
			// VsCode
			"testdata/plugin/__debug_bin",
			"testdata/plugin/__debug_bin.exe",
			// GoLand: Default run config name
			"testdata/GoLand/___XXgo_build_github_com_PACKAGENAME_pkg",
			"testdata/GoLand/___XXgo_build_github_com_PACKAGENAME_pkg.exe",
			// GoLand: Different run config name
			"testdata/GoLand/___1PLUGIN",
			"testdata/GoLand/___1PLUGIN.exe",
		} {
			t.Run(processPaths, func(t *testing.T) {
				require.True(t, debuggerEnabled(processPaths))
			})
		}
	})
}
