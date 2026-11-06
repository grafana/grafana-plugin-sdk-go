package marketplace

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"google.golang.org/grpc/metadata"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace/licensing"
)

var errNoLicenseFound = errors.New("no license found in the request metadata")

const upstreamLicenseHeader = "license"

// licenseInfo is the license information supplied by Grafana on each request.
// Its metadata format matches the Grafana Enterprise SDK.
type licenseInfo struct {
	LicenseToken   string `json:"licenseToken"`
	AppURL         string `json:"appURL"`
	ValidationKeys string `json:"validationKeys"`
}

// licenseFromIncomingGRPCContext returns license info from incoming gRPC metadata.
func licenseFromIncomingGRPCContext(ctx context.Context) (licenseInfo, bool) {
	md, exists := metadata.FromIncomingContext(ctx)
	if !exists {
		return licenseInfo{}, false
	}

	values := md.Get(upstreamLicenseHeader)
	if len(values) == 0 || values[0] == "" {
		return licenseInfo{}, false
	}

	var info licenseInfo
	if err := json.Unmarshal([]byte(values[0]), &info); err != nil {
		return licenseInfo{}, false
	}
	return info, true
}

// licenseParser parses and validates a license token, like licensing.LoadTokenFromValue.
type licenseParser func(token, appURL, validationKeys, pluginID string) *licensing.LicenseToken

// licenseMiddleware validates the license on every request, and forwards the request to
// the next handler only if it is valid. Otherwise it answers like startup mode does.
type licenseMiddleware struct {
	backend.BaseHandler
	pluginID        string
	licensePluginID string
	// parse is licensing.LoadTokenFromValue in production;
	// it is injectable so tests can override license validation.
	parse licenseParser
}

// newLicenseMiddleware returns a middleware that validates the license of every request
// against licensePluginID. pluginID is the ID of the running plugin.
func newLicenseMiddleware(pluginID, licensePluginID string, parse licenseParser) backend.HandlerMiddleware {
	return backend.HandlerMiddlewareFunc(func(next backend.Handler) backend.Handler {
		return &licenseMiddleware{
			BaseHandler:     backend.NewBaseHandler(next),
			pluginID:        pluginID,
			licensePluginID: licensePluginID,
			parse:           parse,
		}
	})
}

// ensureValidLicense returns nil if the request in ctx is licensed. Otherwise it returns the handler
// that answers the request with the license error.
//
// The license is read only from the request's gRPC metadata, never from environment variables
// or license files. Missing or unparsable metadata is refused without calling parse.
func (m *licenseMiddleware) ensureValidLicense(ctx context.Context) *invalidLicenseHandler {
	token := &licensing.LicenseToken{Status: licensing.NotFound, Error: errNoLicenseFound}
	if info, ok := licenseFromIncomingGRPCContext(ctx); ok {
		// parse also validates the license and sets LicenseToken.Error if it is invalid.
		token = m.parse(info.LicenseToken, info.AppURL, info.ValidationKeys, m.licensePluginID)
	}
	accepted, grace := isTokenValidOrWithinGracePeriod(token, time.Now())
	if accepted {
		if token.Error != nil {
			backend.Logger.FromContext(ctx).Debug("Marketplace license expired, the plugin will work until", "date", grace,
				"error", token.Error, "pluginID", m.pluginID, "licensePluginID", m.licensePluginID)
		}
		return nil
	}
	h := &invalidLicenseHandler{
		pluginID:     m.pluginID,
		err:          invalidLicenseError(m.licensePluginID),
		verboseError: token.Error,
	}
	backend.Logger.FromContext(ctx).Debug("Marketplace License Error, refusing request", "err", h.err.Error(),
		"detailed error", token.Error.Error(), "pluginID", m.pluginID, "licensePluginID", m.licensePluginID)
	return h
}

func (m *licenseMiddleware) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	if h := m.ensureValidLicense(ctx); h != nil {
		return h.QueryData(ctx, req)
	}
	return m.BaseHandler.QueryData(ctx, req)
}

func (m *licenseMiddleware) QueryChunkedData(ctx context.Context, req *backend.QueryChunkedDataRequest, w backend.ChunkedDataWriter) error {
	if h := m.ensureValidLicense(ctx); h != nil {
		return h.QueryChunkedData(ctx, req, w)
	}
	return m.BaseHandler.QueryChunkedData(ctx, req, w)
}

func (m *licenseMiddleware) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	if h := m.ensureValidLicense(ctx); h != nil {
		return h.CallResource(ctx, req, sender)
	}
	return m.BaseHandler.CallResource(ctx, req, sender)
}

func (m *licenseMiddleware) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	if h := m.ensureValidLicense(ctx); h != nil {
		return h.CheckHealth(ctx, req)
	}
	return m.BaseHandler.CheckHealth(ctx, req)
}

func (m *licenseMiddleware) SubscribeStream(ctx context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	if h := m.ensureValidLicense(ctx); h != nil {
		return h.SubscribeStream(ctx, req)
	}
	return m.BaseHandler.SubscribeStream(ctx, req)
}

func (m *licenseMiddleware) PublishStream(ctx context.Context, req *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
	if h := m.ensureValidLicense(ctx); h != nil {
		return h.PublishStream(ctx, req)
	}
	return m.BaseHandler.PublishStream(ctx, req)
}

func (m *licenseMiddleware) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	if h := m.ensureValidLicense(ctx); h != nil {
		return h.RunStream(ctx, req, sender)
	}
	return m.BaseHandler.RunStream(ctx, req, sender)
}

func (m *licenseMiddleware) ValidateAdmission(ctx context.Context, req *backend.AdmissionRequest) (*backend.ValidationResponse, error) {
	if h := m.ensureValidLicense(ctx); h != nil {
		return h.ValidateAdmission(ctx, req)
	}
	return m.BaseHandler.ValidateAdmission(ctx, req)
}

func (m *licenseMiddleware) MutateAdmission(ctx context.Context, req *backend.AdmissionRequest) (*backend.MutationResponse, error) {
	if h := m.ensureValidLicense(ctx); h != nil {
		return h.MutateAdmission(ctx, req)
	}
	return m.BaseHandler.MutateAdmission(ctx, req)
}

func (m *licenseMiddleware) ConvertObjects(ctx context.Context, req *backend.ConversionRequest) (*backend.ConversionResponse, error) {
	if h := m.ensureValidLicense(ctx); h != nil {
		return h.ConvertObjects(ctx, req)
	}
	return m.BaseHandler.ConvertObjects(ctx, req)
}

// licensedQueryConversionHandler validates the license before converting queries.
// The SDK does not run QueryConversionHandler through the middleware chain.
type licensedQueryConversionHandler struct {
	next       backend.QueryConversionHandler
	middleware *licenseMiddleware
}

func (h *licensedQueryConversionHandler) ConvertQueryDataRequest(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryConversionResponse, error) {
	if refused := h.middleware.ensureValidLicense(ctx); refused != nil {
		return refused.ConvertQueryDataRequest(ctx, req)
	}
	return h.next.ConvertQueryDataRequest(ctx, req)
}
