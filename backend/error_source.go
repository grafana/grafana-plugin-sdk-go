package backend

import "net/http"

// ErrorSource type defines the source of the error
type ErrorSource string

const (
	ErrorSourcePlugin     ErrorSource = "plugin"
	ErrorSourceDownstream ErrorSource = "downstream"
)

type PluginError struct {
	Source ErrorSource

	Err error
}

func (r PluginError) Error() string {
	return r.Err.Error()
}

// ErrorSourceFromStatus returns an ErrorSource based on provided HTTP status code.
func ErrorSourceFromHTTPStatus(statusCode int) ErrorSource {
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
