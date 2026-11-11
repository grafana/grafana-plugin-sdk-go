package backend

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func TestRequestStatus(t *testing.T) {
	tcs := []struct {
		s             RequestStatus
		expectedLabel string
	}{
		{
			s:             RequestStatusOK,
			expectedLabel: "ok",
		},
		{
			s:             RequestStatusError,
			expectedLabel: "error",
		},
		{
			s:             RequestStatusCancelled,
			expectedLabel: "cancelled",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.s.String(), func(t *testing.T) {
			require.Equal(t, tc.expectedLabel, tc.s.String())
			require.Equal(t, tc.expectedLabel, fmt.Sprint(tc.s))
		})
	}
}

func TestRequestStatusFromError(t *testing.T) {
	tcs := []struct {
		desc           string
		err            error
		expectedStatus RequestStatus
	}{
		{
			desc:           "no error should be status ok",
			err:            nil,
			expectedStatus: RequestStatusOK,
		},
		{
			desc:           "error should be status error",
			err:            errors.New("boom"),
			expectedStatus: RequestStatusError,
		},
		{
			desc:           "context canceled should be status cancelled",
			err:            context.Canceled,
			expectedStatus: RequestStatusCancelled,
		},
		{
			desc:           "gRPC canceled should be status cancelled",
			err:            status.Error(codes.Canceled, "canceled"),
			expectedStatus: RequestStatusCancelled,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			status := RequestStatusFromError(tc.err)
			require.Equal(t, tc.expectedStatus, status)
		})
	}
}

func TestRequestStatusFromQueryDataResponse(t *testing.T) {
	responseWithoutError := NewQueryDataResponse()
	responseWithoutError.Responses["A"] = DataResponse{
		Frames: data.Frames{data.NewFrame("test")},
	}

	responseWithError := NewQueryDataResponse()
	responseWithError.Responses["A"] = DataResponse{
		Error: errors.New("boom"),
	}
	responseWithMultipleErrors := NewQueryDataResponse()
	responseWithMultipleErrors.Responses["A"] = DataResponse{
		Error: context.Canceled,
	}
	responseWithMultipleErrors.Responses["B"] = DataResponse{
		Frames: data.Frames{data.NewFrame("test")},
	}
	responseWithMultipleErrors.Responses["C"] = DataResponse{
		Error: errors.New("boom"),
	}

	tcs := []struct {
		desc           string
		resp           *QueryDataResponse
		err            error
		expectedStatus RequestStatus
	}{
		{
			desc:           "no error should be status ok",
			err:            nil,
			expectedStatus: RequestStatusOK,
		},
		{
			desc:           "error should be status error",
			err:            errors.New("boom"),
			expectedStatus: RequestStatusError,
		},
		{
			desc:           "context canceled should be status cancelled",
			err:            context.Canceled,
			expectedStatus: RequestStatusCancelled,
		},
		{
			desc:           "response without error should be status ok",
			resp:           responseWithoutError,
			expectedStatus: RequestStatusOK,
		},
		{
			desc:           "response with error should be status error",
			resp:           responseWithError,
			expectedStatus: RequestStatusError,
		},
		{
			desc:           "response with multiple error should pick the highest status cancelled",
			resp:           responseWithMultipleErrors,
			expectedStatus: RequestStatusError,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			status := RequestStatusFromQueryDataResponse(tc.resp, tc.err)
			require.Equal(t, tc.expectedStatus, status)
		})
	}
}

func TestRequestStatusFromErrorString(t *testing.T) {
	tcs := []struct {
		desc           string
		err            string
		expectedStatus RequestStatus
	}{
		{
			desc:           "no error should be status ok",
			err:            "",
			expectedStatus: RequestStatusOK,
		},
		{
			desc:           "error should be status error",
			err:            errors.New("boom").Error(),
			expectedStatus: RequestStatusError,
		},
		{
			desc:           "context canceled should be status cancelled",
			err:            context.Canceled.Error(),
			expectedStatus: RequestStatusCancelled,
		},
		{
			desc:           "gRPC canceled should be status cancelled",
			err:            status.Error(codes.Canceled, "canceled").Error(),
			expectedStatus: RequestStatusCancelled,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			status := RequestStatusFromErrorString(tc.err)
			require.Equal(t, tc.expectedStatus, status)
		})
	}
}

func TestRequestStatusFromHTTPStatus(t *testing.T) {
	for code, expected := range map[int]RequestStatus{
		200: RequestStatusOK,
		204: RequestStatusOK,
		302: RequestStatusOK,
		399: RequestStatusOK,
		400: RequestStatusError,
		404: RequestStatusError,
		500: RequestStatusError,
		503: RequestStatusError,
	} {
		require.Equal(t, expected, RequestStatusFromHTTPStatus(code), "status code %d", code)
	}
}

func TestRequestStatusFromCheckHealthResult(t *testing.T) {
	require.Equal(t, RequestStatusOK, RequestStatusFromCheckHealthResult(&CheckHealthResult{Status: HealthStatusOk}, nil))
	require.Equal(t, RequestStatusOK, RequestStatusFromCheckHealthResult(&CheckHealthResult{Status: HealthStatusUnknown}, nil))
	require.Equal(t, RequestStatusOK, RequestStatusFromCheckHealthResult(nil, nil))
	require.Equal(t, RequestStatusError, RequestStatusFromCheckHealthResult(&CheckHealthResult{Status: HealthStatusError}, nil))
	require.Equal(t, RequestStatusError, RequestStatusFromCheckHealthResult(&CheckHealthResult{Status: HealthStatusOk}, errors.New("boom")))
	require.Equal(t, RequestStatusCancelled, RequestStatusFromCheckHealthResult(nil, context.Canceled))
}
