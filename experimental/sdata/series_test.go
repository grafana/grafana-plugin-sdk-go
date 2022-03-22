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
		sc := sdata.NewMultiFrameSeries()

		err := sc.AddMetric(metricName, data.Labels{"host": "a"}, timeSlice, valuesA)
		require.NoError(t, err)

		err = sc.AddMetric(metricName, data.Labels{"host": "b"}, timeSlice, valuesB)
		require.NoError(t, err)

		var r sdata.TimeSeriesCollectionReader = sc

		mFrameRefs, extraFields, err := r.GetMetricRefs()
		require.Nil(t, err)
		require.Nil(t, extraFields)
		require.Equal(t, expectedRefs, mFrameRefs)
	})

	t.Run("wide frame", func(t *testing.T) {
		sc := sdata.NewWideFrameSeries()

		err := sc.SetTime("time", timeSlice)
		require.NoError(t, err)

		err = sc.AddMetric(metricName, data.Labels{"host": "a"}, valuesA)
		require.NoError(t, err)

		err = sc.AddMetric(metricName, data.Labels{"host": "b"}, valuesB)
		require.NoError(t, err)

		var r sdata.TimeSeriesCollectionReader = sc

		mFrameRefs, extraFields, err := r.GetMetricRefs()
		require.Nil(t, err)

		require.Nil(t, extraFields)
		require.Equal(t, expectedRefs, mFrameRefs)
	})
	t.Run("long frame", func(t *testing.T) {
		ls := &sdata.LongSeries{data.NewFrame("",
			data.NewField("time", nil, []time.Time{timeSlice[0], timeSlice[0],
				timeSlice[1], timeSlice[1]}),
			data.NewField("os.cpu", nil, []float64{valuesA[0], valuesB[0],
				valuesA[1], valuesB[1]}),
			data.NewField("host", nil, []string{"a", "b", "a", "b"}),
		).SetMeta(&data.FrameMeta{Type: data.FrameTypeTimeSeriesLong}),
		}

		var r sdata.TimeSeriesCollectionReader = ls

		mFrameRefs, extraFields, err := r.GetMetricRefs()
		require.Nil(t, err)

		require.Nil(t, extraFields)
		require.Equal(t, expectedRefs, mFrameRefs)
	})
}

func addFields(frame *data.Frame, fields ...*data.Field) *data.Frame {
	frame.Fields = append(frame.Fields, fields...)
	return frame
}

func TestEmptyFromNew(t *testing.T) {
	var multi, wide, long sdata.TimeSeriesCollectionReader

	multi = sdata.NewMultiFrameSeries()
	wide = sdata.NewWideFrameSeries()
	long = sdata.NewLongSeries()

	emptyReqs := func(refs []sdata.TimeSeriesMetricRef, ignored []sdata.FrameFieldIndex, err error) {
		require.NoError(t, err)
		require.Nil(t, ignored)
		require.NotNil(t, refs)
		require.Len(t, refs, 0)
	}

	viaFrames := func(r sdata.TimeSeriesCollectionReader) {
		t.Run("should work when losing go type via Frames()", func(t *testing.T) {
			frames := r.Frames()
			r, err := sdata.TimeSeriesReaderFromFrames(frames)
			require.NoError(t, err)

			refs, ignored, err := r.GetMetricRefs()
			emptyReqs(refs, ignored, err)
		})
	}

	t.Run("multi", func(t *testing.T) {
		refs, ignored, err := multi.GetMetricRefs()
		emptyReqs(refs, ignored, err)
		viaFrames(multi)
	})

	t.Run("wide", func(t *testing.T) {
		refs, ignored, err := wide.GetMetricRefs()
		emptyReqs(refs, ignored, err)
		viaFrames(wide)
	})

	t.Run("long", func(t *testing.T) {
		refs, ignored, err := long.GetMetricRefs()
		emptyReqs(refs, ignored, err)
		viaFrames(long)
	})
}
