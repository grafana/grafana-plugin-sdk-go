package marketplace

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace/licensing"
)

const (
	// marketplaceLicenseValidationKeyEnv is the environment variable that holds one or more comma-separated
	// validation key bundles: signed JWKS (JWS) tokens containing validation keys used to verify marketplace
	// plugin license tokens. Bundles that cannot be verified are ignored; see licensing.VerifyValidationKeys.
	marketplaceLicenseValidationKeyEnv = "GF_MARKETPLACE_LICENSE_VALIDATION_KEY"

	// marketplaceAppURLEnv is the environment variable that holds the Grafana app URL
	// used when validating marketplace plugin license tokens.
	marketplaceAppURLEnv = "GF_MARKETPLACE_APP_URL"

	// marketplaceLicenseTextEnv is the environment variable that holds the raw license token text
	// for a marketplace plugin.
	marketplaceLicenseTextEnv = "GF_MARKETPLACE_LICENSE_TEXT"

	// marketplaceLicensePathEnv is the environment variable that holds the file path to the
	// marketplace plugin license JWT file.
	marketplaceLicensePathEnv = "GF_MARKETPLACE_LICENSE_PATH"
)

var validPluginID = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// applyStartupTimeLicense validates the license for licensePluginID, which is
// the plugin's own ID unless it is nested, in which case it is the parent's ID.
// pluginID is the ID of the running plugin.
//
// It returns serveOpts unchanged if the license is valid, or expired but within its
// grace period. Otherwise it returns serve options that only return the license error.
func applyStartupTimeLicense(pluginID, licensePluginID string, serveOpts backend.ServeOpts) backend.ServeOpts {
	return applyStartupTimeLicenseWithDeps(pluginID, licensePluginID, serveOpts, readPluginLicense)
}

func applyStartupTimeLicenseWithDeps(
	pluginID, licensePluginID string,
	serveOpts backend.ServeOpts,
	loadToken func(string) *licensing.LicenseToken,
) backend.ServeOpts {
	token := loadToken(licensePluginID)
	if err := validateLicenseToken(token, pluginID, licensePluginID, backend.Logger); err != nil {
		return invalidLicenseServeOpts(newInvalidLicenseHandler(pluginID, licensePluginID, err), serveOpts)
	}
	return serveOpts
}

// invalidLicenseServeOpts returns serve options whose handlers are all h, which only returns the license error.
// Admission, conversion and query conversion get license-denying handlers only if the plugin
// registered them, so Grafana keeps falling back on Unimplemented for the others. Apart from
// GRPCSettings, everything else (middlewares, ExtraPlugins) is dropped.
func invalidLicenseServeOpts(handler *invalidLicenseHandler, serveOpts backend.ServeOpts) backend.ServeOpts {
	opts := backend.ServeOpts{
		CheckHealthHandler:      handler,
		CallResourceHandler:     handler,
		QueryDataHandler:        handler,
		QueryChunkedDataHandler: handler,
		StreamHandler:           handler,
		GRPCSettings:            serveOpts.GRPCSettings,
	}
	if serveOpts.AdmissionHandler != nil {
		opts.AdmissionHandler = handler
	}
	if serveOpts.ConversionHandler != nil {
		opts.ConversionHandler = handler
	}
	if serveOpts.QueryConversionHandler != nil {
		opts.QueryConversionHandler = handler
	}
	return opts
}

// readPluginLicense looks for a license in environment variables and validates it
func readPluginLicense(pluginID string) *licensing.LicenseToken {
	jwks := os.Getenv(marketplaceLicenseValidationKeyEnv)
	appUrl := os.Getenv(marketplaceAppURLEnv)

	backend.Logger.Debug("Validating marketplace plugin license")
	val := os.Getenv(marketplaceLicenseTextEnv)
	if len(val) > 0 {
		backend.Logger.Debug("Parsing license token from $GF_MARKETPLACE_LICENSE_TEXT")
		return licensing.LoadTokenFromValue(val, appUrl, jwks, pluginID)
	}

	// Will return an error if the path is not found
	val = os.Getenv(marketplaceLicensePathEnv)
	if len(val) == 0 {
		if !validPluginID.MatchString(pluginID) {
			return &licensing.LicenseToken{
				Status: licensing.Invalid,
				Error:  fmt.Errorf("invalid marketplace plugin id %q", pluginID),
			}
		}
		// filepath.Base strips any path separators, protecting against path traversal attacks
		licenseFileName := filepath.Base("license-" + pluginID + ".jwt")
		val = licenseFileName // default license path
		if !fileExists(val) {
			if homedir, err := os.UserHomeDir(); err == nil && homedir != "" {
				val = filepath.Join(homedir, ".grafana", licenseFileName)
			}
		}
	}
	backend.Logger.Debug("Loading license from file", "path", val)

	return licensing.LoadTokenFromFile(val, appUrl, jwks, pluginID)
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
