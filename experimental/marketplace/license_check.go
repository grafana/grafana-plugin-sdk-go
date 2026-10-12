package marketplace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace/licensing"
)

const (
	// marketplaceLicenseValidationKeyEnv is the environment variable that holds a signed JWKS (JWS) token
	// containing validation keys used to verify marketplace plugin license tokens.
	// TODO: this is currently not passed to the plugin from Grafana.
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

// checkMarketplacePluginLicense validates the license for licensePluginID, which is
// the plugin's own ID unless it is nested, in which case it is the parent's ID.
// pluginID is the ID of the running plugin.
func checkMarketplacePluginLicense(pluginID, licensePluginID string) error {
	return checkMarketplacePluginLicenseWithDeps(pluginID, licensePluginID, readPluginLicense, runInvalidLicenseServer)
}

func checkMarketplacePluginLicenseWithDeps(
	pluginID, licensePluginID string,
	loadToken func(string) *licensing.LicenseToken,
	startInvalidLicenseServer func(string, string, error) error,
) error {
	if developmentLicenseBypass {
		backend.Logger.Warn(
			"Marketplace license checks are disabled in this development build. Do not distribute this executable.",
			"pluginId", pluginID,
		)
		return nil
	}

	token := loadToken(licensePluginID)
	if token.Error != nil {
		backend.Logger.Error("Marketplace License Error", "error", token.Error)
		if token.Status == licensing.Expired {
			// Use LicenseExpires instead of Expires for the grace period: still allows using a short-lived
			// token in case an on-prem instance loses internet connection
			grace, ok := marketplaceLicenseGracePeriod(token, time.Now())
			if ok {
				backend.Logger.Error("The plugin will work until", "date", grace)
				backend.Logger.Error("Update your instance soon to avoid any errors")
				return nil
			}
		}
		if err := startInvalidLicenseServer(pluginID, licensePluginID, token.Error); err != nil {
			backend.Logger.Error(err.Error())
			os.Exit(1)
		}
		return fmt.Errorf("you do not have a valid license for the marketplace plugin %s", licensePluginID)
	}
	return nil
}

func marketplaceLicenseGracePeriod(token *licensing.LicenseToken, now time.Time) (time.Time, bool) {
	grace := time.Unix(token.LicenseExpires, 0).Add(14 * 24 * time.Hour)
	return grace, now.Before(grace)
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

// invalidLicenseError builds the user-facing error for a missing or invalid license.
func invalidLicenseError(licensePluginID string) error {
	//nolint:staticcheck // error to be used in grafana
	return fmt.Errorf("The Marketplace plugin %s is not available with your current subscription. To activate this plugin, please purchase a license by visiting https://grafana.com/grafana/plugins/%s", licensePluginID, licensePluginID)
}

// runInvalidLicenseServer when we have an error, this will make it keep running, but returning errors.
// pluginID is the running plugin; licensePluginID is the ID the license is issued under.
func runInvalidLicenseServer(pluginID, licensePluginID string, verboseError error) error {
	err := invalidLicenseError(licensePluginID)
	backend.Logger.Error("Marketplace License Error, starting error server", "err", err.Error(), "detailed error", verboseError.Error(), "pluginID", pluginID, "licensePluginID", licensePluginID)
	handler := &invalidLicenseHandler{
		pluginID:     pluginID,
		err:          err,
		verboseError: verboseError,
	}
	return backend.Manage(pluginID, backend.ServeOpts{
		QueryDataHandler:   handler,
		CheckHealthHandler: handler,
	})
}

type invalidLicenseHandler struct {
	pluginID     string
	err          error
	verboseError error
}

// CheckHealth checks if the plugin is running properly
func (h *invalidLicenseHandler) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	details, err := json.Marshal(struct {
		Message        string `json:"message"`
		VerboseMessage string `json:"verboseMessage"`
	}{
		Message:        h.err.Error(),
		VerboseMessage: strings.ReplaceAll(h.verboseError.Error(), "\n", " "),
	})
	if err != nil {
		return nil, fmt.Errorf("marshal marketplace license health details: %w", err)
	}

	return &backend.CheckHealthResult{
		Status:      backend.HealthStatusError,
		Message:     "Marketplace License Error",
		JSONDetails: details,
	}, nil
}

// QueryData queries for data.
func (h *invalidLicenseHandler) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	result := backend.NewQueryDataResponse()
	//nolint:staticcheck // error to be used in grafana
	err := errors.Join(errors.New("Marketplace License Error"), h.err, h.verboseError)
	for _, query := range req.Queries {
		errResponse := backend.ErrDataResponse(backend.StatusUnauthorized, err.Error())
		result.Responses[query.RefID] = errResponse
	}
	return result, nil
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
