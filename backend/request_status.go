package backend

import (
	"context"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/status"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

type RequestStatus int

const (
	RequestStatusOK RequestStatus = iota
	RequestStatusCancelled
	RequestStatusError
)

func (status RequestStatus) String() string {
	names := [...]string{"ok", "cancelled", "error"}
	if status < RequestStatusOK || status > RequestStatusError {
		return ""
	}

	return names[status]
}

func RequestStatusFromError(err error) RequestStatus {
	requestStatus := RequestStatusOK
	if err != nil {
		requestStatus = RequestStatusError
		if status.IsCancelledError(err) {
			requestStatus = RequestStatusCancelled
		}
	}

	return requestStatus
}

func RequestStatusFromErrorString(errString string) RequestStatus {
	status := RequestStatusOK
	if errString != "" {
		status = RequestStatusError
		if strings.Contains(errString, context.Canceled.Error()) || strings.Contains(errString, "code = Canceled") {
			status = RequestStatusCancelled
		}
	}

	return status
}

func RequestStatusFromQueryDataResponse(res *QueryDataResponse, err error) RequestStatus {
	if err != nil {
		return RequestStatusFromError(err)
	}

	status := RequestStatusOK

	if res != nil {
		for _, dr := range res.Responses {
			if dr.Error != nil {
				s := RequestStatusFromError(dr.Error)
				if s > status {
					status = s
				}

				if status == RequestStatusError {
					break
				}
			}
		}
	}

	return status
}

func RequestStatusFromProtoQueryDataResponse(res *pluginv2.QueryDataResponse, err error) RequestStatus {
	if err != nil {
		return RequestStatusFromError(err)
	}

	status := RequestStatusOK

	if res != nil {
		for _, dr := range res.Responses {
			if dr.Error != "" {
				s := RequestStatusFromErrorString(dr.Error)
				if s > status {
					status = s
				}

				if status == RequestStatusError {
					break
				}
			}
		}
	}

	return status
}

// RequestStatusFromHTTPStatus maps an HTTP status code to a RequestStatus.
// It returns RequestStatusError for 4xx and 5xx codes, otherwise RequestStatusOK.
func RequestStatusFromHTTPStatus(statusCode int) RequestStatus {
	if statusCode >= 400 {
		return RequestStatusError
	}
	return RequestStatusOK
}

// RequestStatusFromCheckHealthResult returns a RequestStatus based on the
// health check result and error. A returned error takes precedence; otherwise
// a result with HealthStatusError is reported as RequestStatusError.
func RequestStatusFromCheckHealthResult(res *CheckHealthResult, err error) RequestStatus {
	if err != nil {
		return RequestStatusFromError(err)
	}
	if res != nil && res.Status == HealthStatusError {
		return RequestStatusError
	}
	return RequestStatusOK
}

// callResourceStatusCapturingSender wraps a CallResourceResponseSender and
// records the HTTP status code of the first response sent.
type callResourceStatusCapturingSender struct {
	sender     CallResourceResponseSender
	statusCode int
}

func newCallResourceStatusCapturingSender(sender CallResourceResponseSender) *callResourceStatusCapturingSender {
	// -1 indicates that no response has been sent yet.
	return &callResourceStatusCapturingSender{sender: sender, statusCode: -1}
}

func (s *callResourceStatusCapturingSender) Send(res *CallResourceResponse) error {
	if s.statusCode == -1 && res != nil {
		s.statusCode = res.Status
	}
	return s.sender.Send(res)
}

// requestStatus returns the RequestStatus for the call. A returned error takes
// precedence; otherwise the status code of the first response is used.
func (s *callResourceStatusCapturingSender) requestStatus(err error) RequestStatus {
	if err != nil || s.statusCode < 0 {
		return RequestStatusFromError(err)
	}
	return RequestStatusFromHTTPStatus(s.statusCode)
}
