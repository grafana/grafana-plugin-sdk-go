//go:build marketplace_dev

package marketplace

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func TestMarketplaceLicenseDevelopmentModeIgnoresRuntimeInputs(t *testing.T) {
	require.True(t, developmentLicenseBypass)

	for _, mode := range []string{"", marketplaceLicenseValidationModeRequest} {
		t.Run("mode="+mode, func(t *testing.T) {
			t.Setenv(marketplaceLicenseValidationModeEnv, mode)
			t.Setenv(marketplaceLicenseTextEnv, "not-a-jwt")
			t.Setenv(marketplaceLicensePathEnv, t.TempDir())
			t.Setenv(marketplaceLicenseValidationKeyEnv, "not-a-jwks")
			t.Setenv(marketplaceAppURLEnv, "://invalid")

			serveOpts := backend.ServeOpts{QueryDataHandler: &backend.BaseHandler{}}
			got, err := Licensing(Opts{ParentPluginID: "../invalid-parent-id"})("../invalid-plugin-id", serveOpts)
			require.NoError(t, err)
			require.Equal(t, serveOpts, got)
		})
	}
}
