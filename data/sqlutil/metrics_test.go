package sqlutil_test

import (
	"reflect"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/data/sqlutil"
)

const rowsScannedMetric = "plugins_sql_rows_scanned"

func rowsScannedHistogram(t *testing.T) *dto.Histogram {
	t.Helper()
	mfs, err := prometheus.DefaultGatherer.Gather()
	require.NoError(t, err)
	for _, mf := range mfs {
		if mf.GetName() != rowsScannedMetric {
			continue
		}
		metrics := mf.GetMetric()
		require.Len(t, metrics, 1, "expected one unlabelled series")
		return metrics[0].GetHistogram()
	}
	t.Fatalf("metric %s not registered", rowsScannedMetric)
	return nil
}

type rowsScannedSample struct {
	count uint64
	sum   float64
}

func rowsScannedSampleValues(t *testing.T) rowsScannedSample {
	t.Helper()
	h := rowsScannedHistogram(t)
	return rowsScannedSample{count: h.GetSampleCount(), sum: h.GetSampleSum()}
}

func TestFrameFromRows_ObservesRowsScanned(t *testing.T) {
	t.Run("counts rows scanned on successful frame build", func(t *testing.T) {
		rows := makeSingleResultSet( //nolint:rowserrcheck
			[]string{"a"},
			[]interface{}{1},
			[]interface{}{2},
			[]interface{}{3},
			[]interface{}{4},
			[]interface{}{5},
		)

		before := rowsScannedSampleValues(t)
		_, err := sqlutil.FrameFromRows(rows, 100)
		require.NoError(t, err)

		got := rowsScannedSampleValues(t)
		require.Equal(t, before.count+1, got.count, "exactly one observation")
		require.InDelta(t, 5.0, got.sum-before.sum, 0, "observed value equals rows scanned")
	})

	t.Run("observes the truncated count when rowLimit is reached", func(t *testing.T) {
		rows := makeSingleResultSet( //nolint:rowserrcheck
			[]string{"a"},
			[]interface{}{1},
			[]interface{}{2},
			[]interface{}{3},
			[]interface{}{4},
			[]interface{}{5},
		)

		before := rowsScannedSampleValues(t)
		_, err := sqlutil.FrameFromRows(rows, 2)
		require.NoError(t, err)

		got := rowsScannedSampleValues(t)
		require.Equal(t, before.count+1, got.count)
		require.InDelta(t, 2.0, got.sum-before.sum, 0)
	})

	t.Run("observes zero for an empty result set", func(t *testing.T) {
		rows := makeSingleResultSet( //nolint:rowserrcheck
			[]string{"a"},
		)

		before := rowsScannedSampleValues(t)
		_, err := sqlutil.FrameFromRows(rows, 100)
		require.NoError(t, err)

		got := rowsScannedSampleValues(t)
		require.Equal(t, before.count+1, got.count)
		require.InDelta(t, 0.0, got.sum-before.sum, 0)
	})

	t.Run("skips observation when pre-loop setup fails", func(t *testing.T) {
		rows := makeSingleResultSetWithScanTypes( //nolint:rowserrcheck
			[]string{"a"},
			[]reflect.Type{nil},
			[]interface{}{1},
		)

		before := rowsScannedSampleValues(t)
		_, err := sqlutil.FrameFromRows(rows, 100)
		require.Error(t, err)

		got := rowsScannedSampleValues(t)
		require.Equal(t, before.count, got.count, "no observation when MakeScanRow errors before the scan loop")
	})

	t.Run("counts rows scanned on the dynamic-converter path", func(t *testing.T) {
		rows := makeSingleResultSet( //nolint:rowserrcheck
			[]string{"a"},
			[]interface{}{"x"},
			[]interface{}{"y"},
			[]interface{}{"z"},
		)

		before := rowsScannedSampleValues(t)
		_, err := sqlutil.FrameFromRows(rows, 100, sqlutil.Converter{
			Name:          "dynamic",
			InputTypeName: "dynamic",
			Dynamic:       true,
		})
		require.NoError(t, err)

		got := rowsScannedSampleValues(t)
		require.Equal(t, before.count+1, got.count)
		require.InDelta(t, 3.0, got.sum-before.sum, 0)
	})
}
