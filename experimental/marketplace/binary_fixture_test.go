package marketplace

import (
	"debug/buildinfo"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
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

func TestMarketplaceExecutableBuildTags(t *testing.T) {
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
		t.Run(platform.goos+"_"+platform.goarch, func(t *testing.T) {
			t.Parallel()
			for _, development := range []bool{false, true} {
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
					buildTags := readExecutableBuildTags(t, binaryPath)
					require.Equal(t, development, hasExactBuildTag(buildTags, "marketplace_dev"))
				})
			}
		})
	}
}

func TestMarketplaceExecutableBuildTagsInDebugBuilds(t *testing.T) {
	if testing.Short() {
		t.Skip("executable fixture builds")
	}
	t.Parallel()

	for _, development := range []bool{false, true} {
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
			buildTags := readExecutableBuildTags(t, binaryPath)
			require.Equal(t, development, hasExactBuildTag(buildTags, "marketplace_dev"))
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
		require.Empty(t, readExecutableBuildTags(t, binaryPath))
	})

	t.Run("development tag without marketplace code", func(t *testing.T) {
		t.Parallel()
		binaryPath := buildFixture(t, fixtureBuildOptions{
			goos:      runtime.GOOS,
			goarch:    runtime.GOARCH,
			tags:      []string{"marketplace_dev"},
			packageID: "datasourcefixture",
		})
		require.True(t, hasExactBuildTag(readExecutableBuildTags(t, binaryPath), "marketplace_dev"))
	})
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

	// The arguments come from test-controlled fixture options, not external input.
	cmd := exec.Command("go", args...) // #nosec G204
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

func readExecutableBuildTags(t *testing.T, path string) []string {
	t.Helper()

	info, err := buildinfo.ReadFile(path)
	require.NoError(t, err)
	for _, setting := range info.Settings {
		if setting.Key == "-tags" {
			return parseBuildTags(setting.Value)
		}
	}
	return nil
}

func parseBuildTags(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	})
}

func hasExactBuildTag(tags []string, expected string) bool {
	return slices.Contains(tags, expected)
}
