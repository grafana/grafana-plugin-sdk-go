package backend

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/require"
)

func TestConvertToProtobufQueryDataResponse(t *testing.T) {
	frames := data.Frames{data.NewFrame("test", data.NewField("test", nil, []int64{1}))}
	tcs := []struct {
		name        string
		err         error
		status      Status
		errorSource ErrorSource

		expectedStatus      int32
		expectedErrorSource string
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
		{
			name:                "ErrorSource is marshalled",
			err:                 errors.New("oh no"),
			status:              StatusBadGateway,
			errorSource:         ErrorSourceDownstream,
			expectedStatus:      int32(StatusBadGateway),
			expectedErrorSource: "downstream",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			protoRes := &QueryDataResponse{
				Responses: map[string]DataResponse{
					"A": {
						Frames:      frames,
						Error:       tc.err,
						Status:      tc.status,
						ErrorSource: tc.errorSource,
					},
				},
			}
			qdr, err := ToProto().QueryDataResponse(protoRes)
			require.NoError(t, err)
			require.NotNil(t, qdr)
			require.NotNil(t, qdr.Responses)
			resp := qdr.Responses["A"]
			require.Equal(t, tc.expectedStatus, resp.Status)
			require.Equal(t, tc.expectedErrorSource, resp.ErrorSource)
		})
	}
}

func TestConvertToProtobufStatus(t *testing.T) {
	ar := ToProto().StatusResult(&StatusResult{
		Status:  "a",
		Message: "b",
		Reason:  "c",
		Code:    234,
	})
	require.NotNil(t, ar)
	require.Equal(t, "a", ar.Status)
	require.Equal(t, "b", ar.Message)
	require.Equal(t, "c", ar.Reason)
	require.Equal(t, int32(234), ar.Code)
}

func TestInstanceSettingsAdmissionConversions(t *testing.T) {
	t.Run("DataSource", func(t *testing.T) {
		before := &DataSourceInstanceSettings{
			URL:      "http://something",
			Updated:  time.Now(),
			User:     "u",
			JSONData: []byte(`{"hello": "world"}`),
			DecryptedSecureJSONData: map[string]string{
				"A": "B",
			},
		}
		wire, err := DataSourceInstanceSettingsToProtoBytes(before)
		require.NoError(t, err)
		after, err := DataSourceInstanceSettingsFromProto(wire, "")
		require.NoError(t, err)
		require.Equal(t, before.URL, after.URL)
		require.Equal(t, before.User, after.User)
		require.Equal(t, before.JSONData, after.JSONData)
		require.Equal(t, before.DecryptedSecureJSONData, after.DecryptedSecureJSONData)
	})

	t.Run("App", func(t *testing.T) {
		before := &AppInstanceSettings{
			Updated:  time.Now(),
			JSONData: []byte(`{"hello": "world"}`),
			DecryptedSecureJSONData: map[string]string{
				"A": "B",
			},
		}
		wire, err := AppInstanceSettingsToProtoBytes(before)
		require.NoError(t, err)
		after, err := AppInstanceSettingsFromProto(wire)
		require.NoError(t, err)
		require.Equal(t, before.JSONData, after.JSONData)
		require.Equal(t, before.DecryptedSecureJSONData, after.DecryptedSecureJSONData)
	})
}
