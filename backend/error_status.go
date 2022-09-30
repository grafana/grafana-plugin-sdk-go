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

// Error represents an error with a status and message. It is
// immutable and should be created with NewError.
type Error struct {
	status ErrorStatus
	msg    string
}

// NewError returns an Error representing status and msg.
func NewError(status ErrorStatus, msg string) Error {
	return Error{
		status: status,
		msg:    msg,
	}
}

// Error implements the error interface.
func (e Error) Error() string {
	if e.msg == "" {
		return fmt.Sprintf("An error occurred: %s", e.Status())
	}
	return e.msg
}

// Status returns the ErrorStatus contained in e.
func (e Error) Status() ErrorStatus {
	if e.status == "" || !e.status.isValid() {
		return ErrorStatusUnknown
	}
	return e.status
}

type ErrorStatus string

const (
	// ErrorStatusUnknown implies an error that should be updated to contain
	// an accurate status code, as none has been provided.
	// HTTP status code 500.
	ErrorStatusUnknown ErrorStatus = "Unknown"

	// ErrorStatusUnauthorized means that the server does not recognize the
	// client's authentication, either because it has not been provided
	// or is invalid for the operation.
	// HTTP status code 401.
	ErrorStatusUnauthorized ErrorStatus = "Unauthorized"

	// ErrorStatusForbidden means that the server refuses to perform the
	// requested action for the authenticated uer.
	// HTTP status code 403.
	ErrorStatusForbidden ErrorStatus = "Forbidden"

	// ErrorStatusNotFound means that the server does not have any
	// corresponding document to return to the request.
	// HTTP status code 404.
	ErrorStatusNotFound ErrorStatus = "Not found"

	// ErrorStatusTooManyRequests means that the client is rate limited
	// by the server and should back-off before trying again.
	// HTTP status code 429.
	ErrorStatusTooManyRequests ErrorStatus = "Resource exhausted"

	// ErrorStatusBadRequest means that the server was unable to parse the
	// parameters or payload for the request.
	// HTTP status code 400.
	ErrorStatusBadRequest ErrorStatus = "Bad request"

	// ErrorStatusValidationFailed means that the server was able to parse
	// the payload for the request but it failed one or more validation
	// checks.
	// HTTP status code 400.
	ErrorStatusValidationFailed ErrorStatus = "Validation failed"

	// ErrorStatusInternal means that the server acknowledges that there's
	// an error, but that there is nothing the client can do to fix it.
	// HTTP status code 500.
	ErrorStatusInternal ErrorStatus = "Internal"

	// ErrorStatusNotImplemented means that the server does not support the
	// requested action. Typically used during development of new
	// features.
	// HTTP status code 501.
	ErrorStatusNotImplemented ErrorStatus = "Not implemented"

	// ErrorStatusTimeout means that the server did not complete the request
	// within the required time and aborted the action.
	// HTTP status code 504.
	ErrorStatusTimeout ErrorStatus = "Timeout"

	// ErrorStatusBadGateway means that the server, while acting as a gateway
	// or proxy, received an invalid response from the upstream server.
	// HTTP status code 502.
	ErrorStatusBadGateway ErrorStatus = "Bad gateway"
)

func (e ErrorStatus) isValid() bool {
	switch e {
	case ErrorStatusUnknown, ErrorStatusUnauthorized, ErrorStatusForbidden, ErrorStatusNotFound,
		ErrorStatusTooManyRequests, ErrorStatusBadRequest, ErrorStatusValidationFailed, ErrorStatusInternal,
		ErrorStatusNotImplemented, ErrorStatusTimeout, ErrorStatusBadGateway:
		return true
	}
	return false
}

// HTTPStatus gets the HTTP status representation of e.
func (e ErrorStatus) HTTPStatus() int {
	switch e {
	case ErrorStatusUnauthorized:
		return http.StatusUnauthorized
	case ErrorStatusForbidden:
		return http.StatusForbidden
	case ErrorStatusNotFound:
		return http.StatusNotFound
	case ErrorStatusTimeout:
		return http.StatusGatewayTimeout
	case ErrorStatusTooManyRequests:
		return http.StatusTooManyRequests
	case ErrorStatusBadRequest, ErrorStatusValidationFailed:
		return http.StatusBadRequest
	case ErrorStatusNotImplemented:
		return http.StatusNotImplemented
	case ErrorStatusUnknown, ErrorStatusInternal:
		return http.StatusInternalServerError
	case ErrorStatusBadGateway:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

// ErrorStatusFromError gets ErrorStatus from err.
func ErrorStatusFromError(err error) ErrorStatus {
	for {
		result := errorStatus(err)
		if result != ErrorStatusUnknown {
			return result
		}

		if err = errors.Unwrap(err); err == nil {
			return ErrorStatusUnknown
		}
	}
}

// ErrorStatusFromHTTPResponse gets ErrorStatus from resp.
func ErrorStatusFromHTTPResponse(resp *http.Response) ErrorStatus {
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return ErrorStatusUnauthorized
	case http.StatusForbidden:
		return ErrorStatusForbidden
	case http.StatusNotFound:
		return ErrorStatusNotFound
	case http.StatusTooManyRequests:
		return ErrorStatusTooManyRequests
	case http.StatusInternalServerError:
		return ErrorStatusInternal
	case http.StatusNotImplemented, http.StatusMethodNotAllowed:
		return ErrorStatusNotImplemented
	case http.StatusGatewayTimeout, http.StatusRequestTimeout:
		return ErrorStatusTimeout
	case http.StatusBadRequest:
		return ErrorStatusBadRequest
	}
	return ErrorStatusUnknown
}

func errorStatus(err error) ErrorStatus {
	if os.IsTimeout(err) {
		return ErrorStatusTimeout
	}
	if os.IsPermission(err) {
		return ErrorStatusUnauthorized
	}
	var (
		connErr *url.Error
		netErr  *net.OpError
	)
	if errors.Is(err, connErr) || errors.Is(err, netErr) || errors.Is(err, syscall.ECONNREFUSED) {
		return ErrorStatusBadGateway
	}
	return ErrorStatusUnknown
}
