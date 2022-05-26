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
	sysErr  *os.SyscallError
)

type ErrorStatus int32

const (
	Undefined ErrorStatus = iota + 1
	Timeout
	Unauthorized
	ConnectionError
)

func calculateErrorStatus(err error) ErrorStatus {
	for {
		result := errorStatus(err)
		if result != Undefined {
			return result
		}

		if err = errors.Unwrap(err); err == nil {
			return Undefined
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

	switch {
	case errors.Is(err, connErr) || errors.Is(err, netErr):
		return ConnectionError
	case errors.Is(err, sysErr):
		if sysErr.Err == syscall.ECONNREFUSED {
			return ConnectionError
		}
	}
	return Undefined
}
