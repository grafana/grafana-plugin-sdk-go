package backend

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"syscall"
)

type Error struct {
	status ErrorStatus
	msg    string
}

/*
	Usage example:
	DataResponse.Error = backend.NewError(TooManyRequestsErrorStatus, fmt.Sprintf("Resource %s was exhausted", resource))
*/
func NewError(status ErrorStatus, msg string) Error {
	return Error{
		status: status,
		msg:    msg,
	}
}

func (e Error) Error() string {
	if e.msg == "" {
		return fmt.Sprintf("An error occurred: %s", e.Status())
	}
	return e.msg
}

func (e Error) Status() ErrorStatus {
	if e.status == "" || !e.status.isValid() {
		return UnknownErrorStatus
	}
	return e.status
}

type ErrorStatus string

const (
	UnknownErrorStatus          ErrorStatus = "Unknown"
	UnauthorizedErrorStatus     ErrorStatus = "Unauthorized"
	ForbiddenErrorStatus        ErrorStatus = "Forbidden"
	NotFoundErrorStatus         ErrorStatus = "Not found"
	TooManyRequestsErrorStatus  ErrorStatus = "Resource exhausted"
	BadRequestErrorStatus       ErrorStatus = "Bad request"
	ValidationFailedErrorStatus ErrorStatus = "Validation failed"
	InternalErrorStatus         ErrorStatus = "Internal"
	NotImplementedErrorStatus   ErrorStatus = "Not implemented"
	TimeoutErrorStatus          ErrorStatus = "Timeout"
	BadGatewayErrorStatus       ErrorStatus = "Bad gateway"
)

func (e ErrorStatus) isValid() bool {
	switch e {
	case UnknownErrorStatus, UnauthorizedErrorStatus, ForbiddenErrorStatus, NotFoundErrorStatus,
		TooManyRequestsErrorStatus, BadRequestErrorStatus, ValidationFailedErrorStatus, InternalErrorStatus,
		NotImplementedErrorStatus, TimeoutErrorStatus, BadGatewayErrorStatus:
		return true
	}
	return false
}

func (e ErrorStatus) HTTPStatus() int {
	switch e {
	case UnauthorizedErrorStatus:
		return http.StatusUnauthorized
	case ForbiddenErrorStatus:
		return http.StatusForbidden
	case NotFoundErrorStatus:
		return http.StatusNotFound
	case TimeoutErrorStatus:
		return http.StatusGatewayTimeout
	case TooManyRequestsErrorStatus:
		return http.StatusTooManyRequests
	case BadRequestErrorStatus, ValidationFailedErrorStatus:
		return http.StatusBadRequest
	case NotImplementedErrorStatus:
		return http.StatusNotImplemented
	case UnknownErrorStatus, InternalErrorStatus:
		return http.StatusInternalServerError
	case BadGatewayErrorStatus:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

func ErrorStatusFromError(err error) ErrorStatus {
	for {
		result := errorStatus(err)
		if result != UnknownErrorStatus {
			return result
		}

		if err = errors.Unwrap(err); err == nil {
			return UnknownErrorStatus
		}
	}
}

func ErrorStatusFromHTTPResponse(resp *http.Response) ErrorStatus {
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return UnauthorizedErrorStatus
	case http.StatusForbidden:
		return ForbiddenErrorStatus
	case http.StatusNotFound:
		return NotFoundErrorStatus
	case http.StatusTooManyRequests:
		return TooManyRequestsErrorStatus
	case http.StatusInternalServerError:
		return InternalErrorStatus
	case http.StatusNotImplemented, http.StatusMethodNotAllowed:
		return NotImplementedErrorStatus
	case http.StatusGatewayTimeout, http.StatusRequestTimeout:
		return TimeoutErrorStatus
	case http.StatusBadRequest:
		return BadRequestErrorStatus
	}
	return UnknownErrorStatus
}

func errorStatus(err error) ErrorStatus {
	if os.IsTimeout(err) {
		return TimeoutErrorStatus
	}
	if os.IsPermission(err) {
		return UnauthorizedErrorStatus
	}
	var (
		connErr *url.Error
		netErr  *net.OpError
	)
	if errors.Is(err, connErr) || errors.Is(err, netErr) || errors.Is(err, syscall.ECONNREFUSED) {
		return BadGatewayErrorStatus
	}
	return UnknownErrorStatus
}
