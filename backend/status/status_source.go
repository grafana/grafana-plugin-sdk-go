package status

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"syscall"

	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

// Source type defines the status source.
type Source string

const (
	// SourcePlugin status originates from plugin.
	SourcePlugin Source = "plugin"

	// SourceDownstream status originates from downstream service.
	SourceDownstream Source = "downstream"

	// DefaultSource is the default [Source] that should be used when it is not explicitly set.
	DefaultSource Source = SourcePlugin
)

// IsValid return true if es is [SourceDownstream] or [SourcePlugin].
func (s Source) IsValid() bool {
	return s == SourceDownstream || s == SourcePlugin
}

// String returns the string representation of s. If s is not valid, [DefaultSource] is returned.
func (s Source) String() string {
	if !s.IsValid() {
		return string(DefaultSource)
	}

	return string(s)
}

// SourceFromHTTPError returns a [Source] based on provided error.
func SourceFromHTTPError(err error) Source {
	if IsDownstreamHTTPError(err) {
		return SourceDownstream
	}
	return SourcePlugin
}

// ErrorSourceFromStatus returns a [Source] based on provided HTTP status code.
func SourceFromHTTPStatus(statusCode int) Source {
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
		return SourcePlugin
	}

	return SourceDownstream
}

type errorWithSourceImpl struct {
	source Source
	err    error
}

// DownstreamError creates a new error with status [SourceDownstream].
func DownstreamError(err error) error {
	return errorWithSourceImpl{
		source: SourceDownstream,
		err:    err,
	}
}

// DownstreamError creates a new error with status [SourceDownstream] and formats
// according to a format specifier and returns the string as a value that satisfies error.
func DownstreamErrorf(format string, a ...any) error {
	return DownstreamError(fmt.Errorf(format, a...))
}

func (e errorWithSourceImpl) ErrorSource() Source {
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

// IsDownstreamError return true if provided error is an error with downstream source or
// a timeout error or a cancelled error.
func IsDownstreamError(err error) bool {
	e := errorWithSourceImpl{
		source: SourceDownstream,
	}
	if errors.Is(err, e) {
		return true
	}

	type errorWithSource interface {
		ErrorSource() Source
	}

	// nolint:errorlint
	if errWithSource, ok := err.(errorWithSource); ok && errWithSource.ErrorSource() == SourceDownstream {
		return true
	}

	return isHTTPTimeoutError(err) || IsCancelledError(err)
}

// IsDownstreamHTTPError return true if provided error is an error with downstream source or
// a HTTP timeout error or a cancelled error or a connection reset/refused error or dns not found error.
func IsDownstreamHTTPError(err error) bool {
	return IsDownstreamError(err) ||
		isConnectionResetOrRefusedError(err) ||
		isDNSNotFoundError(err)
}

// InCancelledError returns true if err is context.Canceled or is gRPC status Canceled.
func IsCancelledError(err error) bool {
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
		var sysErr *os.SyscallError
		if errors.As(netErr.Err, &sysErr) {
			return errors.Is(sysErr.Err, syscall.ECONNRESET) || errors.Is(sysErr.Err, syscall.ECONNREFUSED)
		}
	}

	return false
}

func isDNSNotFoundError(err error) bool {
	var dnsError *net.DNSError
	if errors.As(err, &dnsError) && dnsError.IsNotFound {
		return true
	}

	return false
}

type sourceCtxKey struct{}

// SourceFromContext returns the source stored in the context.
// If no source is stored in the context, [DefaultSource] is returned.
func SourceFromContext(ctx context.Context) Source {
	value, ok := ctx.Value(sourceCtxKey{}).(*Source)
	if ok {
		return *value
	}
	return DefaultSource
}

// InitSource initialize the source for the context.
func InitSource(ctx context.Context) context.Context {
	s := DefaultSource
	return context.WithValue(ctx, sourceCtxKey{}, &s)
}

// WithSource mutates the provided context by setting the source to
// s. If the provided context does not have a source, the context
// will not be mutated and an error returned. This means that [InitSource]
// has to be called before this function.
func WithSource(ctx context.Context, s Source) error {
	v, ok := ctx.Value(sourceCtxKey{}).(*Source)
	if !ok {
		return errors.New("the provided context does not have a status source")
	}
	*v = s
	return nil
}

// WithDownstreamSource mutates the provided context by setting the source to
// [SourceDownstream]. If the provided context does not have a source, the context
// will not be mutated and an error returned. This means that [InitSource] has to be
// called before this function.
func WithDownstreamSource(ctx context.Context) error {
	return WithSource(ctx, SourceDownstream)
}
