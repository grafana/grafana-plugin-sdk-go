package backend_test

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/require"
)

func TestToUsageStats(t *testing.T) {
	t.Run("should ignore metrics without the usage_stats prefix", func(t *testing.T) {
		metrics := `
# TYPE go_goroutines gauge
go_goroutines 10.0
`
		res := &backend.CollectMetricsResult{PrometheusMetrics: []byte(metrics)}
		stats := res.ToUsageStats("grafana-fake-datasource")
		require.Equal(t, map[string]interface{}{}, stats)
	})

	t.Run("should parse counter", func(t *testing.T) {
		metrics := `
# HELP usage_stats_instance_total Number of instances
# TYPE usage_stats_instance_total counter
usage_stats_instance_total 10.0
`
		res := &backend.CollectMetricsResult{PrometheusMetrics: []byte(metrics)}
		stats := res.ToUsageStats("grafana-fake-datasource")
		require.Equal(t, map[string]interface{}{"stats.plugin.grafana-fake-datasource.instance_total": 10.0}, stats)
	})

	t.Run("should parse gauge", func(t *testing.T) {
		metrics := `
# HELP usage_stats_response_size_bytes Response size in bytes
# TYPE usage_stats_response_size_bytes gauge
usage_stats_response_size_bytes 100.0
`
		res := &backend.CollectMetricsResult{PrometheusMetrics: []byte(metrics)}
		stats := res.ToUsageStats("example")
		require.Equal(t, map[string]interface{}{"stats.plugin.example.response_size_bytes": float64(100)}, stats)
	})

	t.Run("should parse summary", func(t *testing.T) {
		metrics := `
# HELP usage_stats_response_size_bytes Response size in bytes
# TYPE usage_stats_response_size_bytes summary
usage_stats_response_size_bytes{quantile="0.5"} 1.0
usage_stats_response_size_bytes{quantile="0.9"} 1.0
usage_stats_response_size_bytes{quantile="0.99"} 1.0
usage_stats_response_size_bytes_sum 4
usage_stats_response_size_bytes_count 6
`
		res := &backend.CollectMetricsResult{PrometheusMetrics: []byte(metrics)}
		stats := res.ToUsageStats("example")
		require.Equal(t, map[string]interface{}{
			"stats.plugin.example.response_size_bytes.count":         6.0,
			"stats.plugin.example.response_size_bytes.quantile.0_50": 1.0,
			"stats.plugin.example.response_size_bytes.quantile.0_90": 1.0,
			"stats.plugin.example.response_size_bytes.quantile.0_99": 1.0,
			"stats.plugin.example.response_size_bytes.sum":           4.0,
		}, stats)
	})

	t.Run("should parse histogram", func(t *testing.T) {
		metrics := `
# HELP usage_stats_response_size_bytes Histogram for response size in bytes
# TYPE usage_stats_response_size_bytes histogram
usage_stats_response_size_bytes_bucket{le="8.999999999999998"} 58475
usage_stats_response_size_bytes_bucket{le="24.999999999999996"} 858715
usage_stats_response_size_bytes_bucket{le="64.99999999999999"} 1.606968e+06
usage_stats_response_size_bytes_bucket{le="144.99999999999997"} 1.963916e+06
usage_stats_response_size_bytes_bucket{le="320.99999999999994"} 2.013705e+06
usage_stats_response_size_bytes_bucket{le="704.9999999999999"} 2.026067e+06
usage_stats_response_size_bytes_bucket{le="1536.9999999999998"} 2.035428e+06
usage_stats_response_size_bytes_bucket{le="3200.9999999999995"} 2.038281e+06
usage_stats_response_size_bytes_bucket{le="6528.999999999999"} 2.041383e+06
usage_stats_response_size_bytes_bucket{le="13568.999999999998"} 2.042418e+06
usage_stats_response_size_bytes_bucket{le="27264.999999999996"} 2.043538e+06
usage_stats_response_size_bytes_bucket{le="+Inf"} 2.043966e+06
usage_stats_response_size_bytes_sum 1.88505312e+08
usage_stats_response_size_bytes_count 2.043966e+06
`
		res := &backend.CollectMetricsResult{PrometheusMetrics: []byte(metrics)}
		stats := res.ToUsageStats("example")
		require.Equal(t, map[string]interface{}{
			"stats.plugin.example.response_size_bytes.bucket.+inf":  2.043966e+06,
			"stats.plugin.example.response_size_bytes.bucket.13569": 2.042418e+06,
			"stats.plugin.example.response_size_bytes.bucket.145":   1.963916e+06,
			"stats.plugin.example.response_size_bytes.bucket.1537":  2.035428e+06,
			"stats.plugin.example.response_size_bytes.bucket.25":    858715.0,
			"stats.plugin.example.response_size_bytes.bucket.27265": 2.043538e+06,
			"stats.plugin.example.response_size_bytes.bucket.3201":  2.038281e+06,
			"stats.plugin.example.response_size_bytes.bucket.321":   2.013705e+06,
			"stats.plugin.example.response_size_bytes.bucket.65":    1.606968e+06,
			"stats.plugin.example.response_size_bytes.bucket.6529":  2.041383e+06,
			"stats.plugin.example.response_size_bytes.bucket.705":   2.026067e+06,
			"stats.plugin.example.response_size_bytes.bucket.9":     58475.0,
			"stats.plugin.example.response_size_bytes.count":        2.043966e+06,
			"stats.plugin.example.response_size_bytes.sum":          1.88505312e+08,
		}, stats)
	})

	t.Run("should parse and clean labels", func(t *testing.T) {
		metrics := `
# HELP usage_stats_queries count of queries
# TYPE usage_stats_queries counter
usage_stats_queries{feature="Analytics",Version="1.2.1"} 5.0
`
		res := &backend.CollectMetricsResult{PrometheusMetrics: []byte(metrics)}
		stats := res.ToUsageStats("example")
		require.Equal(t, map[string]interface{}{"stats.plugin.example.queries.feature.analytics.version.1_2_1": 5.0}, stats)
	})
}
