//go:build !marketplace_dev

package marketplace

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace/licensing"
)

func TestMarketplaceLicenseProductionModeEnforcesLicensing(t *testing.T) {
	t.Setenv("GF_DEFAULT_APP_MODE", "development")
	t.Setenv("GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS", "example-datasource")
	require.False(t, developmentLicenseBypass)

	for _, licensePluginID := range []string{"example-datasource", "parent-app"} {
		t.Run(licensePluginID, func(t *testing.T) {
			licenseErr := errors.New("missing license")
			loadCalled := false
			serverCalled := false

			err := checkMarketplacePluginLicenseWithDeps(
				"example-datasource", licensePluginID,
				func(pluginID string) *licensing.LicenseToken {
					loadCalled = true
					require.Equal(t, licensePluginID, pluginID)
					return &licensing.LicenseToken{Status: licensing.Invalid, Error: licenseErr}
				},
				func(pluginID, loadedLicensePluginID string, err error) error {
					serverCalled = true
					require.Equal(t, "example-datasource", pluginID)
					require.Equal(t, licensePluginID, loadedLicensePluginID)
					require.ErrorIs(t, err, licenseErr)
					return nil
				},
			)

			require.ErrorContains(t, err, "you do not have a valid license for the marketplace plugin "+licensePluginID)
			require.True(t, loadCalled)
			require.True(t, serverCalled)
		})
	}
}
