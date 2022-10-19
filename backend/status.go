package backend

import "net/http"

type Status int

const (
	// StatusUnknown implies an error that should be updated to contain
	// an accurate status code, as none has been provided.
	// HTTP status code 500.
	StatusUnknown Status = http.StatusInternalServerError

	// StatusOK means that the action was successful.
	// HTTP status code 200.
	StatusOK Status = http.StatusOK

	// StatusUnauthorized means that the server does not recognize the
	// client's authentication, either because it has not been provided
	// or is invalid for the operation.
	// HTTP status code 401.
	StatusUnauthorized Status = http.StatusUnauthorized

	// StatusForbidden means that the server refuses to perform the
	// requested action for the authenticated uer.
	// HTTP status code 403.
	StatusForbidden Status = http.StatusForbidden

	// StatusNotFound means that the server does not have any
	// corresponding document to return to the request.
	// HTTP status code 404.
	StatusNotFound Status = http.StatusNotFound

	// StatusTooManyRequests means that the client is rate limited
	// by the server and should back-off before trying again.
	// HTTP status code 429.
	StatusTooManyRequests Status = http.StatusTooManyRequests

	// StatusBadRequest means that the server was unable to parse the
	// parameters or payload for the request.
	// HTTP status code 400.
	StatusBadRequest Status = http.StatusBadRequest

	// StatusValidationFailed means that the server was able to parse
	// the payload for the request, but it failed one or more validation
	// checks.
	// HTTP status code 400.
	StatusValidationFailed Status = http.StatusBadRequest

	// StatusInternal means that the server acknowledges that there's
	// an error, but that there is nothing the client can do to fix it.
	// HTTP status code 500.
	StatusInternal Status = http.StatusInternalServerError

	// StatusNotImplemented means that the server does not support the
	// requested action. Typically used during development of new
	// features.
	// HTTP status code 501.
	StatusNotImplemented Status = http.StatusNotImplemented

	// StatusTimeout means that the server did not complete the request
	// within the required time and aborted the action.
	// HTTP status code 504.
	StatusTimeout Status = http.StatusGatewayTimeout

	// StatusBadGateway means that the server, while acting as a gateway
	// or proxy, received an invalid response from the upstream server.
	// HTTP status code 502.
	StatusBadGateway Status = http.StatusBadGateway
)

func (s Status) String() string {
	if ss := http.StatusText(int(s)); ss != "" {
		return ss
	}
	return StatusUnknown.String()
}
