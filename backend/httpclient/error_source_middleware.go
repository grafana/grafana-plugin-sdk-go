package httpclient

import (
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend/errors"
)

// ErrorSourceMiddlewareName is the middleware name used by ErrorSourceMiddleware
const ErrorSourceMiddlewareName = "ErrorSource"

// ErrorSourceMiddleware applies Error Source to response header
func ErrorSourceMiddleware() Middleware {
	return NamedMiddlewareFunc(ErrorSourceMiddlewareName, func(opts Options, next http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			resp, err := next.RoundTrip(req)
			if resp != nil {
				errorSource := errors.GetErrorSource(resp.StatusCode)
				resp.Header.Add("Error_Source", string(errorSource))
			}
			defer func() {
				_ = resp.Body.Close()
			}()
			return resp, err
		})
	})
}
