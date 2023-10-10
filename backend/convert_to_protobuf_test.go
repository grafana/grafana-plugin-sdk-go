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
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
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
			qdr, err := ToProto().QueryDataResponse(protoRes, "")
			require.NoError(t, err)
			require.NotNil(t, qdr)
			require.NotNil(t, qdr.Responses)
			receivedStatus := qdr.Responses["A"].Status
			require.Equal(t, tc.expectedStatus, receivedStatus)
		})
	}

	t.Run("No accept specified and no proxy initialized should return Arrow protobuf response", func(t *testing.T) {
		dr := testDataResponse(t)
		qdr := NewQueryDataResponse()
		qdr.Responses["A"] = dr
		require.Nil(t, qdr.ResponseProxy())

		protoRes, err := ToProto().QueryDataResponse(qdr, "")
		require.NoError(t, err)
		require.NotNil(t, protoRes)
		require.Equal(t, pluginv2.QueryDataResponse_ARROW, protoRes.DataType)
	})

	t.Run("Arrow accept specified and no proxy initialized should return Arrow protobuf response", func(t *testing.T) {
		dr := testDataResponse(t)
		qdr := NewQueryDataResponse()
		qdr.Responses["A"] = dr
		require.Nil(t, qdr.ResponseProxy())

		protoRes, err := ToProto().QueryDataResponse(qdr, string(DataResponseTypeArrow))
		require.NoError(t, err)
		require.NotNil(t, protoRes)
		require.Equal(t, pluginv2.QueryDataResponse_ARROW, protoRes.DataType)
	})

	t.Run("JSON accept specified and no proxy initialized should return JSON protobuf response", func(t *testing.T) {
		dr := testDataResponse(t)
		qdr := NewQueryDataResponse()
		qdr.Responses["A"] = dr
		require.Nil(t, qdr.ResponseProxy())

		protoRes, err := ToProto().QueryDataResponse(qdr, string(DataResponseTypeJSON))
		require.NoError(t, err)
		require.NotNil(t, protoRes)
		require.Equal(t, pluginv2.QueryDataResponse_JSON, protoRes.DataType)
		str := string(protoRes.Data)
		require.Equal(t, `{"results":{"A":{"status":200,"frames":[{"schema":{"name":"simple","fields":[{"name":"time","type":"time","typeInfo":{"frame":"time.Time"}},{"name":"valid","type":"boolean","typeInfo":{"frame":"bool"}}]},"data":{"values":[[1577934240000,1577934300000],[true,false]]}},{"schema":{"name":"other","fields":[{"name":"value","type":"number","typeInfo":{"frame":"float64"}}]},"data":{"values":[[1]]}}]}}}`, str)
	})
}
