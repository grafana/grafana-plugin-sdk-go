package backend

import "net/http"

// ErrorSource type defines the source of the error
type ErrorSource string

const (
	ErrorSourcePlugin     ErrorSource = "plugin"
	ErrorSourceDownstream ErrorSource = "downstream"
)

// GetErrorSource returns a error source based on http status code
func GetErrorSource(statusCode int) ErrorSource {
	switch statusCode {
	case http.StatusBadRequest,
		http.StatusMethodNotAllowed,
		http.StatusNotAcceptable,
		http.StatusPreconditionFailed,
		http.StatusRequestEntityTooLarge,
		http.StatusRequestHeaderFieldsTooLarge,
		http.StatusRequestURITooLong,
		http.StatusExpectationFailed,
		http.StatusUpgradeRequired,
		http.StatusRequestedRangeNotSatisfiable,
		http.StatusNotImplemented:
		return ErrorSourcePlugin
	}

	return ErrorSourceDownstream
}
