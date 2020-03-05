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
	require.Contains(t, mfs, "go_goroutines")
}

func TestCheckHealth(t *testing.T) {
	t.Run("When check health handler not set should use default implementation", func(t *testing.T) {
		adapter := &sdkAdapter{}
		res, err := adapter.CheckHealth(context.Background(), &pluginv2.CheckHealth_Request{})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, pluginv2.CheckHealth_Response_OK, res.Status)
		require.Empty(t, res.Message)
		require.Empty(t, res.JsonDetails)
	})

	t.Run("When check health handler set should call that", func(t *testing.T) {
		tcs := []struct {
			status              HealthStatus
			message             string
			jsonDetails         string
			err                 error
			expectedStatus      pluginv2.CheckHealth_Response_HealthStatus
			expectedMessage     string
			expectedJSONDetails string
			expectedError       bool
		}{
			{
				status:              HealthStatusUnknown,
				message:             "unknown",
				jsonDetails:         "{}",
				expectedStatus:      pluginv2.CheckHealth_Response_UNKNOWN,
				expectedMessage:     "unknown",
				expectedJSONDetails: "{}",
			},
			{
				status:              HealthStatusOk,
				message:             "all good",
				jsonDetails:         "{}",
				expectedStatus:      pluginv2.CheckHealth_Response_OK,
				expectedMessage:     "all good",
				expectedJSONDetails: "{}",
			},
			{
				status:              HealthStatusError,
				message:             "BOOM",
				jsonDetails:         `{"error": "boom"}`,
				expectedStatus:      pluginv2.CheckHealth_Response_ERROR,
				expectedMessage:     "BOOM",
				expectedJSONDetails: `{"error": "boom"}`,
			},
			{
				err:           errors.New("BOOM"),
				expectedError: true,
			},
		}

		for _, tc := range tcs {
			adapter := &sdkAdapter{
				CheckHealthHandler: &testCheckHealthHandler{
					status:      tc.status,
					message:     tc.message,
					jsonDetails: tc.jsonDetails,
					err:         tc.err,
				},
			}

			req := &pluginv2.CheckHealth_Request{
				Config: &pluginv2.PluginConfig{},
			}
			res, err := adapter.CheckHealth(context.Background(), req)
			if tc.expectedError {
				require.Error(t, err)
				require.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.NotNil(t, res)
				require.Equal(t, tc.expectedStatus, res.Status)
				require.Equal(t, tc.expectedMessage, res.Message)
				require.Equal(t, tc.expectedJSONDetails, res.JsonDetails)
			}
		}
	})
}

type testCheckHealthHandler struct {
	status      HealthStatus
	message     string
	jsonDetails string
	err         error
}

func (h *testCheckHealthHandler) CheckHealth(ctx context.Context, req *CheckHealthRequest) (*CheckHealthResult, error) {
	return &CheckHealthResult{
		Status:      h.status,
		Message:     h.message,
		JSONDetails: h.jsonDetails,
	}, h.err
}
