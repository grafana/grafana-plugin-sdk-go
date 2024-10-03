package httpclient

import (
	"net/http"

	// this is throwing cicular dependency error - will need to refactor it
	// if we want to use it
	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

const ErrorSourceMiddlewareName = "ErrorSource"

func ErrorSourceMiddleware() Middleware {
	return NamedMiddlewareFunc(ResponseLimitMiddlewareName, func(_ Options, next http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			res, err := next.RoundTrip(req)
			if err != nil && backend.IsDownstreamHttpError(err) {
				return res, backend.DownstreamError(err) 
			}

			return res, nil
		})
	})
}