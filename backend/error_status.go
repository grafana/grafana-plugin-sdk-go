package backend

import (
	"errors"
	"net"
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
	Unauthorized
	NotFound
	ResourceExhausted
	Cancelled
	Unknown
	Internal
	NotImplemented
	Unavailable
	Timeout
)

func InferErrorStatus(err error) ErrorStatus {
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
