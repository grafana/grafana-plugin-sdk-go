package marketplace

import (
	"bytes"
	"debug/buildinfo"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/require"
)

type fixtureBuildOptions struct {
	goos      string
	goarch    string
	debug     bool
	tags      []string
	goFlags   string
	packageID string
}

type executableSignals struct {
	tags         []string
	hasMarker    bool
	buildInfoErr error
}

func TestMarketplaceExecutableBuildSignals(t *testing.T) {
	if testing.Short() {
		t.Skip("cross-platform executable fixture matrix")
	}
	t.Parallel()

	platforms := []struct {
		goos   string
		goarch string
	}{
		{goos: "linux", goarch: "amd64"},
		{goos: "linux", goarch: "arm"},
		{goos: "linux", goarch: "arm64"},
		{goos: "darwin", goarch: "amd64"},
		{goos: "darwin", goarch: "arm64"},
		{goos: "windows", goarch: "amd64"},
	}

	for _, platform := range platforms {
		platform := platform
		t.Run(platform.goos+"_"+platform.goarch, func(t *testing.T) {
			t.Parallel()
			for _, development := range []bool{false, true} {
				development := development
				name := "production"
				tags := []string{"arrow_json_stdlib"}
				goFlags := "-tags=marketplace_dev"
				if development {
					name = "development"
					tags = append(tags, "marketplace_dev")
					goFlags = ""
				}

				t.Run(name, func(t *testing.T) {
					binaryPath := buildFixture(t, fixtureBuildOptions{
						goos:      platform.goos,
						goarch:    platform.goarch,
						tags:      tags,
						goFlags:   goFlags,
						packageID: "marketplacefixture",
					})
					signals := inspectExecutable(t, binaryPath)

					require.NoError(t, signals.buildInfoErr)
					require.Equal(t, development, hasExactBuildTag(signals.tags, "marketplace_dev"))
					require.Equal(t, development, signals.hasMarker)
				})
			}
		})
	}
}

func TestMarketplaceExecutableBuildSignalsInDebugBuilds(t *testing.T) {
	if testing.Short() {
		t.Skip("executable fixture builds")
	}
	t.Parallel()

	for _, development := range []bool{false, true} {
		development := development
		name := "production"
		tags := []string{"arrow_json_stdlib"}
		if development {
			name = "development"
			tags = append(tags, "marketplace_dev")
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()
			binaryPath := buildFixture(t, fixtureBuildOptions{
				goos:      runtime.GOOS,
				goarch:    runtime.GOARCH,
				debug:     true,
				tags:      tags,
				packageID: "marketplacefixture",
			})
			signals := inspectExecutable(t, binaryPath)

			require.NoError(t, signals.buildInfoErr)
			require.Equal(t, development, hasExactBuildTag(signals.tags, "marketplace_dev"))
			require.Equal(t, development, signals.hasMarker)
		})
	}
}

func TestOrdinaryDatasourceExecutableIsNonMarketplaceControl(t *testing.T) {
	if testing.Short() {
		t.Skip("executable fixture builds")
	}
	t.Parallel()

	t.Run("readable metadata without tags", func(t *testing.T) {
		t.Parallel()
		binaryPath := buildFixture(t, fixtureBuildOptions{
			goos:      runtime.GOOS,
			goarch:    runtime.GOARCH,
			packageID: "datasourcefixture",
		})
		signals := inspectExecutable(t, binaryPath)

		require.NoError(t, signals.buildInfoErr)
		require.Empty(t, signals.tags)
		require.False(t, signals.hasMarker)
	})

	t.Run("development tag without marketplace code", func(t *testing.T) {
		t.Parallel()
		binaryPath := buildFixture(t, fixtureBuildOptions{
			goos:      runtime.GOOS,
			goarch:    runtime.GOARCH,
			tags:      []string{"marketplace_dev"},
			packageID: "datasourcefixture",
		})
		signals := inspectExecutable(t, binaryPath)

		require.NoError(t, signals.buildInfoErr)
		require.True(t, hasExactBuildTag(signals.tags, "marketplace_dev"))
		require.False(t, signals.hasMarker)
	})
}

func TestExecutableSignalInspectionKeepsMarkerWhenBuildInfoIsUnreadable(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "not-a-go-executable")
	require.NoError(t, os.WriteFile(path, []byte("prefix"+developmentBuildMarker+"suffix"), 0600))

	signals := inspectExecutable(t, path)
	require.Error(t, signals.buildInfoErr)
	require.True(t, signals.hasMarker)
	require.Empty(t, signals.tags)
}

func TestHasExactBuildTag(t *testing.T) {
	t.Parallel()
	tags := parseBuildTags("arrow_json_stdlib,marketplace_dev_tools marketplace_dev")
	require.True(t, hasExactBuildTag(tags, "marketplace_dev"))
	require.False(t, hasExactBuildTag(tags, "marketplace"))
	require.False(t, hasExactBuildTag(tags, "marketplace_dev_tool"))
}

func buildFixture(t *testing.T, opts fixtureBuildOptions) string {
	t.Helper()

	name := opts.packageID + "_" + opts.goos + "_" + opts.goarch
	if opts.goos == "windows" {
		name += ".exe"
	}
	outputPath := filepath.Join(t.TempDir(), name)

	args := []string{"build", "-trimpath", "-o", outputPath}
	if opts.debug {
		args = append(args, "-gcflags=all=-N -l")
	} else {
		args = append(args, "-ldflags=-s -w")
	}
	if len(opts.tags) > 0 {
		args = append(args, "-tags="+strings.Join(opts.tags, ","))
	}
	args = append(args, "./testdata/"+opts.packageID)

	cmd := exec.Command("go", args...)
	cmd.Env = fixtureBuildEnvironment(opts.goos, opts.goarch, opts.goFlags)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "go %s failed:\n%s", strings.Join(args, " "), output)

	return outputPath
}

func fixtureBuildEnvironment(goos, goarch, goFlags string) []string {
	overrides := map[string]string{
		"CGO_ENABLED": "0",
		"GOARCH":      goarch,
		"GOFLAGS":     goFlags,
		"GOOS":        goos,
	}
	env := make([]string, 0, len(os.Environ())+len(overrides))
	for _, entry := range os.Environ() {
		key, _, ok := strings.Cut(entry, "=")
		if ok {
			if _, overridden := overrides[key]; overridden {
				continue
			}
		}
		env = append(env, entry)
	}
	for key, value := range overrides {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	return env
}

func inspectExecutable(t *testing.T, path string) executableSignals {
	t.Helper()

	contents, err := os.ReadFile(path) // #nosec G304 -- the test creates this path.
	require.NoError(t, err)

	signals := executableSignals{
		hasMarker: bytes.Contains(contents, []byte(developmentBuildMarker)),
	}
	info, err := buildinfo.ReadFile(path)
	if err != nil {
		signals.buildInfoErr = err
		return signals
	}
	for _, setting := range info.Settings {
		if setting.Key == "-tags" {
			signals.tags = parseBuildTags(setting.Value)
			break
		}
	}
	return signals
}

func parseBuildTags(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	})
}

func hasExactBuildTag(tags []string, expected string) bool {
	for _, tag := range tags {
		if tag == expected {
			return true
		}
	}
	return false
}
