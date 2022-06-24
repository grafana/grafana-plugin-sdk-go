package backend

import (
	"fmt"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/require"
	"io/fs"
	"os"
	"syscall"
	"testing"
)

func TestConvertToProtobufQueryDataResponse(t *testing.T) {
	frames := data.Frames{data.NewFrame("test", data.NewField("test", nil, []int64{1}))}

	tcs := []struct {
		name       string
		err        error
		errDetails *ErrorDetails

		expectedErrorStatus ErrorStatus
	}{
		{
			name: "If ErrorDetails is set, the ErrorDetails status is the expected error status",
			err:  fmt.Errorf("something went wrong"),
			errDetails: &ErrorDetails{
				Status: Timeout,
			},
			expectedErrorStatus: Timeout,
		},
		{
			name:                "If ErrorDetails is not set, a connection error status is calculated based on the Error field",
			err:                 syscall.ECONNREFUSED,
			errDetails:          nil,
			expectedErrorStatus: ConnectionError,
		},
		{
			name:                "If ErrorDetails is not set, a timeout error status is calculated based on the Error field",
			err:                 os.ErrDeadlineExceeded,
			errDetails:          nil,
			expectedErrorStatus: Timeout,
		},
		{
			name:                "If ErrorDetails is not set, a unauthorized error status is calculated based on the Error field",
			err:                 fs.ErrPermission,
			errDetails:          nil,
			expectedErrorStatus: Unauthorized,
		},
		{
			name:                "If ErrorDetails is not set, an unknown error status is calculated based on the Error field",
			err:                 fmt.Errorf("some custom error"),
			errDetails:          nil,
			expectedErrorStatus: Unknown,
		},
		{
			name:                "If ErrorDetails is not set, an unknown error status is calculated by unwrapping the Error field",
			err:                 fmt.Errorf("wrap 2: %w", fmt.Errorf("wrap 1: %w", os.ErrDeadlineExceeded)),
			errDetails:          nil,
			expectedErrorStatus: Timeout,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			protoRes := &QueryDataResponse{
				Responses: map[string]DataResponse{
					"A": {
						Frames:       frames,
						Error:        tc.err,
						ErrorDetails: tc.errDetails,
					},
				},
			}
			qdr, err := ToProto().QueryDataResponse(protoRes)
			require.NoError(t, err)
			require.NotNil(t, qdr)
			require.NotNil(t, qdr.Responses)
			require.Equal(t, int32(tc.expectedErrorStatus), qdr.Responses["A"].ErrorDetails.Status)
		})
	}
}
