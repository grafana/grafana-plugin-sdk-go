package standalone

import (
	"errors"
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
		testDataDir, err := filepath.Abs("testdata")
		require.NoError(t, err)

		before := standaloneEnabled
		t.Cleanup(func() {
			standaloneEnabled = before
		})
		truthy := true
		standaloneEnabled = &truthy

		settings, enabled := ServerModeEnabled(pluginID)
		require.True(t, enabled)
		require.NotEmpty(t, settings.Address)
		require.Equal(t, filepath.Join(testDataDir, "plugin"), settings.Dir)
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
			pluginID: pluginID,
			dir:      "testdata",
			want:     filepath.Join("testdata", "plugin"),
			exists:   true,
		},
		{
			name:     "existing plugin found from plugin directory",
			pluginID: pluginID,
			dir:      filepath.Join("testdata", "plugin"),
			want:     filepath.Join("testdata", "plugin"),
			exists:   true,
		},
		{
			name:     "existing plugin found from nested plugin directory",
			pluginID: "grafana-nested-datasource",
			dir:      filepath.Join("testdata", "plugin"),
			want:     filepath.Join("testdata", "plugin", "datasource"),
			exists:   true,
		},
		{
			name:     "existing plugin found from dist directory",
			pluginID: "grafana-foobar-datasource",
			dir:      filepath.Join("testdata", "dist"),
			want:     filepath.Join("testdata", "dist"),
			exists:   true,
		},
		{
			name:     "non matching plugin id",
			pluginID: pluginID,
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
			require.Equal(t, tt.exists, found)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestServerSettings(t *testing.T) {
	originalWd := currentWd
	defer func() { currentWd = originalWd }()

	t.Run("Returns settings for valid plugin", func(t *testing.T) {
		testDataDir, err := filepath.Abs(filepath.Join("testdata", "plugin"))
		require.NoError(t, err)

		currentWd = func() (string, error) {
			return testDataDir, nil
		}

		settings, err := serverSettings(pluginID)
		require.NoError(t, err)
		require.NotEmpty(t, settings.Address)
		require.Equal(t, testDataDir, settings.Dir)
	})

	t.Run("Returns settings for nested plugin", func(t *testing.T) {
		testDataDir, err := filepath.Abs(filepath.Join("testdata", "plugin"))
		require.NoError(t, err)

		currentWd = func() (string, error) {
			return testDataDir, nil
		}

		settings, err := serverSettings("grafana-nested-datasource")
		require.NoError(t, err)
		require.NotEmpty(t, settings.Address)
		require.Equal(t, filepath.Join(testDataDir, "datasource"), settings.Dir)
	})

	t.Run("Returns error for invalid plugin ID", func(t *testing.T) {
		testDataDir, err := filepath.Abs("testdata")
		require.NoError(t, err)

		currentWd = func() (string, error) {
			return testDataDir, nil
		}

		settings, err := serverSettings("")
		require.Error(t, err)
		require.Contains(t, err.Error(), "plugin directory not found")
		require.Empty(t, settings)
	})

	t.Run("Handles pkg directory by moving up one level", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "plugin-test")
		require.NoError(t, err)
		defer func() {
			err = os.RemoveAll(tmpDir)
			t.Log("Error removing temp directory:", err)
		}()

		pkgDir := filepath.Join(tmpDir, "pkg")
		err = os.Mkdir(pkgDir, 0755)
		require.NoError(t, err)

		distDir := filepath.Join(tmpDir, "dist")
		err = os.Mkdir(distDir, 0755)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(distDir, "plugin.json"), []byte(`{"id": "`+pluginID+`"}`), 0600)
		require.NoError(t, err)

		currentWd = func() (string, error) {
			return pkgDir, nil
		}

		settings, err := serverSettings(pluginID)
		require.NoError(t, err)
		require.Equal(t, filepath.Join(tmpDir, "dist"), settings.Dir)
		require.NotEmpty(t, settings.Address)
	})

	t.Run("Uses dist directory when available", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "plugin-test")
		require.NoError(t, err)
		defer func() {
			err = os.RemoveAll(tmpDir)
			t.Log("Error removing temp directory:", err)
		}()

		distDir := filepath.Join(tmpDir, "dist")
		err = os.Mkdir(distDir, 0755)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(distDir, "plugin.json"), []byte(`{"id": "`+pluginID+`"}`), 0600)
		require.NoError(t, err)

		currentWd = func() (string, error) {
			return tmpDir, nil
		}

		settings, err := serverSettings(pluginID)
		require.NoError(t, err)
		require.Equal(t, filepath.Join(tmpDir, "dist"), settings.Dir)
		require.NotEmpty(t, settings.Address)
	})

	t.Run("Returns error when working directory cannot be determined", func(t *testing.T) {
		wdErr := errors.New("mock working directory error")
		currentWd = func() (string, error) {
			return "", wdErr
		}

		settings, err := serverSettings(pluginID)
		require.Error(t, err)
		require.ErrorIs(t, err, wdErr)
		require.Empty(t, settings)
	})
}
