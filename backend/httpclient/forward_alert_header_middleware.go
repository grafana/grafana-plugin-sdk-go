package httpclient

import (
	"net/http"
)

// FromAlertHeaderName is the header used to mark a request as originating from
// the alerting engine. Mirrors ngalertmodels.FromAlertHeaderName in core Grafana.
const FromAlertHeaderName = "FromAlert"
const ForwardFromAlertHeaderMiddlewareName = "forward-from-alert-header"

// ForwardFromAlertHeaderMiddleware forwards the FromAlert header to the
// outgoing request when it is present in opts.Header.
func ForwardFromAlertHeaderMiddleware() Middleware {
	return NamedMiddlewareFunc(ForwardFromAlertHeaderMiddlewareName, func(opts Options, next http.RoundTripper) http.RoundTripper {
		fromAlert := opts.Header.Get(FromAlertHeaderName)
		if fromAlert == "" {
			return next
		}

		return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if req.Header.Get(FromAlertHeaderName) == "" {
				req.Header.Set(FromAlertHeaderName, fromAlert)
			}
			return next.RoundTrip(req)
		})
	})
}
