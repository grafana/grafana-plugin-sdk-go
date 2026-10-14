//go:build marketplace_dev

package marketplace

import (
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
