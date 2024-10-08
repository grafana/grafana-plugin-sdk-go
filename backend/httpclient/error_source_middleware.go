package httpclient

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

const ErrorSourceMiddlewareName = "ErrorSource"

func ErrorSourceMiddleware() Middleware {
	return NamedMiddlewareFunc(ResponseLimitMiddlewareName, func(_ Options, next http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			res, err := next.RoundTrip(req)
			if err != nil && IsDownstreamHTTPError(err) {
				return res, DownstreamError(err)
			}

			return res, err
		})
	})
}

type ErrorSource string

const (
	ErrorSourcePlugin     ErrorSource = "plugin"
	ErrorSourceDownstream ErrorSource = "downstream"
)

type errorWithSourceImpl struct {
	source ErrorSource
	err    error
}

func IsDownstreamHTTPError(err error) bool {
	e := errorWithSourceImpl{
		source: ErrorSourceDownstream,
	}
	if errors.Is(err, e) {
		return true
	}

	// nolint:errorlint
	if errWithSource, ok := err.(errorWithSourceImpl); ok && errWithSource.ErrorSource() == ErrorSourceDownstream {
		return true
	}

	// Check if the error is a HTTP timeout error or a context cancelled error
	if isHTTPTimeoutError(err) {
		return true
	}

	if isCancelledError(err) {
		return true
	}

	if isConnectionResetOrRefusedError(err) {
		return true
	}

	if isDNSNotFoundError(err) {
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

func DownstreamError(err error) error {
	return errorWithSourceImpl{
		source: ErrorSourceDownstream,
		err:    err,
	}
}
