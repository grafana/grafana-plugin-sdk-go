//go:build marketplace_dev

package marketplace

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace/licensing"
)

func TestMarketplaceLicenseDevelopmentModeSkipsLicensing(t *testing.T) {
	t.Parallel()
	require.True(t, developmentLicenseBypass)

	err := checkMarketplacePluginLicense(
		"example-datasource",
		func(string) *licensing.LicenseToken {
			require.FailNow(t, "development mode loaded a license")
			return nil
		},
		func(string, error) error {
			require.FailNow(t, "development mode started the invalid-license server")
			return errors.New("unreachable")
		},
	)
	require.NoError(t, err)
}

func TestCheckMarketplacePluginLicenseDevelopmentModeIgnoresRuntimeInputs(t *testing.T) {
	t.Setenv(marketplaceLicenseTextEnv, "not-a-jwt")
	t.Setenv(marketplaceLicensePathEnv, t.TempDir())
	t.Setenv(marketplaceLicenseValidationKeyEnv, "not-a-jwks")
	t.Setenv(marketplaceAppURLEnv, "://invalid")

	require.NoError(t, CheckMarketplacePluginLicense("../invalid-plugin-id"))
}
