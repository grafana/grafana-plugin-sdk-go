package errorsource

import (
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
)

// ErrorSourceMiddleware captures error source metric
func ErrorSourceMiddleware(plugin string) httpclient.Middleware {
	return httpclient.NamedMiddlewareFunc(plugin, func(opts httpclient.Options, next http.RoundTripper) http.RoundTripper {
		return httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			res, err := next.RoundTrip(req)
			if res != nil && res.StatusCode >= 400 {
				errorSource := backend.ErrorSourceFromHTTPStatus(res.StatusCode)
				return res, &backend.PluginError{Source: errorSource, Err: err}
			}
			return res, err
		})
	})
}
