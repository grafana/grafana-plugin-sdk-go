package backend

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/status"
)

// ErrorSource type defines the source of the error
type ErrorSource = status.Source

const (
	// ErrorSourcePlugin error originates from plugin.
	ErrorSourcePlugin = status.SourcePlugin

	// ErrorSourceDownstream error originates from downstream service.
	ErrorSourceDownstream = status.SourceDownstream

	// DefaultErrorSource is the default [ErrorSource] that should be used when it is not explicitly set.
	DefaultErrorSource = status.SourcePlugin
)

// ErrorSourceFromHTTPError returns an [ErrorSource] based on provided error.
func ErrorSourceFromHTTPError(err error) ErrorSource {
	return status.SourceFromHTTPError(err)
}

// ErrorSourceFromHTTPStatus returns an [ErrorSource] based on provided HTTP status code.
func ErrorSourceFromHTTPStatus(statusCode int) ErrorSource {
	return status.SourceFromHTTPStatus(statusCode)
}

// IsDownstreamError return true if provided error is an error with downstream source or
// a timeout error or a cancelled error.
func IsDownstreamError(err error) bool {
	return status.IsDownstreamError(err)
}

// IsDownstreamError return true if provided error is an error with downstream source or
// a HTTP timeout error or a cancelled error or a connection reset/refused error or dns not found error.
func IsDownstreamHTTPError(err error) bool {
	return status.IsDownstreamHTTPError(err)
}

func DownstreamError(err error) error {
	return status.DownstreamError(err)
}

func DownstreamErrorf(format string, a ...any) error {
	return DownstreamError(fmt.Errorf(format, a...))
}

func errorSourceFromContext(ctx context.Context) ErrorSource {
	return status.SourceFromContext(ctx)
}

// initErrorSource initialize the error source for the context.
func initErrorSource(ctx context.Context) context.Context {
	return status.InitSource(ctx)
}

// WithErrorSource mutates the provided context by setting the error source to
// s. If the provided context does not have a error source, the context
// will not be mutated and an error returned. This means that [initErrorSource]
// has to be called before this function.
func WithErrorSource(ctx context.Context, s ErrorSource) error {
	return status.WithSource(ctx, s)
}

// WithDownstreamErrorSource mutates the provided context by setting the error source to
// [ErrorSourceDownstream]. If the provided context does not have a error source, the context
// will not be mutated and an error returned. This means that [initErrorSource] has to be
// called before this function.
func WithDownstreamErrorSource(ctx context.Context) error {
	return status.WithDownstreamSource(ctx)
}
