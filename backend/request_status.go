package backend

import (
	"context"
	"errors"
	"net"
	"os"
	"strings"

	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

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
	status := RequestStatusOK
	if err != nil {
		status = RequestStatusError
		if isCancelledError(err) {
			status = RequestStatusCancelled
		}
	}

	return status
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

func isCancelledError(err error) bool {
	return errors.Is(err, context.Canceled) || grpcstatus.Code(err) == grpccodes.Canceled
}

func isHTTPTimeoutError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	return errors.Is(err, os.ErrDeadlineExceeded) // replacement for os.IsTimeout(err)
}
