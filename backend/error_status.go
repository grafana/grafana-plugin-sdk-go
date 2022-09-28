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
	InvalidArgumentErrorStatus   ErrorStatus = "Invalid argument"
	UnauthorizedErrorStatus      ErrorStatus = "Unauthorized"
	NotFoundErrorStatus          ErrorStatus = "Not found"
	ResourceExhaustedErrorStatus ErrorStatus = "Resource exhausted"
	CancelledErrorStatus         ErrorStatus = "Cancelled"
	UnknownErrorStatus           ErrorStatus = "Unknown"
	InternalErrorStatus          ErrorStatus = "Internal"
	NotImplementedErrorStatus    ErrorStatus = "Not implemented"
	UnavailableErrorStatus       ErrorStatus = "Unavailable"
	TimeoutErrorStatus           ErrorStatus = "Timeout"
)

func ErrorStatuses() []ErrorStatus {
	return []ErrorStatus{
		InvalidArgumentErrorStatus,
		UnauthorizedErrorStatus,
		NotFoundErrorStatus,
		ResourceExhaustedErrorStatus,
		CancelledErrorStatus,
		UnknownErrorStatus,
		InternalErrorStatus,
		NotImplementedErrorStatus,
		UnavailableErrorStatus,
		TimeoutErrorStatus,
	}
}

func (e ErrorStatus) HTTPStatus() int {
	switch e {
	case UnauthorizedErrorStatus:
		return http.StatusUnauthorized
	case NotFoundErrorStatus:
		return http.StatusNotFound
	case TimeoutErrorStatus:
		return http.StatusGatewayTimeout
	case ResourceExhaustedErrorStatus:
		return http.StatusTooManyRequests
	case InvalidArgumentErrorStatus:
		return http.StatusBadRequest
	case NotImplementedErrorStatus:
		return http.StatusNotImplemented
	case UnknownErrorStatus, InternalErrorStatus:
		return http.StatusInternalServerError
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
	case http.StatusUnauthorized, http.StatusForbidden:
		return UnauthorizedErrorStatus
	case http.StatusNotFound:
		return NotFoundErrorStatus
	case http.StatusTooManyRequests:
		return ResourceExhaustedErrorStatus
	case http.StatusInternalServerError:
		return InternalErrorStatus
	case http.StatusNotImplemented, http.StatusMethodNotAllowed:
		return NotImplementedErrorStatus
	case http.StatusGatewayTimeout, http.StatusRequestTimeout:
		return TimeoutErrorStatus
	case http.StatusServiceUnavailable, http.StatusBadGateway:
		return UnavailableErrorStatus
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
		return UnavailableErrorStatus // ConnectionError
	}
	return UnknownErrorStatus
}
