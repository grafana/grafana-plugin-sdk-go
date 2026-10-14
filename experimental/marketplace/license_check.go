package marketplace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
//
// It returns serveOpts unchanged if the license is valid, or expired but within its
// grace period. Otherwise it returns serve options that only return the license error.
func checkMarketplacePluginLicense(pluginID, licensePluginID string, serveOpts backend.ServeOpts) backend.ServeOpts {
	return checkMarketplacePluginLicenseWithDeps(pluginID, licensePluginID, serveOpts, readPluginLicense)
}

func checkMarketplacePluginLicenseWithDeps(
	pluginID, licensePluginID string,
	serveOpts backend.ServeOpts,
	loadToken func(string) *licensing.LicenseToken,
) backend.ServeOpts {
	if developmentLicenseBypass {
		backend.Logger.Warn(
			"Marketplace license checks are disabled in this development build. Do not distribute this executable.",
			"pluginId", pluginID,
		)
		return serveOpts
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
				return serveOpts
			}
		}
		return invalidLicenseServeOpts(pluginID, licensePluginID, token.Error, serveOpts)
	}
	return serveOpts
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

// invalidLicenseServeOpts returns serve options whose handlers only return the license error.
// Only GRPCSettings are kept from serveOpts: every other handler, ExtraPlugins included, is
// dropped so that nothing is served without a license.
// pluginID is the running plugin; licensePluginID is the ID the license is issued under.
func invalidLicenseServeOpts(pluginID, licensePluginID string, verboseError error, serveOpts backend.ServeOpts) backend.ServeOpts {
	err := invalidLicenseError(licensePluginID)
	backend.Logger.Error("Marketplace License Error, serving license errors only", "err", err.Error(), "detailed error", verboseError.Error(), "pluginID", pluginID, "licensePluginID", licensePluginID)
	handler := &invalidLicenseHandler{
		pluginID:     pluginID,
		err:          err,
		verboseError: verboseError,
	}
	return backend.ServeOpts{
		CheckHealthHandler:      handler,
		CallResourceHandler:     handler,
		QueryDataHandler:        handler,
		QueryChunkedDataHandler: handler,
		StreamHandler:           handler,
		GRPCSettings:            serveOpts.GRPCSettings,
	}
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
	err := h.queryError()
	for _, query := range req.Queries {
		errResponse := backend.ErrDataResponse(backend.StatusUnauthorized, err.Error())
		result.Responses[query.RefID] = errResponse
	}
	return result, nil
}

// QueryChunkedData returns the license error for every query.
func (h *invalidLicenseHandler) QueryChunkedData(ctx context.Context, req *backend.QueryChunkedDataRequest, w backend.ChunkedDataWriter) error {
	err := h.queryError()
	for _, query := range req.Queries {
		if writeErr := w.WriteError(ctx, query.RefID, backend.StatusUnauthorized, err); writeErr != nil {
			return writeErr
		}
	}
	return nil
}

// CallResource returns the license error for every resource call.
// It responds with 403 Forbidden rather than 401 Unauthorized, so that the Grafana
// frontend does not treat the response as an expired user session.
func (h *invalidLicenseHandler) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	body, err := json.Marshal(struct {
		Message string `json:"message"`
	}{
		Message: h.err.Error(),
	})
	if err != nil {
		return fmt.Errorf("marshal marketplace license resource response: %w", err)
	}
	return sender.Send(&backend.CallResourceResponse{
		Status:  http.StatusForbidden,
		Headers: map[string][]string{"Content-Type": {"application/json"}},
		Body:    body,
	})
}

// SubscribeStream denies every stream subscription.
func (h *invalidLicenseHandler) SubscribeStream(ctx context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	return &backend.SubscribeStreamResponse{Status: backend.SubscribeStreamStatusPermissionDenied}, nil
}

// PublishStream denies every stream publication.
func (h *invalidLicenseHandler) PublishStream(ctx context.Context, req *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
	return &backend.PublishStreamResponse{Status: backend.PublishStreamStatusPermissionDenied}, nil
}

// RunStream returns the license error for every stream.
func (h *invalidLicenseHandler) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	return h.err
}

// queryError is the error returned for every data query.
func (h *invalidLicenseHandler) queryError() error {
	//nolint:staticcheck // error to be used in grafana
	return errors.Join(errors.New("Marketplace License Error"), h.err, h.verboseError)
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
