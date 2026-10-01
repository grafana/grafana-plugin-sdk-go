package marketplace

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/build/buildinfo"
)

func TestMarketplaceManageLicenseMode(t *testing.T) {
	if os.Getenv("GO_WANT_MARKETPLACE_MANAGE_HELPER") == "1" {
		buildinfo.GetBuildInfo = buildinfo.GetterFunc(func() (buildinfo.Info, error) {
			return buildinfo.Info{PluginID: "test-plugin", Version: "test"}, nil
		})
		err := Manage("test-plugin", func(context.Context, backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
			panic("build info mode must not create an instance")
		}, datasource.ManageOpts{})
		if err != nil {
			t.Fatal(err)
		}
		return
	}

	executable, err := os.Executable()
	require.NoError(t, err)
	for _, queryTime := range []bool{false, true} {
		name := "startup validation by default"
		if queryTime {
			name = "query-time validation skips startup license"
		}
		t.Run(name, func(t *testing.T) {
			// Build info mode exits before serving. Only -qtlv can reach it without
			// a startup license, so this checks Manage without starting a server.
			args := []string{"-test.run=^TestMarketplaceManageLicenseMode$", "-buildinfo"}
			if queryTime {
				args = append(args, "-qtlv")
			}
			cmd := exec.Command(executable, args...)
			cmd.Env = []string{
				"GO_WANT_MARKETPLACE_MANAGE_HELPER=1",
				"HOME=" + t.TempDir(),
				"USERPROFILE=" + t.TempDir(),
				marketplaceLicensePathEnv + "=" + filepath.Join(t.TempDir(), "missing.jwt"),
			}
			output, err := cmd.CombinedOutput()
			if queryTime {
				require.NoError(t, err, "%s", output)
				require.True(t, json.Valid(output), "%s", output)
			} else {
				require.Error(t, err)
				require.Contains(t, string(output), "Marketplace License Error")
			}
		})
	}
}
