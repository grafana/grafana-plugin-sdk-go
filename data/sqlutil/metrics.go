package sqlutil

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// rowsScannedHistogram observes the number of database rows scanned by
// FrameFromRows before any frame reshaping (e.g. LongToWide, LongToMulti).
// This is the driver-side row count, distinct from frame-level row metrics
// emitted downstream in sqlds.
var rowsScannedHistogram = promauto.NewHistogram(
	prometheus.HistogramOpts{
		Namespace: "plugins",
		Name:      "sql_rows_scanned",
		Help:      "Histogram of database rows scanned by FrameFromRows before any frame reshaping.",
		Buckets: []float64{
			1, 10, 100,
			1_000, 10_000, 100_000,
			1_000_000, 10_000_000, 100_000_000,
		},
		NativeHistogramBucketFactor:     1.1,
		NativeHistogramMaxBucketNumber:  100,
		NativeHistogramMinResetDuration: time.Hour,
	},
)

func observeRowsScanned(n int64) {
	rowsScannedHistogram.Observe(float64(n))
}
