package errorsource

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorSource type defines the source of the error
type ErrorSource string

const (
	// ErrorSourcePlugin error originates from plugin.
	ErrorSourcePlugin ErrorSource = "plugin"

	// ErrorSourceDownstream error originates from downstream service.
	ErrorSourceDownstream ErrorSource = "downstream"

	// DefaultErrorSource is the default [ErrorSource] that should be used when it is not explicitly set.
	DefaultErrorSource ErrorSource = ErrorSourcePlugin
)

func (es ErrorSource) IsValid() bool {
	return es == ErrorSourceDownstream || es == ErrorSourcePlugin
}

// ErrorSourceFromStatus returns an [ErrorSource] based on provided HTTP status code.
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

type errorWithSourceImpl struct {
	source ErrorSource
	err    error
}

func IsDownstreamError(err error) bool {
	e := errorWithSourceImpl{
		source: ErrorSourceDownstream,
	}
	if errors.Is(err, e) {
		return true
	}

	type errorWithSource interface {
		ErrorSource() ErrorSource
	}

	// nolint:errorlint
	if errWithSource, ok := err.(errorWithSource); ok && errWithSource.ErrorSource() == ErrorSourceDownstream {
		return true
	}

	if IsHTTPTimeoutError(err) || IsCancelledError(err) {
		return true
	}

	return false
}

func DownstreamError(err error) error {
	return errorWithSourceImpl{
		source: ErrorSourceDownstream,
		err:    err,
	}
}

func DownstreamErrorf(format string, a ...any) error {
	return DownstreamError(fmt.Errorf(format, a...))
}

func (e errorWithSourceImpl) ErrorSource() ErrorSource {
	return e.source
}

func (e errorWithSourceImpl) Error() string {
	return fmt.Errorf("%s error: %w", e.source, e.err).Error()
}

// Is Implements the interface used by [errors.Is].
func (e errorWithSourceImpl) Is(err error) bool {
	if errWithSource, ok := err.(errorWithSourceImpl); ok {
		return errWithSource.ErrorSource() == e.source
	}

	return false
}

func (e errorWithSourceImpl) Unwrap() error {
	return e.err
}

type errorSourceCtxKey struct{}

// FromContext returns the error source stored in the context.
// If no error source is stored in the context, [DefaultErrorSource] is returned.
func FromContext(ctx context.Context) ErrorSource {
	value, ok := ctx.Value(errorSourceCtxKey{}).(*ErrorSource)
	if ok {
		return *value
	}
	return DefaultErrorSource
}

// InitContext initialize the status source for the context.
func InitContext(ctx context.Context) context.Context {
	s := DefaultErrorSource
	return context.WithValue(ctx, errorSourceCtxKey{}, &s)
}

// WithErrorSource mutates the provided context by setting the error source to
// s. If the provided context does not have a error source, the context
// will not be mutated and an error returned. This means that [initErrorSource]
// has to be called before this function.
func WithErrorSource(ctx context.Context, s ErrorSource) error {
	v, ok := ctx.Value(errorSourceCtxKey{}).(*ErrorSource)
	if !ok {
		return errors.New("the provided context does not have a status source")
	}
	*v = s
	return nil
}

// WithDownstreamErrorSource mutates the provided context by setting the error source to
// [ErrorSourceDownstream]. If the provided context does not have a error source, the context
// will not be mutated and an error returned. This means that [initErrorSource] has to be
// called before this function.
func WithDownstreamErrorSource(ctx context.Context) error {
	return WithErrorSource(ctx, ErrorSourceDownstream)
}

func New(err error, source ErrorSource, status Status) Error {
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

// Source provides the error source
func (r Error) Source() ErrorSource {
	return r.source
}

func (r Error) ErrorSource() ErrorSource {
	return r.source
}

// WithPluginSource will apply the source as plugin
func WithPluginSource(err error, override bool) error {
	return WithSource(ErrorSourcePlugin, err, override)
}

// WithDownstreamSource will apply the source as downstream
func WithDownstreamSource(err error, override bool) error {
	return WithSource(ErrorSourceDownstream, err, override)
}

// WithSource returns an error with the source
// If source is already defined, it will return it, or you can override
func WithSource(source ErrorSource, err error, override bool) Error {
	var sourceError Error
	if errors.As(err, &sourceError) && !override {
		return sourceError // already has a source
	}
	return Error{
		source: source,
		err:    err,
	}
}

// GetSourceAndStatus returns the error source and status set, if err has them, otherwise
// the default values ErrorSourcePlugin and StatusUnknown
func GetSourceAndStatus(err error) (source ErrorSource, status Status) {
	var e Error
	if errors.As(err, &e) {
		return e.source, e.status
	}
	// generic error, default to "plugin" error source
	return ErrorSourcePlugin, StatusUnknown

}

// FromStatus returns error source from status
func FromStatus(status Status) ErrorSource {
	return ErrorSourceFromHTTPStatus(int(status))
}

func IsCancelledError(err error) bool {
	return errors.Is(err, context.Canceled) || status.Code(err) == codes.Canceled
}

func IsHTTPTimeoutError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	return errors.Is(err, os.ErrDeadlineExceeded) // replacement for os.IsTimeout(err)
}
