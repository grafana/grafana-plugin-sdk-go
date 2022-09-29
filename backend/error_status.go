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
	Status        ErrorStatus
	PublicMessage string
}

func (e ErrorDetails) Error() string {
	return e.PublicMessage
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
	CancelledErrorStatus        ErrorStatus = "Cancelled"   // TODO keep?
	BadGatewayErrorStatus       ErrorStatus = "Bad gateway" // TODO keep?
)

func ErrorStatuses() []ErrorStatus {
	return []ErrorStatus{
		UnknownErrorStatus,
		UnauthorizedErrorStatus,
		ForbiddenErrorStatus,
		NotFoundErrorStatus,
		TooManyRequestsErrorStatus,
		BadRequestErrorStatus,
		ValidationFailedErrorStatus,
		InternalErrorStatus,
		NotImplementedErrorStatus,
		TimeoutErrorStatus,
		TimeoutErrorStatus,
		CancelledErrorStatus,
		BadGatewayErrorStatus,
	}
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

func InferErrorStatusFromError(err error) ErrorStatus {
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

func InferErrorStatusFromHTTPResponse(resp *http.Response) ErrorStatus {
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
	if errors.Is(err, connErr) || errors.Is(err, netErr) || errors.Is(err, syscall.ECONNREFUSED) {
		return BadGatewayErrorStatus
	}
	return UnknownErrorStatus
}
