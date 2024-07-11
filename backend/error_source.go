package backend

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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

type ErrorWithSource interface {
	ErrorSource() ErrorSource
}

type errorWithSource struct {
	source ErrorSource
	err    error
}

func DownstreamError(err error) error {
	return &errorWithSource{
		source: ErrorSourceDownstream,
		err:    err,
	}
}

func DownstreamErrorf(format string, a ...any) error {
	return &errorWithSource{
		source: ErrorSourceDownstream,
		err:    fmt.Errorf(format, a...),
	}
}

func (e errorWithSource) ErrorSource() ErrorSource {
	return e.source
}

func (e errorWithSource) Error() string {
	return fmt.Errorf("%s error: %w", e.source, e.err).Error()
}

func (e errorWithSource) Unwrap() error {
	return e.err
}

type errorSourceCtxKey struct{}

// ErrorSourceFromContext returns the error source stored in the context.
// If no error source is stored in the context, [DefaultErrorSource] is returned.
func ErrorSourceFromContext(ctx context.Context) ErrorSource {
	value, ok := ctx.Value(errorSourceCtxKey{}).(*ErrorSource)
	if ok {
		return *value
	}
	return DefaultErrorSource
}

// InitErrorSource initialize the status source for the context.
func InitErrorSource(ctx context.Context) context.Context {
	s := DefaultErrorSource
	return context.WithValue(ctx, errorSourceCtxKey{}, &s)
}

// WithErrorSource mutates the provided context by setting the error source to
// s. If the provided context does not have a error source, the context
// will not be mutated and an error returned. This means that [InitErrorSource]
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
// will not be mutated and an error returned. This means that [InitErrorSource] has to be
// called before this function.
func WithDownstreamErrorSource(ctx context.Context) error {
	return WithErrorSource(ctx, ErrorSourceDownstream)
}
