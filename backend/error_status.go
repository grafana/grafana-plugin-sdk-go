package backend

import (
	"errors"
	"net"
	"net/url"
	"os"
	"syscall"
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

	switch t := err.(type) {
	case *url.Error:
		return ConnectionError
	case *net.OpError:
		return ConnectionError
	case syscall.Errno:
		if t == syscall.ECONNREFUSED {
			return ConnectionError
		}
	}
	return Undefined
}
