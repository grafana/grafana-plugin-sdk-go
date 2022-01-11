package backend

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/require"
)

func TestCollectUsageStats(t *testing.T) {
	t.Run("Usage stats handler not set", func(t *testing.T) {
		adapter := &diagnosticsSDKAdapter{}
		res, err := adapter.CollectUsageStats(context.Background(), &pluginv2.CollectUsageStatsRequest{})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, int64(0), res.Stats["stats.ds.example.feature.count"])
	})

	t.Run("Usage stats handler set", func(t *testing.T) {
		adapter := &diagnosticsSDKAdapter{
			usageStatsHandler: &testCollectUsageStats{stats: map[string]int64{"stats.ds.example.feature.count": 1}},
		}
		res, err := adapter.CollectUsageStats(context.Background(), &pluginv2.CollectUsageStatsRequest{PluginContext: &pluginv2.PluginContext{}})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, int64(1), res.Stats["stats.ds.example.feature.count"])
	})
}

func TestCollectMetrcis(t *testing.T) {
	adapter := &diagnosticsSDKAdapter{
		metricGatherer: prometheus.DefaultGatherer,
	}
	res, err := adapter.CollectMetrics(context.Background(), &pluginv2.CollectMetricsRequest{})
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
		adapter := &diagnosticsSDKAdapter{}
		res, err := adapter.CheckHealth(context.Background(), &pluginv2.CheckHealthRequest{})
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, pluginv2.CheckHealthResponse_OK, res.Status)
		require.Empty(t, res.Message)
		require.Empty(t, res.JsonDetails)
	})

	t.Run("When check health handler set should call that", func(t *testing.T) {
		tcs := []struct {
			status              HealthStatus
			message             string
			jsonDetails         []byte
			err                 error
			expectedStatus      pluginv2.CheckHealthResponse_HealthStatus
			expectedMessage     string
			expectedJSONDetails []byte
			expectedError       bool
		}{
			{
				status:              HealthStatusUnknown,
				message:             "unknown",
				jsonDetails:         []byte("{}"),
				expectedStatus:      pluginv2.CheckHealthResponse_UNKNOWN,
				expectedMessage:     "unknown",
				expectedJSONDetails: []byte("{}"),
			},
			{
				status:              HealthStatusOk,
				message:             "all good",
				jsonDetails:         []byte("{}"),
				expectedStatus:      pluginv2.CheckHealthResponse_OK,
				expectedMessage:     "all good",
				expectedJSONDetails: []byte("{}"),
			},
			{
				status:              HealthStatusError,
				message:             "BOOM",
				jsonDetails:         []byte(`{"error": "boom"}`),
				expectedStatus:      pluginv2.CheckHealthResponse_ERROR,
				expectedMessage:     "BOOM",
				expectedJSONDetails: []byte(`{"error": "boom"}`),
			},
			{
				err:           errors.New("BOOM"),
				expectedError: true,
			},
		}

		for _, tc := range tcs {
			adapter := newDiagnosticsSDKAdapter(nil, &testCheckHealthHandler{
				status:      tc.status,
				message:     tc.message,
				jsonDetails: tc.jsonDetails,
				err:         tc.err,
			}, nil)

			req := &pluginv2.CheckHealthRequest{
				PluginContext: &pluginv2.PluginContext{},
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
	jsonDetails []byte
	err         error
}

func (h *testCheckHealthHandler) CheckHealth(_ context.Context, _ *CheckHealthRequest) (*CheckHealthResult, error) {
	return &CheckHealthResult{
		Status:      h.status,
		Message:     h.message,
		JSONDetails: h.jsonDetails,
	}, h.err
}

type testCollectUsageStats struct {
	stats map[string]int64
	err   error
}

func (h *testCollectUsageStats) CollectUsageStats(_ context.Context, _ *CollectUsageStatsRequest) (*CollectUsageStatsResponse, error) {
	return &CollectUsageStatsResponse{
		Stats: h.stats,
	}, h.err
}
