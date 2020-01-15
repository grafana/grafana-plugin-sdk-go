package backend

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/require"
)

func TestCollectMetrcis(t *testing.T) {
	adapter := &sdkAdapter{}
	res, err := adapter.CollectMetrics(context.Background(), &pluginv2.CollectMetrics_Request{})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotNil(t, res.Metrics)
	require.NotNil(t, res.Metrics.Prometheus)

	reader := bytes.NewReader(res.Metrics.Prometheus)
	var parser expfmt.TextParser
	mfs, err := parser.TextToMetricFamilies(reader)
	require.NoError(t, err)
	require.Contains(t, mfs, "go_gc_duration_seconds")
	require.Contains(t, mfs, "process_virtual_memory_max_bytes")
}

func TestCheckHealth(t *testing.T) {
	t.Run("When check health handler not set should use default implementation", func(t *testing.T) {
		adapter := &sdkAdapter{}
		res, err := adapter.CheckHealth(context.Background(), &pluginv2.CheckHealth_Request{})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, pluginv2.CheckHealth_Response_OK, res.Status)
		require.Empty(t, res.Info)
	})

	t.Run("When check health handler set should call that", func(t *testing.T) {
		tcs := []struct {
			status         HealthStatus
			info           string
			err            error
			expectedStatus pluginv2.CheckHealth_Response_HealthStatus
			expectedInfo   string
			expectedError  bool
		}{
			{
				status:         HealthStatusUnknown,
				info:           "unknown",
				expectedStatus: pluginv2.CheckHealth_Response_UNKNOWN,
				expectedInfo:   "unknown",
			},
			{
				status:         HealthStatusOk,
				info:           "all good",
				expectedStatus: pluginv2.CheckHealth_Response_OK,
				expectedInfo:   "all good",
			},
			{
				status:         HealthStatusError,
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
			adapter := &sdkAdapter{
				checkHealthHandler: &testCheckHealthHandler{
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
	status HealthStatus
	info   string
	err    error
}

func (h *testCheckHealthHandler) CheckHealth(ctx context.Context) (*CheckHealthResult, error) {
	return &CheckHealthResult{
		Status: h.status,
		Info:   h.info,
	}, h.err
}
