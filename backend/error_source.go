package backend

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"syscall"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"

	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
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

func ErrorSourceFromHttpError(err error) ErrorSource {
	if httpclient.IsDownstreamHttpError(err) {
		return ErrorSourceDownstream
	}
	return ErrorSourcePlugin
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

	if isHTTPTimeoutError(err) || isCancelledError(err) {
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

// Implements the interface used by [errors.Is].
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

// errorSourceFromContext returns the error source stored in the context.
// If no error source is stored in the context, [DefaultErrorSource] is returned.
func errorSourceFromContext(ctx context.Context) ErrorSource {
	value, ok := ctx.Value(errorSourceCtxKey{}).(*ErrorSource)
	if ok {
		return *value
	}
	return DefaultErrorSource
}

// initErrorSource initialize the status source for the context.
func initErrorSource(ctx context.Context) context.Context {
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

func IsDownstreamHttpError(err error) bool {
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

	// Check if the error is a HTTP timeout error or a context cancelled error
	if isHTTPTimeoutError(err){
		return true
	}

	if isCancelledError(err) {	
		return true
	}

	if isConnectionResetOrRefusedError(err) {
		return true
	}

	if isDnsNotFoundError(err) {
		return true
	}

	return false
}


func isCancelledError(err error) bool {
	return errors.Is(err, context.Canceled) || grpcstatus.Code(err) == grpccodes.Canceled
}

func isHTTPTimeoutError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	return errors.Is(err, os.ErrDeadlineExceeded) // replacement for os.IsTimeout(err)
}

func isConnectionResetOrRefusedError(err error) bool {
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		if sysErr, ok := netErr.Err.(*os.SyscallError); ok {
			return sysErr.Err == syscall.ECONNRESET || sysErr.Err == syscall.ECONNREFUSED
		}
	}

	return false
}

func isDnsNotFoundError(err error) bool {
	var dnsError *net.DNSError
	if errors.As(err, &dnsError) && dnsError.IsNotFound {
		return true
	}

	return false
}