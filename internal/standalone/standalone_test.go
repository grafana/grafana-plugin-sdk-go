package standalone

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	pluginID = "grafana-test-datasource"
	addr     = "localhost:1234"
)

func TestServerModeEnabled(t *testing.T) {
	t.Run("Disabled by default", func(t *testing.T) {
		settings, enabled := ServerModeEnabled(pluginID)
		require.False(t, enabled)
		require.Empty(t, settings)
	})

	t.Run("Enabled by flag", func(t *testing.T) {
		before := standaloneEnabled
		t.Cleanup(func() {
			standaloneEnabled = before
		})
		truthy := true
		standaloneEnabled = &truthy

		curProcPath, err := os.Executable()
		require.NoError(t, err)

		settings, enabled := ServerModeEnabled(pluginID)
		require.True(t, enabled)
		require.NotEmpty(t, settings.Address)
		require.Equal(t, filepath.Dir(curProcPath), settings.Dir)
	})

	t.Run("Nearby dist folder will be used as server directory",
		func(t *testing.T) {
			curProcPath, err := os.Executable()
			require.NoError(t, err)

			procDir := filepath.Dir(curProcPath)
			distDir := filepath.Join(procDir, "dist")

			err = os.MkdirAll(distDir, 0755)
			require.NoError(t, err)
			_, err = os.Create(filepath.Join(distDir, "plugin.json"))
			require.NoError(t, err)
			t.Cleanup(func() {
				err = os.RemoveAll(distDir)
				require.NoError(t, err)
			})

			before := standaloneEnabled
			t.Cleanup(func() {
				standaloneEnabled = before
			})
			truthy := true
			standaloneEnabled = &truthy

			settings, enabled := ServerModeEnabled(pluginID)
			require.True(t, enabled)
			require.NotEmpty(t, settings.Address)
			require.Equal(t, distDir, settings.Dir)
		})
}

func TestClientModeEnabled(t *testing.T) {
	t.Run("Disabled by default", func(t *testing.T) {
		settings, enabled := ClientModeEnabled(pluginID)
		require.False(t, enabled)
		require.Empty(t, settings)
	})

	t.Run("Enabled by env var", func(t *testing.T) {
		t.Setenv("GF_PLUGIN_GRPC_ADDRESS_GRAFANA_TEST_DATASOURCE", addr)

		settings, enabled := ClientModeEnabled(pluginID)
		require.True(t, enabled)
		require.Equal(t, addr, settings.TargetAddress)
		require.Zero(t, settings.TargetPID)
	})

	t.Run("Enabled by standalone.txt file with valid address", func(t *testing.T) {
		curProcPath, err := os.Executable()
		require.NoError(t, err)

		dir := filepath.Dir(curProcPath)

		file, err := os.Create(filepath.Join(dir, "standalone.txt"))
		require.NoError(t, err)
		_, err = file.WriteString(addr)
		require.NoError(t, err)
		t.Cleanup(func() {
			err = os.Remove(file.Name())
			require.NoError(t, err)
		})

		settings, enabled := ClientModeEnabled(pluginID)
		require.True(t, enabled)
		require.Equal(t, addr, settings.TargetAddress)
		require.Zero(t, settings.TargetPID)
	})

	t.Run("Disabled if standalone.txt does not contain a valid address", func(t *testing.T) {
		curProcPath, err := os.Executable()
		require.NoError(t, err)

		dir := filepath.Dir(curProcPath)

		file, err := os.Create(filepath.Join(dir, "standalone.txt"))
		require.NoError(t, err)
		t.Cleanup(func() {
			err = os.Remove(file.Name())
			require.NoError(t, err)
		})

		settings, enabled := ClientModeEnabled(pluginID)
		require.False(t, enabled)
		require.Empty(t, settings.TargetAddress)
		require.Zero(t, settings.TargetPID)
	})

	t.Run("Enabled if pid.txt exists, but is empty", func(t *testing.T) {
		t.Setenv("GF_PLUGIN_GRPC_ADDRESS_GRAFANA_TEST_DATASOURCE", addr)

		curProcPath, err := os.Executable()
		require.NoError(t, err)

		dir := filepath.Dir(curProcPath)
		file, err := os.Create(filepath.Join(dir, "pid.txt"))
		require.NoError(t, err)
		t.Cleanup(func() {
			err = os.Remove(file.Name())
			require.NoError(t, err)
		})

		settings, enabled := ClientModeEnabled(pluginID)
		require.True(t, enabled)
		require.Equal(t, addr, settings.TargetAddress)
		require.Zero(t, settings.TargetPID)
	})

	t.Run("Disabled if pid.txt exists, but has invalid pid", func(t *testing.T) {
		t.Setenv("GF_PLUGIN_GRPC_ADDRESS_GRAFANA_TEST_DATASOURCE", addr)

		curProcPath, err := os.Executable()
		require.NoError(t, err)

		dir := filepath.Dir(curProcPath)
		file, err := os.Create(filepath.Join(dir, "pid.txt"))
		require.NoError(t, err)
		_, err = file.WriteString("100000000000000")
		require.NoError(t, err)
		t.Cleanup(func() {
			err = os.Remove(file.Name())
			require.NoError(t, err)
		})

		settings, enabled := ClientModeEnabled(pluginID)
		require.False(t, enabled)
		require.Empty(t, settings.TargetAddress)
		require.Zero(t, settings.TargetPID)
	})
}

func Test_findPluginDir(t *testing.T) {
	tests := []struct {
		name     string
		dir      string
		pluginID string
		want     string
		exists   bool
	}{
		{
			name:     "existing plugin found from root directory",
			pluginID: "grafana-foobar-datasource",
			dir:      "testdata",
			want:     filepath.Join("testdata", "plugin"),
			exists:   true,
		},
		{
			name:     "existing plugin found from plugin directory",
			pluginID: "grafana-foobar-datasource",
			dir:      filepath.Join("testdata", "plugin"),
			want:     filepath.Join("testdata", "plugin"),
			exists:   true,
		},
		{
			name:     "non matching plugin id",
			pluginID: "grafana-foobar-datasource",
			dir:      filepath.Join("testdata", "GoLand"),
			want:     "",
		},
		{
			name:     "non-existing plugin",
			pluginID: "non-existing-plugin",
			dir:      "testdata",
			want:     "",
		},
		{
			name:     "empty plugin ID",
			pluginID: "",
			dir:      "testdata",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := findPluginDir(tt.dir, tt.pluginID)
			if found != tt.exists {
				t.Errorf("findPluginDir() found = %v, want %v", found, tt.exists)
			}
			if got != tt.want {
				t.Errorf("findPluginDir() got = %v, want %v", got, tt.want)
			}
		})
	}
}
