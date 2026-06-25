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

const forwardPluginRequestHTTPHeaders = "forward-plugin-request-http-headers"

// NewHTTPClientMiddleware creates a new HandlerMiddleware
// that will forward plugin request headers as outgoing HTTP headers.
func NewHTTPClientMiddleware() HandlerMiddleware {
	return HandlerMiddlewareFunc(func(next Handler) Handler {
		return &HTTPClientMiddleware{
			BaseHandler: NewBaseHandler(next),
		}
	})
}

type HTTPClientMiddleware struct {
	BaseHandler
}

func (m *HTTPClientMiddleware) applyHeaders(ctx context.Context, pReq any) context.Context {
	if pReq == nil {
		return ctx
	}

	mw := httpclient.NamedMiddlewareFunc(forwardPluginRequestHTTPHeaders, func(opts httpclient.Options, next http.RoundTripper) http.RoundTripper {
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

			if h, ok := pReq.(ForwardHTTPHeaders); ok {
				for k, v := range h.GetHTTPHeaders() {
					// Only set a header if it is not already set.
					if req.Header.Get(k) == "" {
						req.Header[k] = v
					}
				}
			}

			return next.RoundTrip(req)
		})
	})

	return httpclient.WithContextualMiddleware(ctx, mw)
}

func (m *HTTPClientMiddleware) QueryData(ctx context.Context, req *QueryDataRequest) (*QueryDataResponse, error) {
	if req == nil {
		return m.BaseHandler.QueryData(ctx, req)
	}

	ctx = m.applyHeaders(ctx, req)

	return m.BaseHandler.QueryData(ctx, req)
}

func (m *HTTPClientMiddleware) CallResource(ctx context.Context, req *CallResourceRequest, sender CallResourceResponseSender) error {
	if req == nil {
		return m.BaseHandler.CallResource(ctx, req, sender)
	}

	ctx = m.applyHeaders(ctx, req)

	return m.BaseHandler.CallResource(ctx, req, sender)
}

func (m *HTTPClientMiddleware) CheckHealth(ctx context.Context, req *CheckHealthRequest) (*CheckHealthResult, error) {
	if req == nil {
		return m.BaseHandler.CheckHealth(ctx, req)
	}

	ctx = m.applyHeaders(ctx, req)

	return m.BaseHandler.CheckHealth(ctx, req)
}
