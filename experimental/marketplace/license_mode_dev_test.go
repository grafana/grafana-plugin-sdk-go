//go:build marketplace_dev

package marketplace

import (
	"github.com/hashicorp/go-plugin"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace/licensing"
)

func TestMarketplaceLicenseDevelopmentModeSkipsLicensing(t *testing.T) {
	t.Parallel()
	require.True(t, developmentLicenseBypass)

	serveOpts := backend.ServeOpts{QueryDataHandler: &backend.BaseHandler{}}
	got := checkMarketplacePluginLicenseWithDeps(
		"example-datasource", "parent-app", serveOpts,
		func(string) *licensing.LicenseToken {
			require.FailNow(t, "development mode loaded a license")
			return nil
		},
	)
	require.Equal(t, serveOpts, got)
}

func TestMarketplaceLicenseDevelopmentModeIgnoresRuntimeInputs(t *testing.T) {
	t.Setenv(marketplaceLicenseTextEnv, "not-a-jwt")
	t.Setenv(marketplaceLicensePathEnv, t.TempDir())
	t.Setenv(marketplaceLicenseValidationKeyEnv, "not-a-jwks")
	t.Setenv(marketplaceAppURLEnv, "://invalid")

	serveOpts := backend.ServeOpts{QueryDataHandler: &backend.BaseHandler{}}
	got, err := Licensing(Opts{ParentPluginID: "../invalid-parent-id"})("../invalid-plugin-id", serveOpts)
	require.NoError(t, err)
	require.Equal(t, serveOpts, got)
}

// Not parallel: it flips the package-level -qtlv flag and the mode environment variable.
func TestMarketplaceLicenseDevelopmentModeSkipsQueryTimeValidation(t *testing.T) {
	for _, tt := range []struct {
		name string
		flag bool
		env  string
	}{
		{"flag", true, ""},
		{"env", false, "request"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			old := *queryTimeLicenseValidation
			*queryTimeLicenseValidation = tt.flag
			t.Cleanup(func() { *queryTimeLicenseValidation = old })
			t.Setenv(marketplaceLicenseValidationModeEnv, tt.env)

			// ExtraPlugins would be rejected, and a middleware installed, if the bypass did not apply.
			serveOpts := backend.ServeOpts{
				QueryDataHandler: &backend.BaseHandler{},
				ExtraPlugins:     plugin.PluginSet{"extra": nil},
			}
			got, err := Licensing(Opts{})("example-datasource", serveOpts)
			require.NoError(t, err)
			require.Equal(t, serveOpts, got)
			require.Empty(t, got.HandlerMiddlewares)
		})
	}
}
