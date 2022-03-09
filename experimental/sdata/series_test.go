package sdata_test

import (
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/sdata"
	"github.com/stretchr/testify/require"
)

func TestSeriesCollectionReaderInterface(t *testing.T) {
	timeSlice := []time.Time{time.Unix(1234567890, 0), time.Unix(1234567891, 0)}

	metricName := "os.cpu"
	valuesA := []float64{1, 2}
	valuesB := []float64{3, 4}

	// refs should be same across the formats
	expectedRefs := []sdata.TimeSeriesMetricRef{
		{
			data.NewField("time", nil, timeSlice),
			data.NewField(metricName, data.Labels{"host": "a"}, valuesA),
		},
		{
			data.NewField("time", nil, timeSlice),
			data.NewField(metricName, data.Labels{"host": "b"}, valuesB),
		},
	}

	t.Run("multi frame", func(t *testing.T) {
		sc := sdata.MultiFrameSeries{}

		err := sc.AddMetric(metricName, data.Labels{"host": "a"}, timeSlice, valuesA)
		require.NoError(t, err)

		err = sc.AddMetric(metricName, data.Labels{"host": "b"}, timeSlice, valuesB)
		require.NoError(t, err)

		var r sdata.TimeSeriesCollectionReader = &sc

		mFrameRefs, extraFields := r.GetMetricRefs()
		require.Nil(t, extraFields)
		require.Equal(t, expectedRefs, mFrameRefs)
	})

	t.Run("wide frame", func(t *testing.T) {
		sc := sdata.NewWideFrameSeries("time", timeSlice)
		err := sc.AddMetric(metricName, data.Labels{"host": "a"}, valuesA)
		require.NoError(t, err)

		err = sc.AddMetric(metricName, data.Labels{"host": "b"}, valuesB)
		require.NoError(t, err)

		var r sdata.TimeSeriesCollectionReader = &sc

		mFrameRefs, extraFields := r.GetMetricRefs()
		require.Nil(t, extraFields)
		require.Equal(t, expectedRefs, mFrameRefs)
	})

	t.Run("long frame", func(t *testing.T) {
		ls := &sdata.LongSeries{
			Frame: data.NewFrame("",
				data.NewField("time", nil, []time.Time{timeSlice[0], timeSlice[0],
					timeSlice[1], timeSlice[1]}),
				data.NewField("os.cpu", nil, []float64{valuesA[0], valuesB[0],
					valuesA[1], valuesB[1]}),
				data.NewField("host", nil, []string{"a", "b", "a", "b"}),
			).SetMeta(&data.FrameMeta{Type: data.FrameTypeTimeSeriesLong}),
		}

		var r sdata.TimeSeriesCollectionReader = ls

		mFrameRefs, extraFields := r.GetMetricRefs()

		require.Nil(t, extraFields)
		require.Equal(t, expectedRefs, mFrameRefs)
	})
}

func addFields(frame *data.Frame, fields ...*data.Field) *data.Frame {
	frame.Fields = append(frame.Fields, fields...)
	return frame
}
