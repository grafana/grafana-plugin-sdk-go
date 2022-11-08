package backend

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func TestConvertToProtobufQueryDataResponse(t *testing.T) {
	frames := data.Frames{data.NewFrame("test", data.NewField("test", nil, []int64{1}))}
	tcs := []struct {
		name           string
		err            error
		status         Status
		expectedStatus int32
	}{
		{
			name:           "If a HTTP Status code is used, use backend.Status equivalent status code",
			status:         http.StatusOK,
			expectedStatus: int32(StatusOK),
		},
		{
			name:           "If a backend.Status is used, use backend.Status int code",
			status:         StatusTooManyRequests,
			expectedStatus: int32(StatusTooManyRequests),
		},
		{
			name:           "syscall.ECONNREFUSED is inferred as a Status Bad Gateway",
			err:            syscall.ECONNREFUSED,
			expectedStatus: int32(StatusBadGateway),
		},
		{
			name:           "os.ErrDeadlineExceeded is inferred as a Status Timeout",
			err:            os.ErrDeadlineExceeded,
			expectedStatus: int32(StatusTimeout),
		},
		{
			name:           "fs.ErrPermission is inferred as a Status Unauthorized",
			err:            fs.ErrPermission,
			expectedStatus: int32(StatusUnauthorized),
		},
		{
			name:           "Custom error is inferred as a Status Unknown",
			err:            fmt.Errorf("some custom error"),
			expectedStatus: int32(StatusUnknown),
		},
		{
			name:           "A wrapped error is appropriately inferred",
			err:            fmt.Errorf("wrap 2: %w", fmt.Errorf("wrap 1: %w", os.ErrDeadlineExceeded)),
			expectedStatus: int32(StatusTimeout),
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			protoRes := &QueryDataResponse{
				Responses: map[string]DataResponse{
					"A": {
						Frames: frames,
						Error:  tc.err,
						Status: tc.status,
					},
				},
			}
			qdr, err := ToProto().QueryDataResponse(protoRes)
			require.NoError(t, err)
			require.NotNil(t, qdr)
			require.NotNil(t, qdr.Responses)
			receivedStatus := qdr.Responses["A"].Status
			require.Equal(t, tc.expectedStatus, receivedStatus)
		})
	}
}
