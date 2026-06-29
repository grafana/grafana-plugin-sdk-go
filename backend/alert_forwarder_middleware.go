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

// NewAlertForwarderMiddleware creates a new HandlerMiddleware
// that will forward plugin request headers as outgoing HTTP headers.
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

	ctx = httpclient.WithContextualMiddleware(ctx,
		httpclient.MiddlewareFunc(func(opts httpclient.Options, next http.RoundTripper) http.RoundTripper {
			return httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
				switch t := pReq.(type) {
				case *QueryDataRequest:
					if val, exists := t.Headers[FromAlertHeaderName]; exists {
						req.Header.Set(FromAlertHeaderName, val)
					}
				case *CallResourceRequest:
					if val, exists := t.Headers[FromAlertHeaderName]; exists {
						req.Header.Set(FromAlertHeaderName, val[0])
					}
				case *CheckHealthRequest:
					if val, exists := t.Headers[FromAlertHeaderName]; exists {
						req.Header.Set(FromAlertHeaderName, val)
					}
				}

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
