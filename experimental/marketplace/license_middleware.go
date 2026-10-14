package marketplace

import (
	"context"
	"errors"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace/licensing"
)

var errNoLicenseFound = errors.New("no license found in the request metadata")

// licenseTokenLoader returns the token to validate for the request in ctx. It never returns nil.
type licenseTokenLoader func(ctx context.Context, licensePluginID string) *licensing.LicenseToken

// licenseParser parses and validates a license token, like licensing.LoadTokenFromValue.
type licenseParser func(token, appURL, validationKeys, pluginID string) *licensing.LicenseToken

// requestLicenseLoader returns a loader that reads the license gRPC metadata of the incoming
// request and validates it with parse. It fails closed: missing or unparsable metadata yields
// an invalid token without calling parse. Environment variables and license files are never consulted.
func requestLicenseLoader(parse licenseParser) licenseTokenLoader {
	return func(ctx context.Context, licensePluginID string) *licensing.LicenseToken {
		info, ok := licensing.LicenseFromIncomingGRPCContext(ctx)
		if !ok {
			return &licensing.LicenseToken{Status: licensing.NotFound, Error: errNoLicenseFound}
		}
		return parse(info.LicenseToken, info.AppURL, info.ValidationKeys, licensePluginID)
	}
}

// licenseMiddleware validates the license on every request, and forwards the request to
// the next handler only if it is valid. Otherwise it answers like startup mode does.
type licenseMiddleware struct {
	backend.BaseHandler
	pluginID        string
	licensePluginID string
	loadToken       licenseTokenLoader
}

// newLicenseMiddleware returns a middleware that validates the license of every request
// against licensePluginID. pluginID is the ID of the running plugin.
func newLicenseMiddleware(pluginID, licensePluginID string, loadToken licenseTokenLoader) backend.HandlerMiddleware {
	return backend.HandlerMiddlewareFunc(func(next backend.Handler) backend.Handler {
		return &licenseMiddleware{
			BaseHandler:     backend.NewBaseHandler(next),
			pluginID:        pluginID,
			licensePluginID: licensePluginID,
			loadToken:       loadToken,
		}
	})
}

// refuse returns nil if the request in ctx is licensed. Otherwise it returns the handler
// that answers the request with the license error.
func (m *licenseMiddleware) refuse(ctx context.Context) *invalidLicenseHandler {
	token := m.loadToken(ctx, m.licensePluginID)
	accepted, grace := acceptLicenseToken(token, time.Now())
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
	if h := m.refuse(ctx); h != nil {
		return h.QueryData(ctx, req)
	}
	return m.BaseHandler.QueryData(ctx, req)
}

func (m *licenseMiddleware) QueryChunkedData(ctx context.Context, req *backend.QueryChunkedDataRequest, w backend.ChunkedDataWriter) error {
	if h := m.refuse(ctx); h != nil {
		return h.QueryChunkedData(ctx, req, w)
	}
	return m.BaseHandler.QueryChunkedData(ctx, req, w)
}

func (m *licenseMiddleware) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	if h := m.refuse(ctx); h != nil {
		return h.CallResource(ctx, req, sender)
	}
	return m.BaseHandler.CallResource(ctx, req, sender)
}

func (m *licenseMiddleware) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	if h := m.refuse(ctx); h != nil {
		return h.CheckHealth(ctx, req)
	}
	return m.BaseHandler.CheckHealth(ctx, req)
}

func (m *licenseMiddleware) SubscribeStream(ctx context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	if h := m.refuse(ctx); h != nil {
		return h.SubscribeStream(ctx, req)
	}
	return m.BaseHandler.SubscribeStream(ctx, req)
}

func (m *licenseMiddleware) PublishStream(ctx context.Context, req *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
	if h := m.refuse(ctx); h != nil {
		return h.PublishStream(ctx, req)
	}
	return m.BaseHandler.PublishStream(ctx, req)
}

func (m *licenseMiddleware) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	if h := m.refuse(ctx); h != nil {
		return h.RunStream(ctx, req, sender)
	}
	return m.BaseHandler.RunStream(ctx, req, sender)
}

func (m *licenseMiddleware) ValidateAdmission(ctx context.Context, req *backend.AdmissionRequest) (*backend.ValidationResponse, error) {
	if h := m.refuse(ctx); h != nil {
		return nil, h.err
	}
	return m.BaseHandler.ValidateAdmission(ctx, req)
}

func (m *licenseMiddleware) MutateAdmission(ctx context.Context, req *backend.AdmissionRequest) (*backend.MutationResponse, error) {
	if h := m.refuse(ctx); h != nil {
		return nil, h.err
	}
	return m.BaseHandler.MutateAdmission(ctx, req)
}

func (m *licenseMiddleware) ConvertObjects(ctx context.Context, req *backend.ConversionRequest) (*backend.ConversionResponse, error) {
	if h := m.refuse(ctx); h != nil {
		return nil, h.err
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
	if refused := h.middleware.refuse(ctx); refused != nil {
		return nil, refused.err
	}
	return h.next.ConvertQueryDataRequest(ctx, req)
}
