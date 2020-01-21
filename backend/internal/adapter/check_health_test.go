package adapter

import (
	"context"
	"errors"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend/models"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	"github.com/stretchr/testify/require"
)

func TestCheckHealth(t *testing.T) {
	t.Run("When check health handler not set should use default implementation", func(t *testing.T) {
		adapter := &SDKAdapter{}
		res, err := adapter.CheckHealth(context.Background(), &pluginv2.CheckHealth_Request{})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, pluginv2.CheckHealth_Response_OK, res.Status)
		require.Empty(t, res.Info)
	})

	t.Run("When check health handler set should call that", func(t *testing.T) {
		tcs := []struct {
			status         models.HealthStatus
			info           string
			err            error
			expectedStatus pluginv2.CheckHealth_Response_HealthStatus
			expectedInfo   string
			expectedError  bool
		}{
			{
				status:         models.HealthStatusUnknown,
				info:           "unknown",
				expectedStatus: pluginv2.CheckHealth_Response_UNKNOWN,
				expectedInfo:   "unknown",
			},
			{
				status:         models.HealthStatusOk,
				info:           "all good",
				expectedStatus: pluginv2.CheckHealth_Response_OK,
				expectedInfo:   "all good",
			},
			{
				status:         models.HealthStatusError,
				info:           "BOOM",
				expectedStatus: pluginv2.CheckHealth_Response_ERROR,
				expectedInfo:   "BOOM",
			},
			{
				err:           errors.New("BOOM"),
				expectedError: true,
			},
		}

		for _, tc := range tcs {
			adapter := &SDKAdapter{
				CheckHealthHandler: &testCheckHealthHandler{
					status: tc.status,
					info:   tc.info,
					err:    tc.err,
				},
			}

			res, err := adapter.CheckHealth(context.Background(), &pluginv2.CheckHealth_Request{})
			if tc.expectedError {
				require.Error(t, err)
				require.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.NotNil(t, res)
				require.Equal(t, tc.expectedStatus, res.Status)
				require.Equal(t, tc.expectedInfo, res.Info)
			}
		}
	})
}

type testCheckHealthHandler struct {
	status models.HealthStatus
	info   string
	err    error
}

func (h *testCheckHealthHandler) CheckHealth(ctx context.Context) (*models.CheckHealthResult, error) {
	return &models.CheckHealthResult{
		Status: h.status,
		Info:   h.info,
	}, h.err
}
