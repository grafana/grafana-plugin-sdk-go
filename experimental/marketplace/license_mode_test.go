//go:build !marketplace_dev

package marketplace

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/go-plugin"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace/licensing"
)

// testQueryConversionHandler is a pointer-comparable backend.QueryConversionHandler (func values are not).
type testQueryConversionHandler struct{}

func (*testQueryConversionHandler) ConvertQueryDataRequest(context.Context, *backend.QueryDataRequest) (*backend.QueryConversionResponse, error) {
	return nil, nil
}

// licensedServeOpts returns serve options with every handler and option set, so tests can check
// what the license decorator keeps or replaces.
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
		QueryConversionHandler:  &testQueryConversionHandler{},
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
		AdmissionHandler:        handler,
		ConversionHandler:       handler,
		QueryConversionHandler:  handler,
		GRPCSettings:            backend.GRPCSettings{MaxReceiveMsgSize: 1, MaxSendMsgSize: 2},
	}, got)
}

func TestInvalidLicenseServeOptsOnlyDeniesRegisteredHandlers(t *testing.T) {
	full := licensedServeOpts()
	core := backend.ServeOpts{
		CheckHealthHandler:      full.CheckHealthHandler,
		CallResourceHandler:     full.CallResourceHandler,
		QueryDataHandler:        full.QueryDataHandler,
		QueryChunkedDataHandler: full.QueryChunkedDataHandler,
		StreamHandler:           full.StreamHandler,
	}
	withAdmission, withConversion, withQueryConversion := core, core, core
	withAdmission.AdmissionHandler = full.AdmissionHandler
	withConversion.ConversionHandler = full.ConversionHandler
	withQueryConversion.QueryConversionHandler = full.QueryConversionHandler

	tests := []struct {
		name                                            string
		opts                                            backend.ServeOpts
		admissionSet, conversionSet, queryConversionSet bool
	}{
		{"none", core, false, false, false},
		{"admission only", withAdmission, true, false, false},
		{"conversion only", withConversion, false, true, false},
		{"query conversion only", withQueryConversion, false, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := invalidLicenseServeOpts("example-app", "example-app", errors.New("missing license"), tt.opts)
			handler, ok := got.CheckHealthHandler.(*invalidLicenseHandler)
			require.True(t, ok)
			want := backend.ServeOpts{
				CheckHealthHandler:      handler,
				CallResourceHandler:     handler,
				QueryDataHandler:        handler,
				QueryChunkedDataHandler: handler,
				StreamHandler:           handler,
			}
			if tt.admissionSet {
				want.AdmissionHandler = handler
			}
			if tt.conversionSet {
				want.ConversionHandler = handler
			}
			if tt.queryConversionSet {
				want.QueryConversionHandler = handler
			}
			require.Equal(t, want, got)
		})
	}
}

func TestMarketplaceLicenseProductionModeEnforcesLicensing(t *testing.T) {
	t.Setenv("GF_DEFAULT_APP_MODE", "development")
	t.Setenv("GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS", "example-datasource")
	require.False(t, developmentLicenseBypass)

	for _, licensePluginID := range []string{"example-datasource", "parent-app"} {
		t.Run(licensePluginID, func(t *testing.T) {
			licenseErr := errors.New("missing license")
			loadCalled := false

			got := checkMarketplacePluginLicenseWithDeps(
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

	invalidRecentlyExpired := &licensing.LicenseToken{
		Status:         licensing.Invalid,
		Error:          errors.New("plugin not included in license"),
		Expires:        time.Now().Add(-24 * time.Hour).Unix(),
		LicenseExpires: time.Now().Add(-24 * time.Hour).Unix(),
	}

	tests := []struct {
		name     string
		token    *licensing.LicenseToken
		licensed bool
	}{
		{"valid license", &licensing.LicenseToken{Status: licensing.Valid}, true},
		{"expired license within grace period", expiredWithinGrace, true},
		{"expired license after grace period", expiredAfterGrace, false},
		{"invalid license with recent expiry gets no grace", invalidRecentlyExpired, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serveOpts := licensedServeOpts()
			got := checkMarketplacePluginLicenseWithDeps("example-app", "example-app", serveOpts, func(string) *licensing.LicenseToken {
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
