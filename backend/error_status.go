package backend

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"syscall"
)

var (
	connErr *url.Error
	netErr  *net.OpError
)

type ErrorDetails struct {
	Status  ErrorStatus
	Message string
}

type ErrorStatus int32

const (
	InvalidArgument ErrorStatus = iota + 1
	Unauthenticated
	Unauthorized // remove?
	NotFound
	ResourceExhausted
	Cancelled
	Unknown
	Internal
	NotImplemented
	Unavailable
	Timeout
)

func InferErrorStatusFromError(err error) ErrorStatus {
	for {
		result := errorStatus(err)
		if result != Unknown {
			return result
		}

		if err = errors.Unwrap(err); err == nil {
			return Unknown
		}
	}
}

func InferErrorStatusFromHTTPResponse(resp *http.Response) ErrorStatus {
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return Unauthorized
	case http.StatusNotFound:
		return NotFound
	case http.StatusTooManyRequests:
		return ResourceExhausted
	case http.StatusInternalServerError:
		return Internal
	case http.StatusNotImplemented, http.StatusMethodNotAllowed:
		return NotImplemented
	case http.StatusGatewayTimeout, http.StatusRequestTimeout:
		return Timeout
	case http.StatusServiceUnavailable, http.StatusBadGateway:
		return Unavailable
	}
	return Unknown
}

func errorStatus(err error) ErrorStatus {
	if os.IsTimeout(err) {
		return Timeout
	}
	if os.IsPermission(err) {
		return Unauthorized
	}
	if errors.Is(err, connErr) || errors.Is(err, netErr) || errors.Is(err, syscall.ECONNREFUSED) {
		return Unavailable // ConnectionError
	}
	return Unknown
}
