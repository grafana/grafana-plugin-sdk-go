package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetExecutableFromPluginJSON(t *testing.T) {
	type args struct {
		pluginDir string
	}
	tcs := []struct {
		name          string
		args          args
		executable    string
		pluginJSONDir string
		expected      string
		err           bool
	}{
		{
			name: "Can retrieve executable field from a plugin.json found in provided directory",
			args: args{
				pluginDir: "foobar-datasource",
			},
			pluginJSONDir: "foobar-datasource",
			executable:    "gpx_foo",
			expected:      "gpx_foo",
		},
		{
			name: "Can retrieve executable field of nested 'datasource' plugin.json as long as the executable path is in the root directory",
			args: args{
				pluginDir: "foobar-app",
			},
			pluginJSONDir: filepath.Join("foobar-app", "datasource"),
			executable:    "../gpx_foo",
			expected:      "gpx_foo",
		},
		{
			name: "Cannot retrieve executable field of nested 'datasource' plugin.json when executable path is not in the root directory",
			args: args{
				pluginDir: "foobar-app",
			},
			pluginJSONDir: filepath.Join("foobar-app", "datasource"),
			executable:    "gpx_foo",
			err:           true,
		},
		{
			name: "Cannot retrieve executable when no plugin.json found in root or nested 'datasource' directory",
			args: args{
				pluginDir: "foobar-app",
			},
			pluginJSONDir: filepath.Join("foobar-app", "foobar-datasource"),
			executable:    "gpx_foo",
			err:           true,
		},
	}

	for _, tc := range tcs {
		rootDir := t.TempDir()
		pluginRootDir := filepath.Join(rootDir, tc.pluginJSONDir)
		err := os.MkdirAll(pluginRootDir, os.ModePerm)
		require.NoError(t, err)
		f, err := os.Create(filepath.Join(pluginRootDir, "plugin.json"))
		require.NoError(t, err)

		_, err = f.WriteString(fmt.Sprintf(`{"executable": %q}`, tc.executable))
		require.NoError(t, err)
		err = f.Close()
		require.NoError(t, err)

		t.Run(tc.name, func(t *testing.T) {
			got, err := GetExecutableFromPluginJSON(filepath.Join(rootDir, tc.args.pluginDir))
			if tc.err {
				require.Error(t, err)
				return
			}
			require.Equal(t, tc.expected, got)
		})
	}
}
