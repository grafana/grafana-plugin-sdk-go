package marketplace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace/licensing"
)

// errLicenseNotValidated is returned for a token that is neither accepted nor carries an error,
// which only happens if it was never parsed.
var errLicenseNotValidated = errors.New("license token was not validated")

// validateLicenseToken decides whether a parsed token allows the plugin to be served.
// It is shared by startup and request-time validation. It returns nil if the token is valid,
// or is expired but within its grace period (logging a warning). Otherwise it logs an error and
// returns the token's error.
func validateLicenseToken(token *licensing.LicenseToken, pluginID, licensePluginID string, logger log.Logger) error {
	if token.Accepted() {
		if token.Status == licensing.ExpiredInGracePeriod {
			logger.Warn("Marketplace license expired, update your instance soon to avoid any errors. The plugin will work until", "date", token.GraceDeadline(),
				"error", token.Error, "pluginID", pluginID, "licensePluginID", licensePluginID)
		}
		return nil
	}
	err := token.Error
	if err == nil {
		err = errLicenseNotValidated
	}
	logger.Error("Marketplace License Error, serving license errors only", "err", invalidLicenseError(licensePluginID).Error(),
		"detailed error", err.Error(), "pluginID", pluginID, "licensePluginID", licensePluginID)
	return err
}

// invalidLicenseError builds the user-facing error for a missing or invalid license.
func invalidLicenseError(licensePluginID string) error {
	//nolint:staticcheck // error to be used in grafana
	return fmt.Errorf("The Marketplace plugin %s is not available with your current subscription. To activate this plugin, please purchase a license by visiting https://grafana.com/grafana/plugins/%s", licensePluginID, licensePluginID)
}

// newInvalidLicenseHandler returns a handler that answers every request with the license error
// for licensePluginID. verboseError is the detailed validation error, only shown in health check details.
func newInvalidLicenseHandler(pluginID, licensePluginID string, verboseError error) *invalidLicenseHandler {
	return &invalidLicenseHandler{
		pluginID:     pluginID,
		err:          invalidLicenseError(licensePluginID),
		verboseError: verboseError,
	}
}

type invalidLicenseHandler struct {
	pluginID     string
	err          error
	verboseError error
}

// deniedResult is the status returned for denied requests. Grafana maps Reason
// "Forbidden" (metav1.StatusReasonForbidden) to a 403; the literal avoids a k8s dependency.
func (h *invalidLicenseHandler) deniedResult() *backend.StatusResult {
	return &backend.StatusResult{
		Status:  "Failure",
		Message: h.err.Error(),
		Reason:  "Forbidden",
		Code:    http.StatusForbidden,
	}
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
	return sender.Send(&backend.CallResourceResponse{
		Status:  http.StatusForbidden,
		Headers: map[string][]string{"Content-Type": {"text/plain; charset=utf-8"}},
		Body:    []byte(h.err.Error()),
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

// queryError is the error returned for every data query. The verbose error is
// left out on purpose: it can contain the license file path, instance URL,
// licensed subject and expiry details, and query errors reach every viewer.
// It stays in the plugin logs and in the health check details.
func (h *invalidLicenseHandler) queryError() error {
	//nolint:staticcheck // error to be used in grafana
	return errors.Join(errors.New("Marketplace License Error"), h.err)
}
