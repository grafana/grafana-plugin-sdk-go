package errors

import (
	"net/http"
)

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

	if statusCode < 400 {
		return ErrorSourceNone
	}

	return ErrorSourceDownstream
}

// ErrorSource type defines the source of the error
type ErrorSource string

const (
	ErrorSourcePlugin     ErrorSource = "plugin"
	ErrorSourceDownstream ErrorSource = "downstream"
	ErrorSourceNone       ErrorSource = "none"
)
