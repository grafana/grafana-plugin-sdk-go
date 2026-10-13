//go:build !marketplace_dev

package marketplace

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/go-plugin"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace/licensing"
)

// licensedServeOpts returns serve options as datasource.Manage or app.Manage would build them.
func licensedServeOpts() backend.ServeOpts {
	handler := &backend.BaseHandler{}
	return backend.ServeOpts{
		CheckHealthHandler:      handler,
		CallResourceHandler:     handler,
		QueryDataHandler:        handler,
		QueryChunkedDataHandler: handler,
		StreamHandler:           handler,
		AdmissionHandler:        handler,
		ConversionHandler:       handler,
		GRPCSettings:            backend.GRPCSettings{MaxReceiveMsgSize: 1, MaxSendMsgSize: 2},
		ExtraPlugins:            plugin.PluginSet{"extra": nil},
	}
}

func requireInvalidLicenseServeOpts(t *testing.T, got backend.ServeOpts, pluginID, licensePluginID string, verboseError error) {
	t.Helper()
	handler, ok := got.CheckHealthHandler.(*invalidLicenseHandler)
	require.True(t, ok, "CheckHealthHandler is %T", got.CheckHealthHandler)
	require.Equal(t, pluginID, handler.pluginID)
	require.ErrorIs(t, handler.verboseError, verboseError)
	require.ErrorContains(t, handler.err, "The Marketplace plugin "+licensePluginID+" is not available")

	require.Equal(t, backend.ServeOpts{
		CheckHealthHandler:      handler,
		CallResourceHandler:     handler,
		QueryDataHandler:        handler,
		QueryChunkedDataHandler: handler,
		StreamHandler:           handler,
		GRPCSettings:            backend.GRPCSettings{MaxReceiveMsgSize: 1, MaxSendMsgSize: 2},
	}, got)
}

func TestMarketplaceLicenseProductionModeEnforcesLicensing(t *testing.T) {
	t.Setenv("GF_DEFAULT_APP_MODE", "development")
	t.Setenv("GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS", "example-datasource")
	require.False(t, developmentLicenseBypass)

	for _, licensePluginID := range []string{"example-datasource", "parent-app"} {
		t.Run(licensePluginID, func(t *testing.T) {
			licenseErr := errors.New("missing license")
			loadCalled := false

			got := applyMarketplaceLicense(
				"example-datasource", licensePluginID, licensedServeOpts(),
				func(pluginID string) *licensing.LicenseToken {
					loadCalled = true
					require.Equal(t, licensePluginID, pluginID)
					return &licensing.LicenseToken{Status: licensing.Invalid, Error: licenseErr}
				},
			)

			require.True(t, loadCalled)
			requireInvalidLicenseServeOpts(t, got, "example-datasource", licensePluginID, licenseErr)
		})
	}
}

func TestMarketplaceLicenseProductionModeServesLicensedPlugin(t *testing.T) {
	expiredWithinGrace := &licensing.LicenseToken{
		Status:         licensing.Expired,
		Error:          errors.New("token expired"),
		LicenseExpires: time.Now().Add(-24 * time.Hour).Unix(),
	}
	expiredAfterGrace := &licensing.LicenseToken{
		Status:         licensing.Expired,
		Error:          errors.New("token expired"),
		LicenseExpires: time.Now().Add(-30 * 24 * time.Hour).Unix(),
	}

	tests := []struct {
		name     string
		token    *licensing.LicenseToken
		licensed bool
	}{
		{"valid license", &licensing.LicenseToken{Status: licensing.Valid}, true},
		{"expired license within grace period", expiredWithinGrace, true},
		{"expired license after grace period", expiredAfterGrace, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serveOpts := licensedServeOpts()
			got := applyMarketplaceLicense("example-app", "example-app", serveOpts, func(string) *licensing.LicenseToken {
				return tt.token
			})
			if tt.licensed {
				require.Equal(t, serveOpts, got)
				return
			}
			requireInvalidLicenseServeOpts(t, got, "example-app", "example-app", tt.token.Error)
		})
	}
}

func TestLicensingWithoutLicenseServesLicenseErrors(t *testing.T) {
	t.Setenv(marketplaceLicenseTextEnv, "")
	t.Setenv(marketplaceLicensePathEnv, filepath.Join(t.TempDir(), "missing.jwt"))

	for _, tt := range []struct {
		name            string
		opts            Opts
		licensePluginID string
	}{
		{"own plugin ID", Opts{}, "example-app"},
		{"parent plugin ID", Opts{ParentPluginID: "parent-app"}, "parent-app"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Licensing(tt.opts)("example-app", licensedServeOpts())
			require.NoError(t, err)
			handler, ok := got.CheckHealthHandler.(*invalidLicenseHandler)
			require.True(t, ok, "CheckHealthHandler is %T", got.CheckHealthHandler)
			requireInvalidLicenseServeOpts(t, got, "example-app", tt.licensePluginID, handler.verboseError)
		})
	}
}
