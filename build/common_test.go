package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_getExecutableNameForPlugin(t *testing.T) {
	rootDir := t.TempDir()
	plugins := map[string]string{
		"foo-datasource": "gpx_foo",
		"bar-datasource": "gpx_bar",
		"baz-datasource": "gpx_baz",
	}

	type args struct {
		os        string
		arch      string
		pluginDir string
	}
	tcs := []struct {
		name          string
		args          args
		expected      string
		expectedCache map[string]string
		wantErr       assert.ErrorAssertionFunc
	}{
		{
			name: "Valid plugin with executable is found and cached",
			args: args{
				os:        "darwin",
				arch:      "arm64",
				pluginDir: filepath.Join(rootDir, "foo-datasource"),
			},
			expected: "gpx_foo_darwin_arm64",
			expectedCache: map[string]string{
				filepath.Join(rootDir, "foo-datasource"): "gpx_foo",
			},
			wantErr: assert.NoError,
		},
		{
			name: "Another valid plugin with executable is found and cached",
			args: args{
				os:        "windows",
				arch:      "amd64",
				pluginDir: filepath.Join(rootDir, "baz-datasource"),
			},
			expected: "gpx_baz_windows_amd64.exe",
			expectedCache: map[string]string{
				filepath.Join(rootDir, "foo-datasource"): "gpx_foo",
				filepath.Join(rootDir, "baz-datasource"): "gpx_baz",
			},
			wantErr: assert.NoError,
		},
		{
			name: "Same plugin with executable is found in cache",
			args: args{
				os:        "windows",
				arch:      "amd64",
				pluginDir: filepath.Join(rootDir, "baz-datasource"),
			},
			expected: "gpx_baz_windows_amd64.exe",
			expectedCache: map[string]string{
				filepath.Join(rootDir, "foo-datasource"): "gpx_foo",
				filepath.Join(rootDir, "baz-datasource"): "gpx_baz",
			},
			wantErr: assert.NoError,
		},
		{
			name: "Non existing plugin returns an error",
			args: args{
				os:        "linux",
				arch:      "amd64",
				pluginDir: filepath.Join(rootDir, "foobarbaz-datasource"),
			},
			expected: "",
			expectedCache: map[string]string{
				filepath.Join(rootDir, "foo-datasource"): "gpx_foo",
				filepath.Join(rootDir, "baz-datasource"): "gpx_baz",
			},
			wantErr: assert.Error,
		},
	}

	for pluginID, executable := range plugins {
		pluginRootDir := filepath.Join(rootDir, pluginID)
		err := os.MkdirAll(pluginRootDir, os.ModePerm)
		require.NoError(t, err)
		f, err := os.Create(filepath.Join(pluginRootDir, "plugin.json"))
		require.NoError(t, err)

		_, err = f.WriteString(fmt.Sprintf(`{"executable": %q}`, executable))
		require.NoError(t, err)
		err = f.Close()
		require.NoError(t, err)
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got, err := getExecutableNameForPlugin(tc.args.os, tc.args.arch, tc.args.pluginDir)
			if !tc.wantErr(t, err, fmt.Sprintf("getExecutableNameForPlugin(%v, %v, %v)", tc.args.os, tc.args.arch, tc.args.pluginDir)) {
				return
			}
			assert.Equalf(t, tc.expected, got, "getExecutableNameForPlugin(%v, %v, %v)", tc.args.os, tc.args.arch, tc.args.pluginDir)

			numCached := 0
			executableNameCache.Range(func(_, _ any) bool {
				numCached++
				return true
			})

			assert.Equal(t, len(tc.expectedCache), numCached)
			for s, s2 := range tc.expectedCache {
				val, ok := executableNameCache.Load(s)
				assert.True(t, ok)
				assert.Equal(t, s2, val.(string))
			}
		})
	}
}

func Test_getBuildBackendCmdInfo(t *testing.T) {
	tmpDir := t.TempDir()
	tcs := []struct {
		name             string
		pluginJSONCreate func(t *testing.T)
		cfg              Config
		expectedCfg      Config
		expectedArgs     []string
		wantErr          assert.ErrorAssertionFunc
	}{
		{
			name: "Happy path",
			cfg: Config{
				OS:             "darwin",
				Arch:           "arm64",
				Env:            make(map[string]string),
				PluginJSONPath: filepath.Join(tmpDir, "foobar-datasource"),
			},
			pluginJSONCreate: func(t *testing.T) {
				t.Helper()
				createPluginJSON(t, filepath.Join(tmpDir, "foobar-datasource"), "gpx_foo")
			},
			expectedCfg: Config{
				OS:             "darwin",
				Arch:           "arm64",
				Env:            map[string]string{"CGO_ENABLED": "0", "GOARCH": "arm64", "GOOS": "darwin"},
				PluginJSONPath: filepath.Join(tmpDir, "foobar-datasource"),
			},
			expectedArgs: []string{"build", "-o", filepath.Join(defaultOutputBinaryPath, "gpx_foo_darwin_arm64"), "-tags", "arrow_json_stdlib", "-ldflags", "-w -s -extldflags \"-static\" -X 'github.com/grafana/grafana-plugin-sdk-go/build.buildInfoJSON={.*}'", "./pkg"},
			wantErr:      assert.NoError,
		},
		{
			name: "Happy path with nested datasource",
			cfg: Config{
				OS:             "darwin",
				Arch:           "arm64",
				Env:            make(map[string]string),
				PluginJSONPath: filepath.Join(tmpDir, "foobar-app"),
			},
			pluginJSONCreate: func(t *testing.T) {
				t.Helper()
				createPluginJSON(t, filepath.Join(tmpDir, "foobar-app", defaultNestedDataSourcePath), "gpx_foo")
			},
			expectedCfg: Config{
				OS:             "darwin",
				Arch:           "arm64",
				Env:            map[string]string{"CGO_ENABLED": "0", "GOARCH": "arm64", "GOOS": "darwin"},
				PluginJSONPath: filepath.Join(tmpDir, "foobar-app"),
			},
			expectedArgs: []string{"build", "-o", filepath.Join(defaultOutputBinaryPath, defaultNestedDataSourcePath, "gpx_foo_darwin_arm64"), "-tags", "arrow_json_stdlib", "-ldflags", "-w -s -extldflags \"-static\" -X 'github.com/grafana/grafana-plugin-sdk-go/build.buildInfoJSON={.*}'", "./pkg"},
			wantErr:      assert.NoError,
		},
		{
			name: "Happy path with nested datasource that has executable path in root directory",
			cfg: Config{
				OS:             "windows",
				Arch:           "amd64",
				Env:            make(map[string]string),
				PluginJSONPath: filepath.Join(tmpDir, "foobarbaz-app"),
			},
			pluginJSONCreate: func(t *testing.T) {
				t.Helper()
				createPluginJSON(t, filepath.Join(tmpDir, "foobarbaz-app", defaultNestedDataSourcePath), "../gpx_foobarbaz")
			},
			expectedCfg: Config{
				OS:             "windows",
				Arch:           "amd64",
				Env:            map[string]string{"CGO_ENABLED": "0", "GOARCH": "amd64", "GOOS": "windows"},
				PluginJSONPath: filepath.Join(tmpDir, "foobarbaz-app"),
			},
			expectedArgs: []string{"build", "-o", filepath.Join(defaultOutputBinaryPath, "gpx_foobarbaz_windows_amd64.exe"), "-tags", "arrow_json_stdlib", "-ldflags", "-w -s -extldflags \"-static\" -X 'github.com/grafana/grafana-plugin-sdk-go/build.buildInfoJSON={.*}'", "./pkg"},
			wantErr:      assert.NoError,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tc.pluginJSONCreate(t)

			cfg, args, err := getBuildBackendCmdInfo(tc.cfg)
			if !tc.wantErr(t, err, fmt.Sprintf("getBuildBackendCmdInfo(%v)", tc.cfg)) {
				return
			}
			assert.Equalf(t, tc.expectedCfg, cfg, "getBuildBackendCmdInfo(%v)", tc.cfg)

			// check if expected build arg regex matches against actual build arg
			buildArg := strings.Join(args, " ")
			expectedBuildArg := strings.Join(tc.expectedArgs, " ")
			assert.Regexp(t, expectedBuildArg, buildArg, "getBuildBackendCmdInfo(%v)", tc.cfg)
		})
	}
}

func createPluginJSON(t *testing.T, pluginDir string, executable string) {
	t.Helper()
	err := os.MkdirAll(pluginDir, os.ModePerm)
	require.NoError(t, err)
	f, err := os.Create(filepath.Join(pluginDir, "plugin.json"))
	require.NoError(t, err)

	_, err = f.WriteString(fmt.Sprintf(`{"executable": %q}`, executable))
	require.NoError(t, err)
	err = f.Close()
	require.NoError(t, err)
}
