package sqlutil

import (
	"context"
	"errors"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "grafana"
	subsystem = "datasources"
)

// The allowed label keys used across all metrics
var metricLabelKeys = []string{"query_type", "datasource_type"}

var rowsProcessed = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "sqlutil_rows_processed_total",
		Help:      "Total rows processed by FrameFromRows",
	},
	metricLabelKeys,
)

var rowCountHistogram = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "sqlutil_rows_per_query",
		Help:      "Histogram of row counts returned by FrameFromRows",
		Buckets:   prometheus.ExponentialBuckets(1, 2, 10),
	},
	metricLabelKeys,
)

var cellsProcessed = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "sqlutil_cells_processed_total",
		Help:      "Total number of individual SQL cells processed",
	},
	metricLabelKeys,
)

var cellCountHistogram = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "sqlutil_cells_per_query",
		Help:      "Histogram of the number of SQL cells processed per query",
		Buckets:   prometheus.ExponentialBuckets(1, 2, 12),
	},
	metricLabelKeys,
)

// RegisterMetrics registers Prometheus metrics for sqlutil.
// It safely handles duplicate registration and returns any non-duplicate errors.
func RegisterMetrics(reg prometheus.Registerer) error {
	return registerAll(reg,
		rowsProcessed,
		rowCountHistogram,
		cellsProcessed,
		cellCountHistogram,
	)
}

func registerAll(reg prometheus.Registerer, collectors ...prometheus.Collector) error {
	for _, c := range collectors {
		if err := reg.Register(c); err != nil {
			var are prometheus.AlreadyRegisteredError
			if errors.As(err, &are) {
				// Copy underlying collector pointer to avoid nil metric errors
				switch v := c.(type) {
				case *prometheus.CounterVec:
					if existing, ok := are.ExistingCollector.(*prometheus.CounterVec); ok {
						*v = *existing
					}
				case *prometheus.HistogramVec:
					if existing, ok := are.ExistingCollector.(*prometheus.HistogramVec); ok {
						*v = *existing
					}
				}
				continue // skip AlreadyRegisteredError
			}
			return err
		}
	}
	return nil
}

// Context key for metric labels
type ctxKeyMetricLabels struct{}

// ContextWithMetricLabels returns a context with the given Prometheus labels attached.
// Callers should provide keys matching metricLabelKeys: "query_type" and "datasource_type".
func ContextWithMetricLabels(ctx context.Context, labels map[string]string) context.Context {
	return context.WithValue(ctx, ctxKeyMetricLabels{}, labels)
}

// getMetricLabels extracts only the allowed metric labels from context.
// Missing keys are filled with empty strings.
func getMetricLabels(ctx context.Context) prometheus.Labels {
	out := prometheus.Labels{}
	for _, key := range metricLabelKeys {
		out[key] = ""
	}

	if v := ctx.Value(ctxKeyMetricLabels{}); v != nil {
		if m, ok := v.(map[string]string); ok {
			for _, key := range metricLabelKeys {
				if val, exists := m[key]; exists {
					out[key] = val
				}
			}
		}
	}

	return out
}
