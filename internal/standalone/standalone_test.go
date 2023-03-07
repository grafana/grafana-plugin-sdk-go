package standalone

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetInfo(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		info, err := getInfo("plugin", "testdata/plugin/gpx_plugin", false, false)
		require.NoError(t, err)
		require.False(t, info.Standalone)
		require.False(t, info.Debugger)
	})

	t.Run("standalone", func(t *testing.T) {
		t.Run("without debug flag", func(t *testing.T) {
			info, err := getInfo("plugin", "testdata/plugin/gpx_plugin", true, false)
			require.NoError(t, err)
			require.True(t, info.Standalone)
			require.False(t, info.Debugger)
			// No random free port, must be read from standalone.txt
			require.Empty(t, info.Address)
		})

		t.Run("with debug flag", func(t *testing.T) {
			info, err := getInfo("plugin", "testdata/plugin/gpx_plugin", true, true)
			require.NoError(t, err)
			require.True(t, info.Standalone)
			require.True(t, info.Debugger)
			// Should have a random free port
			require.NotEmpty(t, info.Address)
		})

		t.Run("with debug executable", func(t *testing.T) {
			for _, fn := range []string{
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
				t.Run(fn, func(t *testing.T) {
					info, err := getInfo("plugin", fn, true, false)
					require.NoError(t, err)
					require.True(t, info.Standalone)
					require.True(t, info.Debugger)
					// Should have a random free port
					require.NotEmpty(t, info.Address)
				})
			}
		})

		t.Run("no debug with standalone.txt file", func(t *testing.T) {
			info, err := getInfo("plugin", "testdata/standalone-txt/gpx_plugin", true, false)
			require.NoError(t, err)
			require.True(t, info.Standalone)
			require.False(t, info.Debugger)
			// Read from standalone.txt
			require.Equal(t, ":1234", info.Address)
		})
	})
}
