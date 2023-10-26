package backend

import (
	"errors"
	"net/http"
)

// ErrorSource type defines the source of the error
type ErrorSource string

const (
	ErrorSourcePlugin     ErrorSource = "plugin"
	ErrorSourceDownstream ErrorSource = "downstream"
)

// ErrorSourceFromStatus returns an ErrorSource based on provided HTTP status code.
func ErrorSourceFromHTTPStatus(statusCode int) ErrorSource {
	switch statusCode {
	case http.StatusMethodNotAllowed,
		http.StatusNotAcceptable,
		http.StatusPreconditionFailed,
		http.StatusRequestEntityTooLarge,
		http.StatusRequestHeaderFieldsTooLarge,
		http.StatusRequestURITooLong,
		http.StatusExpectationFailed,
		http.StatusUpgradeRequired,
		http.StatusRequestedRangeNotSatisfiable,
		http.StatusNotImplemented:
		return ErrorSourcePlugin
	}

	return ErrorSourceDownstream
}

func NewError(err error, source ErrorSource, status Status) Error {
	return Error{err: err, source: source, status: status}
}

// Error captures error source and implements the error interface
type Error struct {
	source ErrorSource
	status Status

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

// PluginError will apply the source as plugin
func PluginError(err error, override bool) error {
	return SourceError(ErrorSourcePlugin, err, override)
}

// DownstreamError will apply the source as downstream
func DownstreamError(err error, override bool) error {
	return SourceError(ErrorSourceDownstream, err, override)
}

// SourceError returns an error with the source
// If source is already defined, it will return it, or you can override
func SourceError(source ErrorSource, err error, override bool) Error {
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
func Response(err error) DataResponse {
	var e Error
	if !errors.As(err, &e) {
		// generic error, default to "plugin" error source
		return DataResponse{
			Error:       err,
			ErrorSource: ErrorSourcePlugin,
			Status:      StatusUnknown,
		}
	}
	return DataResponse{
		Error:       err,
		ErrorSource: e.source,
		Status:      e.status,
	}
}

// FromStatus returns error source from status
func FromStatus(status Status) ErrorSource {
	return ErrorSourceFromHTTPStatus(int(status))
}
