package errorsource

import (
	"errors"
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
)

func New(err error, source backend.ErrorSource, status backend.Status) Error {
	return Error{err: err, source: source, status: status}
}

// Error captures error source and implements the error interface
type Error struct {
	source backend.ErrorSource
	status backend.Status

	err error
}

// Error implements the interface
func (r Error) Error() string {
	return r.err.Error()
}

// Unwrap implements the interface
func (r Error) Unwrap() error {
	return r.err
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
				return nil, &Error{source: errorSource, err: err}
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
		return sourceError // already has a source
	}
	return Error{
		source: source,
		err:    err,
	}
}

// Response returns an error DataResponse given status, source of the error and message.
func Response(err error) backend.DataResponse {
	var e Error
	if !errors.As(err, &e) {
		// generic error, default to "plugin" error source
		return backend.DataResponse{
			Error:       err,
			ErrorSource: backend.ErrorSourcePlugin,
			Status:      backend.Status(backend.StatusUnknown),
		}
	}
	return backend.DataResponse{
		Error:       err,
		ErrorSource: e.source,
		Status:      e.status,
	}
}

// FromStatus returns error source from status
func FromStatus(status backend.Status) backend.ErrorSource {
	return backend.ErrorSourceFromHTTPStatus(int(status))
}
