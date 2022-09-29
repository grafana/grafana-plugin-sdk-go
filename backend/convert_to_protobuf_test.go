package backend

import (
	"fmt"
	"io/fs"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

func TestConvertToProtobufQueryDataResponse(t *testing.T) {
	frames := data.Frames{data.NewFrame("test", data.NewField("test", nil, []int64{1}))}

	tcs := []struct {
		name string
		err  error

		expectedErrorStatus pluginv2.ErrorDetails_Status
	}{
		{
			name: "If Error type is used, Error.status is the expected error status",
			err: Error{
				status: TimeoutErrorStatus,
				msg:    fmt.Errorf("something went wrong").Error(),
			},
			expectedErrorStatus: pluginv2.ErrorDetails_TIMEOUT,
		},
		{
			name:                "If Error type is used, Error.status is the expected error status",
			err:                 NewError(TooManyRequestsErrorStatus, "Resource exhausted"),
			expectedErrorStatus: pluginv2.ErrorDetails_TOO_MANY_REQUESTS,
		},
		{
			name:                "If Error is not set, a connection error status is calculated based on the Error field",
			err:                 syscall.ECONNREFUSED,
			expectedErrorStatus: pluginv2.ErrorDetails_BAD_GATEWAY,
		},
		{
			name:                "If Error is not set, a timeout error status is calculated based on the Error field",
			err:                 os.ErrDeadlineExceeded,
			expectedErrorStatus: pluginv2.ErrorDetails_TIMEOUT,
		},
		{
			name:                "If Error is not set, a unauthorized error status is calculated based on the Error field",
			err:                 fs.ErrPermission,
			expectedErrorStatus: pluginv2.ErrorDetails_UNAUTHORIZED,
		},
		{
			name:                "If Error is not set, an unknown error status is calculated based on the Error field",
			err:                 fmt.Errorf("some custom error"),
			expectedErrorStatus: pluginv2.ErrorDetails_UNKNOWN,
		},
		{
			name:                "If Error is not set, an unknown error status is calculated by unwrapping the Error field",
			err:                 fmt.Errorf("wrap 2: %w", fmt.Errorf("wrap 1: %w", os.ErrDeadlineExceeded)),
			expectedErrorStatus: pluginv2.ErrorDetails_TIMEOUT,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			protoRes := &QueryDataResponse{
				Responses: map[string]DataResponse{
					"A": {
						Frames: frames,
						Error:  tc.err,
					},
				},
			}
			qdr, err := ToProto().QueryDataResponse(protoRes)
			require.NoError(t, err)
			require.NotNil(t, qdr)
			require.NotNil(t, qdr.Responses)

			receivedStatus := qdr.Responses["A"].ErrorDetails.Status

			require.Equal(t, tc.expectedErrorStatus, receivedStatus)
		})
	}
}
