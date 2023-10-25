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
	Status backend.Status

	Err error
}

// Error implements the interface
func (r Error) Error() string {
	return r.Err.Error()
}

// Unwrap implements the interface
func (r Error) Unwrap() error {
	return r.Err
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
				return nil, &Error{Source: errorSource, Err: err}
			}
			return res, err
		})
	})
}

// PluginError will apply the source as plugin
func PluginError(err error, override bool) error {
	return SourceError(backend.ErrorSourcePlugin, err, override)
}

// DownstreamError will apply the source as downstream
func DownstreamError(err error, override bool) error {
	return SourceError(backend.ErrorSourceDownstream, err, override)
}

// SourceError returns an error with the source
// If source is already defined, it will return it, or you can override
func SourceError(source backend.ErrorSource, err error, override bool) Error {
	var sourceError Error
	if errors.As(err, &sourceError) && !override {
		// nolint:gosec
		return err.(Error) // already has a source
	}
	return Error{
		Source: source,
		Err:    err,
	}
}

// Response returns an error DataResponse given status, source of the error and message.
func Response(err error) backend.DataResponse {
	e := SourceError(backend.ErrorSourcePlugin, err, false)
	return backend.DataResponse{
		Error:       errors.New(err.Error()),
		ErrorSource: e.Source,
		Status:      e.Status,
	}
}

// FromStatus returns error source from status
func FromStatus(status backend.Status) backend.ErrorSource {
	return backend.ErrorSourceFromHTTPStatus(int(status))
}
