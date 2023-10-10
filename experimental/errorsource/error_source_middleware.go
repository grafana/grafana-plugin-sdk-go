package errorsource

import (
	"errors"
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
)

// Error captures error source and implements the error interface
type Error struct {
	Source backend.ErrorSource

	Err error
}

func (r Error) Error() string {
	return r.Err.Error()
}

// Middleware captures error source metric
func Middleware(plugin string) httpclient.Middleware {
	return httpclient.NamedMiddlewareFunc(plugin, func(opts httpclient.Options, next http.RoundTripper) http.RoundTripper {
		return httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			res, err := next.RoundTrip(req)
			if res != nil && res.StatusCode >= 400 {
				errorSource := backend.ErrorSourceFromHTTPStatus(res.StatusCode)
				if err == nil {
					err = errors.New(res.Status)
				}
				return res, &Error{Source: errorSource, Err: err}
			}
			return res, err
		})
	})
}
