package adapter

import (
	"bytes"
	"context"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	"github.com/prometheus/common/expfmt"
	"github.com/stretchr/testify/require"
)

func TestCollectMetrcis(t *testing.T) {
	adapter := &SDKAdapter{}
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
