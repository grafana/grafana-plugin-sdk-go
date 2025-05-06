package sqlutil_test

import (
	"context"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data/sqlutil"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestFrameFromRowsWithContext_MetricsRecorded(t *testing.T) {
	reg := prometheus.NewRegistry()
	err := sqlutil.RegisterMetrics(reg)
	require.NoError(t, err)

	ctx := sqlutil.ContextWithMetricLabels(context.Background(), map[string]string{
		"query_type":      "test",
		"datasource_type": "fake",
	})

	// 2 rows × 3 columns
	rows := makeSingleResultSet(
		[]string{"a", "b", "c"},
		[]interface{}{1, 2, 3},
		[]interface{}{4, 5, 6},
	)
	require.NoError(t, rows.Err())

	_, err = sqlutil.FrameFromRowsWithContext(ctx, rows, 100)
	require.NoError(t, err)

	// Gather and inspect metrics
	metrics, err := reg.Gather()
	require.NoError(t, err)

	labels := map[string]string{
		"query_type":      "test",
		"datasource_type": "fake",
	}

	assertCounter := func(name string, want float64) {
		m := findMetricWithLabels(metrics, name, labels)
		require.NotNil(t, m, "metric %s not found", name)
		require.Equal(t, want, m.GetCounter().GetValue(), "metric %s value mismatch", name)
	}

	assertHistogram := func(name string, minSamples uint64) {
		m := findMetricWithLabels(metrics, name, labels)
		require.NotNil(t, m, "metric %s not found", name)
		require.GreaterOrEqual(t, m.GetHistogram().GetSampleCount(), minSamples, "metric %s histogram count too low", name)
	}

	assertCounter("grafana_datasources_sqlutil_rows_processed_total", 2)
	assertCounter("grafana_datasources_sqlutil_cells_processed_total", 6)
	assertHistogram("grafana_datasources_sqlutil_rows_per_query", 1)
	assertHistogram("grafana_datasources_sqlutil_cells_per_query", 1)
}

// findMetricWithLabels finds a metric by name and label set
func findMetricWithLabels(metrics []*dto.MetricFamily, name string, expectedLabels map[string]string) *dto.Metric {
	for _, mf := range metrics {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.Metric {
			matches := true
			for _, label := range m.GetLabel() {
				if val, ok := expectedLabels[label.GetName()]; ok {
					if val != label.GetValue() {
						matches = false
						break
					}
				}
			}
			if matches {
				return m
			}
		}
	}
	return nil
}
