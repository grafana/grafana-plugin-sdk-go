package errorsource

import (
	"errors"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
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

// Source provides the error source
func (r Error) Source() backend.ErrorSource {
	return r.source
}

func (r Error) ErrorSource() backend.ErrorSource {
	return r.source
}

// Options provides options for error source functions
type Options struct {
	// Override will override the error source if it already exists
	Override bool
}

// WithOverride will set the override option when creating an error source
// This will override the error source if it already exists
func WithOverride() func(*Options) {
	return func(s *Options) {
		s.Override = true
	}
}

// PluginError will apply the source as plugin
func PluginError(err error, options ...func(*Options)) error {
	return ErrorWithSource(err, backend.ErrorSourcePlugin, options...)
}

// DownstreamError will apply the source as downstream
func DownstreamError(err error, options ...func(*Options)) error {
	return ErrorWithSource(err, backend.ErrorSourceDownstream, options...)
}

// SourceError returns an error with the source
// If source is already defined, it will return it, or you can override
func ErrorWithSource(err error, source backend.ErrorSource, options ...func(*Options)) Error {
	var sourceError Error

	opts := &Options{
		// default to not override
		Override: false,
	}

	for _, o := range options {
		o(opts)
	}

	if errors.As(err, &sourceError) && !opts.Override {
		return sourceError // already has a source
	}

	return Error{
		source: source,
		err:    err,
	}
}

// Response returns an error DataResponse given status, source of the error and message.
func ResponseWithErrorSource(err error) backend.DataResponse {
	var e Error
	if !errors.As(err, &e) {
		// generic error, default to "plugin" error source
		return backend.DataResponse{
			Error:       err,
			ErrorSource: backend.ErrorSourcePlugin,
			Status:      backend.StatusUnknown,
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

// AddPluginErrorToResponse adds the error as plugin error source to the response
// if the error already has a source, the existing source will be used
func AddPluginErrorToResponse(refID string, response *backend.QueryDataResponse, err error) *backend.QueryDataResponse {
	return AddErrorToResponse(refID, response, PluginError(err))
}

// AddDownstreamErrorToResponse adds the error as downstream source to the response
// if the error already has a source, the existing source will be used
func AddDownstreamErrorToResponse(refID string, response *backend.QueryDataResponse, err error) *backend.QueryDataResponse {
	return AddErrorToResponse(refID, response, DownstreamError(err))
}

// AddErrorToResponse adds the error to the response
func AddErrorToResponse(refID string, response *backend.QueryDataResponse, err error) *backend.QueryDataResponse {
	response.Responses[refID] = ResponseWithErrorSource(err)
	return response
}
