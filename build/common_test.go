package build

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
		err := os.MkdirAll(pluginRootDir, os.ModePerm) // #nosec G301
		require.NoError(t, err)
		f, err := os.Create(filepath.Join(pluginRootDir, "plugin.json")) // #nosec G301 G304
		require.NoError(t, err)

		_, err = fmt.Fprintf(f, `{"executable": %q}`, executable)
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
			expectedArgs: []string{"build", "-o", filepath.Join(defaultOutputBinaryPath, "gpx_foo_darwin_arm64"), "-tags", "arrow_json_stdlib", "-ldflags", "-w -s -extldflags \"-static\" -X 'github.com/grafana/grafana-plugin-sdk-go/build/buildinfo.buildInfoJSON={.*}'", "./pkg"},
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
			expectedArgs: []string{"build", "-o", filepath.Join(defaultOutputBinaryPath, defaultNestedDataSourcePath, "gpx_foo_darwin_arm64"), "-tags", "arrow_json_stdlib", "-ldflags", "-w -s -extldflags \"-static\" -X 'github.com/grafana/grafana-plugin-sdk-go/build/buildinfo.buildInfoJSON={.*}'", "./pkg"},
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
			expectedArgs: []string{"build", "-o", filepath.Join(defaultOutputBinaryPath, "gpx_foobarbaz_windows_amd64.exe"), "-tags", "arrow_json_stdlib", "-ldflags", "-w -s -extldflags \"-static\" -X 'github.com/grafana/grafana-plugin-sdk-go/build/buildinfo.buildInfoJSON={.*}'", "./pkg"},
			wantErr:      assert.NoError,
		},
		{
			name: "Debug keeps marketplace licensing enabled",
			cfg: Config{
				OS:             "linux",
				Arch:           "amd64",
				EnableDebug:    true,
				Env:            map[string]string{"GOFLAGS": "-tags=marketplace_dev"},
				PluginJSONPath: filepath.Join(tmpDir, "debug-datasource"),
			},
			pluginJSONCreate: func(t *testing.T) {
				t.Helper()
				createPluginJSON(t, filepath.Join(tmpDir, "debug-datasource"), "gpx_debug")
			},
			expectedCfg: Config{
				OS:             "linux",
				Arch:           "amd64",
				EnableDebug:    true,
				Env:            map[string]string{"CGO_ENABLED": "0", "GOARCH": "amd64", "GOFLAGS": "-tags=marketplace_dev", "GOOS": "linux"},
				PluginJSONPath: filepath.Join(tmpDir, "debug-datasource"),
			},
			expectedArgs: []string{"build", "-o", filepath.Join(defaultOutputBinaryPath, "gpx_debug_linux_amd64"), "-tags", "arrow_json_stdlib", "-ldflags", "-extldflags \"-static\" -X 'github.com/grafana/grafana-plugin-sdk-go/build/buildinfo.buildInfoJSON={.*}'", "-gcflags=all=-N -l", "./pkg"},
			wantErr:      assert.NoError,
		},
		{
			name: "Marketplace development mode composes tags independently of debug mode",
			cfg: Config{
				OS:             "linux",
				Arch:           "arm64",
				MarketplaceDev: true,
				Env:            make(map[string]string),
				PluginJSONPath: filepath.Join(tmpDir, "marketplace-datasource"),
			},
			pluginJSONCreate: func(t *testing.T) {
				t.Helper()
				createPluginJSON(t, filepath.Join(tmpDir, "marketplace-datasource"), "gpx_marketplace")
			},
			expectedCfg: Config{
				OS:             "linux",
				Arch:           "arm64",
				MarketplaceDev: true,
				Env:            map[string]string{"CGO_ENABLED": "0", "GOARCH": "arm64", "GOOS": "linux"},
				PluginJSONPath: filepath.Join(tmpDir, "marketplace-datasource"),
			},
			expectedArgs: []string{"build", "-o", filepath.Join(defaultOutputBinaryPath, "gpx_marketplace_linux_arm64"), "-tags", "arrow_json_stdlib,marketplace_dev", "-ldflags", "-w -s -extldflags \"-static\" -X 'github.com/grafana/grafana-plugin-sdk-go/build/buildinfo.buildInfoJSON={.*}'", "./pkg"},
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

func Test_getBuildBackendCmdInfoAllowsMarketplaceDevFromCallback(t *testing.T) {
	originalBeforeBuild := beforeBuild
	t.Cleanup(func() {
		beforeBuild = originalBeforeBuild
	})

	pluginDir := filepath.Join(t.TempDir(), "callback-datasource")
	createPluginJSON(t, pluginDir, "gpx_callback")

	beforeBuild = func(cfg Config) (Config, error) {
		cfg.MarketplaceDev = true
		cfg.PluginJSONPath = pluginDir
		return cfg, nil
	}

	cfg, args, err := getBuildBackendCmdInfo(newBuildConfig("linux", "arm64"))
	require.NoError(t, err)
	require.True(t, cfg.MarketplaceDev)
	require.Contains(t, args, "arrow_json_stdlib,marketplace_dev")
}

func TestBuildTargetsSelectMarketplaceDevelopmentModeExplicitly(t *testing.T) {
	originalBeforeBuild := beforeBuild
	t.Cleanup(func() {
		beforeBuild = originalBeforeBuild
	})

	productionOS := runtime.GOOS
	productionArch := runtime.GOARCH
	if runtime.GOOS == "darwin" {
		productionArch = "arm64"
	}

	testErr := errors.New("stop before running go build")
	tests := []struct {
		name     string
		run      func() error
		expected Config
	}{
		{
			name: "production backend",
			run:  Build{}.Backend,
			expected: Config{
				OS:   productionOS,
				Arch: productionArch,
			},
		},
		{
			name: "debug",
			run:  Build{}.Debug,
			expected: Config{
				OS:          runtime.GOOS,
				Arch:        runtime.GOARCH,
				EnableDebug: true,
			},
		},
		{
			name: "custom platform",
			run: func() error {
				return Build{}.Custom("linux", "arm64")
			},
			expected: Config{
				OS:   "linux",
				Arch: "arm64",
			},
		},
		{
			name: "marketplace development current platform",
			run:  Build{}.MarketplaceDev,
			expected: Config{
				OS:             runtime.GOOS,
				Arch:           runtime.GOARCH,
				EnableDebug:    true,
				MarketplaceDev: true,
			},
		},
		{
			name: "marketplace development custom platform",
			run: func() error {
				return Build{}.MarketplaceDevFor("linux", "arm64")
			},
			expected: Config{
				OS:             "linux",
				Arch:           "arm64",
				EnableDebug:    true,
				MarketplaceDev: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var captured Config
			beforeBuild = func(cfg Config) (Config, error) {
				captured = cfg
				return cfg, testErr
			}

			err := tt.run()
			require.ErrorIs(t, err, testErr)
			require.Equal(t, tt.expected.OS, captured.OS)
			require.Equal(t, tt.expected.Arch, captured.Arch)
			require.Equal(t, tt.expected.EnableDebug, captured.EnableDebug)
			require.Equal(t, tt.expected.MarketplaceDev, captured.MarketplaceDev)
		})
	}
}

func createPluginJSON(t *testing.T, pluginDir string, executable string) {
	t.Helper()
	err := os.MkdirAll(pluginDir, os.ModePerm) // #nosec G301
	require.NoError(t, err)
	f, err := os.Create(filepath.Join(pluginDir, "plugin.json")) // #nosec G301 G304
	require.NoError(t, err)

	_, err = fmt.Fprintf(f, `{"executable": %q}`, executable)
	require.NoError(t, err)
	err = f.Close()
	require.NoError(t, err)
}
