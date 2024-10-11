package httpclient

import (
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/status"
)

// ErrorSourceMiddlewareName is the middleware name used by ErrorSourceMiddleware.
const ErrorSourceMiddlewareName = "ErrorSource"

// ErrorSourceMiddleware inspect the response error and wraps it in a [status.DownstreamError] if [status.IsDownstreamHTTPError] returns true.
func ErrorSourceMiddleware() Middleware {
	return NamedMiddlewareFunc(ErrorSourceMiddlewareName, func(_ Options, next http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			res, err := next.RoundTrip(req)
			if err != nil && status.IsDownstreamHTTPError(err) {
				return res, status.DownstreamError(err)
			}

			return res, err
		})
	})
}
