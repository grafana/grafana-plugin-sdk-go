package backend

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"syscall"
)

type Status string

const (
	// StatusUnknown implies an error that should be updated to contain
	// an accurate status code, as none has been provided.
	// HTTP status code 500.
	StatusUnknown Status = "UNKNOWN"

	// StatusOK means that the action was successful.
	// HTTP status code 200.
	StatusOK Status = "OK"

	// StatusUnauthorized means that the server does not recognize the
	// client's authentication, either because it has not been provided
	// or is invalid for the operation.
	// HTTP status code 401.
	StatusUnauthorized Status = "UNAUTHORIZED"

	// StatusForbidden means that the server refuses to perform the
	// requested action for the authenticated uer.
	// HTTP status code 403.
	StatusForbidden Status = "FORBIDDEN"

	// StatusNotFound means that the server does not have any
	// corresponding document to return to the request.
	// HTTP status code 404.
	StatusNotFound Status = "NOT_FOUND"

	// StatusTooManyRequests means that the client is rate limited
	// by the server and should back-off before trying again.
	// HTTP status code 429.
	StatusTooManyRequests Status = "TOO_MANY_REQUESTS"

	// StatusBadRequest means that the server was unable to parse the
	// parameters or payload for the request.
	// HTTP status code 400.
	StatusBadRequest Status = "BAD_REQUEST"

	// StatusValidationFailed means that the server was able to parse
	// the payload for the request, but it failed one or more validation
	// checks.
	// HTTP status code 400.
	StatusValidationFailed Status = "VALIDATION_FAILED"

	// StatusInternal means that the server acknowledges that there's
	// an error, but that there is nothing the client can do to fix it.
	// HTTP status code 500.
	StatusInternal Status = "INTERNAL"

	// StatusNotImplemented means that the server does not support the
	// requested action. Typically used during development of new
	// features.
	// HTTP status code 501.
	StatusNotImplemented Status = "NOT_IMPLEMENTED"

	// StatusTimeout means that the server did not complete the request
	// within the required time and aborted the action.
	// HTTP status code 504.
	StatusTimeout Status = "TIMEOUT"

	// StatusBadGateway means that the server, while acting as a gateway
	// or proxy, received an invalid response from the upstream server.
	// HTTP status code 502.
	StatusBadGateway Status = "BAD_GATEWAY"
)

// HTTPStatus gets the HTTP status representation of e.
func (s Status) HTTPStatus() int {
	switch s {
	case StatusOK:
		return http.StatusOK
	case StatusUnauthorized:
		return http.StatusUnauthorized
	case StatusForbidden:
		return http.StatusForbidden
	case StatusNotFound:
		return http.StatusNotFound
	case StatusTimeout:
		return http.StatusGatewayTimeout
	case StatusTooManyRequests:
		return http.StatusTooManyRequests
	case StatusBadRequest, StatusValidationFailed:
		return http.StatusBadRequest
	case StatusNotImplemented:
		return http.StatusNotImplemented
	case StatusUnknown, StatusInternal:
		return http.StatusInternalServerError
	case StatusBadGateway:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

// StatusFromError gets Status from err.
func StatusFromError(err error) Status {
	for {
		result := statusFromError(err)
		if result != StatusUnknown {
			return result
		}

		if err = errors.Unwrap(err); err == nil {
			return StatusUnknown
		}
	}
}

// StatusFromHTTPStatus gets Status from statusCode.
func StatusFromHTTPStatus(statusCode int) Status {
	switch statusCode {
	case http.StatusOK:
		return StatusOK
	case http.StatusUnauthorized:
		return StatusUnauthorized
	case http.StatusForbidden:
		return StatusForbidden
	case http.StatusNotFound:
		return StatusNotFound
	case http.StatusTooManyRequests:
		return StatusTooManyRequests
	case http.StatusInternalServerError:
		return StatusInternal
	case http.StatusNotImplemented, http.StatusMethodNotAllowed:
		return StatusNotImplemented
	case http.StatusGatewayTimeout, http.StatusRequestTimeout:
		return StatusTimeout
	case http.StatusBadRequest:
		return StatusBadRequest
	case http.StatusBadGateway:
		return StatusBadGateway
	}
	return StatusUnknown
}

func statusFromError(err error) Status {
	if os.IsTimeout(err) {
		return StatusTimeout
	}
	if os.IsPermission(err) {
		return StatusUnauthorized
	}
	var (
		connErr *url.Error
		netErr  *net.OpError
	)
	if errors.Is(err, connErr) || errors.Is(err, netErr) || errors.Is(err, syscall.ECONNREFUSED) {
		return StatusBadGateway
	}
	return StatusUnknown
}
