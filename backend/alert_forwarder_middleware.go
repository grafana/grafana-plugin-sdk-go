package backend

import (
	"context"
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
)

// FromAlertHeaderName is the header name used to mark a request as originating
// from the alerting engine. Mirrors ngalertmodels.FromAlertHeaderName in core
// Grafana.
const FromAlertHeaderName = "FromAlert"

// NewAlertForwarderMiddleware returns a HandlerMiddleware that propagates the
// "FromAlert" HTTP header from inbound plugin requests (QueryData, CallResource,
// CheckHealth) to any outbound HTTP requests made by the plugin. This lets
// downstream data sources detect that a query originates from the alerting engine.
func NewAlertForwarderMiddleware() HandlerMiddleware {
	return HandlerMiddlewareFunc(func(next Handler) Handler {
		return &AlertForwarderMiddleware{
			BaseHandler: NewBaseHandler(next),
		}
	})
}

type AlertForwarderMiddleware struct {
	BaseHandler
}

func (m *AlertForwarderMiddleware) applyHeaders(ctx context.Context, pReq any) context.Context {
	if pReq == nil {
		return ctx
	}

	var alertVal string
	switch t := pReq.(type) {
	case *QueryDataRequest:
		if val, exists := t.Headers[FromAlertHeaderName]; exists {
			alertVal = val
		}
	case *QueryChunkedDataRequest:
		if val, exists := t.Headers[FromAlertHeaderName]; exists {
			alertVal = val
		}
	case *CallResourceRequest:
		if vals, exists := t.Headers[FromAlertHeaderName]; exists && len(vals) > 0 {
			alertVal = vals[0]
		}
	case *CheckHealthRequest:
		if val, exists := t.Headers[FromAlertHeaderName]; exists {
			alertVal = val
		}
	}

	// Only register the middleware if the header is actually set.
	if alertVal == "" {
		return ctx
	}

	ctx = httpclient.WithContextualMiddleware(ctx,
		httpclient.MiddlewareFunc(func(opts httpclient.Options, next http.RoundTripper) http.RoundTripper {
			return httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				req.Header.Set(FromAlertHeaderName, alertVal)
				return next.RoundTrip(req)
			})
		}))

	return ctx
}

func (m *AlertForwarderMiddleware) QueryData(ctx context.Context, req *QueryDataRequest) (*QueryDataResponse, error) {
	if req == nil {
		return m.BaseHandler.QueryData(ctx, req)
	}

	ctx = m.applyHeaders(ctx, req)

	return m.BaseHandler.QueryData(ctx, req)
}

func (m *AlertForwarderMiddleware) QueryChunkedData(ctx context.Context, req *QueryChunkedDataRequest, w ChunkedDataWriter) error {
	if req == nil {
		return m.BaseHandler.QueryChunkedData(ctx, req, w)
	}

	ctx = m.applyHeaders(ctx, req)
	return m.BaseHandler.QueryChunkedData(ctx, req, w)
}

func (m *AlertForwarderMiddleware) CallResource(ctx context.Context, req *CallResourceRequest, sender CallResourceResponseSender) error {
	if req == nil {
		return m.BaseHandler.CallResource(ctx, req, sender)
	}

	ctx = m.applyHeaders(ctx, req)

	return m.BaseHandler.CallResource(ctx, req, sender)
}

func (m *AlertForwarderMiddleware) CheckHealth(ctx context.Context, req *CheckHealthRequest) (*CheckHealthResult, error) {
	if req == nil {
		return m.BaseHandler.CheckHealth(ctx, req)
	}

	ctx = m.applyHeaders(ctx, req)

	return m.BaseHandler.CheckHealth(ctx, req)
}
