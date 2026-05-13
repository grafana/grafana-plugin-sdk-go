package marketplace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace/licensing"
)

// CheckMarketplaceLicense checks if a valid marketplace plugin license exists.
//
// Note that it will start a license error server `runInvalidLicenseServer`
func CheckMarketplaceLicense() error {
	return CheckMarketplacePluginLicense("")
}

func CheckMarketplacePluginLicense(pluginId string) error {
	token := readPluginLicense(pluginId)
	if token.Error != nil {
		backend.Logger.Error("Marketplace License Error", "error", token.Error)
		if token.Status == licensing.Expired {
			grace := time.Unix(token.LicenseExpires, 0).Add(time.Hour * 24 * 14) // 2 weeks
			if time.Now().Before(grace) {
				backend.Logger.Error("The plugin will work until", "date", grace)
				backend.Logger.Error("Update your instance soon to avoid any errors")
				return nil
			}
		}
		if err := runInvalidLicenseServer(pluginId, token.Error); err != nil {
			backend.Logger.Error(err.Error())
			os.Exit(1)
		}
		return fmt.Errorf("you do not have a valid license for the marketplace plugin %s", pluginId)
	}
	return nil
}

// readPluginLicense looks for a license in environment variables and validates it
func readPluginLicense(pluginId string) *licensing.LicenseToken {
	jwks := os.Getenv("GF_MARKETPLACE_LICENSE_VALIDATION_KEY")
	appUrl := os.Getenv("GF_MARKETPLACE_APP_URL")

	backend.Logger.Debug("Validating plugin license")
	val := os.Getenv("GF_MARKETPLACE_LICENSE_TEXT")
	if len(val) > 0 {
		backend.Logger.Debug("Parsing license token from $GF_MARKETPLACE_LICENSE_TEXT")
		tok := &licensing.LicenseToken{}
		tok.Parse(val, appUrl, jwks, pluginId)
		return tok
	}

	// Will return an error if the path is not found
	val = os.Getenv("GF_MARKETPLACE_LICENSE_PATH")
	if len(val) < 2 {
		// TODO: path traversal etc
		licenseFileName := "marketplace-" + pluginId + ".jwt"
		val = licenseFileName // default license path
		if !fileExists(val) {
			homedir, _ := os.UserHomeDir()
			val = filepath.Join(homedir, ".grafana", licenseFileName)
		}
	}
	backend.Logger.Debug("Loading license from file", "path", val)

	return licensing.LoadToken(val, appUrl, jwks, pluginId)
}

// runInvalidLicenseServer when we have an error, this will make it keep running, but returning errors
func runInvalidLicenseServer(pluginId string, verboseError error) error {
	//lint:ignore ST1005 // error to be used in grafana
	err := fmt.Errorf("The Marketplace plugin %s is not available with your current subscription. To activate this data source, please upgrade your plan by visiting https://grafana.com/pricing", pluginId)
	backend.Logger.Error("Marketplace License Error, starting error server", "err", err.Error(), "detailed error", verboseError.Error())
	handler := &invalidLicenseHandler{
		pluginId:     pluginId,
		err:          err,
		verboseError: verboseError,
	}
	return backend.Manage("invalid-license-server", backend.ServeOpts{
		QueryDataHandler:   handler,
		CheckHealthHandler: handler,
	})
}

type invalidLicenseHandler struct {
	pluginId     string
	err          error
	verboseError error
}

// CheckHealth checks if the plugin is running properly
func (h *invalidLicenseHandler) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	return &backend.CheckHealthResult{
		Status:      backend.HealthStatusError,
		Message:     "Marketplace License Error",
		JSONDetails: []byte(fmt.Sprintf(`{ "message": "%s",  "verboseMessage":"%s"  }`, h.err, strings.ReplaceAll(h.verboseError.Error(), "\n", " "))),
	}, nil
}

// QueryData queries for data.
func (h *invalidLicenseHandler) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	result := backend.NewQueryDataResponse()
	//lint:ignore ST1005 // error to be used in grafana
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
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
