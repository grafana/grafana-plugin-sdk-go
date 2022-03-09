package sdata_test

import (
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/sdata"
	"github.com/stretchr/testify/require"
)

func TestPlayGround(t *testing.T) {
	var mfs sdata.MultiFrameSeries
	mfs.AddMetric("os.cpu", data.Labels{"host": "a"}, []time.Time{time.Unix(1234567890, 0)}, []float64{3})
}

func emptyFrameWithTypeMD(t data.FrameType) *data.Frame {
	return data.NewFrame("").SetMeta(&data.FrameMeta{Type: t})
}

func addFields(frame *data.Frame, fields ...*data.Field) *data.Frame {
	frame.Fields = append(frame.Fields, fields...)
	return frame
}

func TestReaderWriterInterface_SharedTime(t *testing.T) {
	timeSlice := []time.Time{time.Unix(1234567890, 0), time.Unix(1234567891, 0)}

	addMetrics := func(c sdata.TimeSeriesCollectionWriter, rErr require.ErrorAssertionFunc) {
		err := c.AddMetric("os.cpu", data.Labels{"host": "a"}, timeSlice, []float64{1, 2})
		require.NoError(t, err)
		err = c.AddMetric("os.cpu", data.Labels{"host": "b"}, timeSlice, []float64{3, 4})
		rErr(t, err)
	}

	// refs should be same across the formats
	expectedRefs := []sdata.TimeSeriesMetricRef{
		{
			data.NewField("time", nil, timeSlice),
			data.NewField("os.cpu", data.Labels{"host": "a"}, []float64{1, 2}),
		},
		{
			data.NewField("time", nil, timeSlice),
			data.NewField("os.cpu", data.Labels{"host": "b"}, []float64{3, 4}),
		},
	}

	t.Run("multi frame", func(t *testing.T) {
		var mFrameTSC sdata.TimeSeriesCollection = &sdata.MultiFrameSeries{}
		addMetrics(mFrameTSC, require.NoError)

		mFrameRefs, extraFields := mFrameTSC.GetMetricRefs()
		require.Nil(t, extraFields)
		require.Equal(t, expectedRefs, mFrameRefs)
	})

	t.Run("wide frame", func(t *testing.T) {
		var mFrameTSC sdata.TimeSeriesCollection = &sdata.WideFrameSeries{}
		addMetrics(mFrameTSC, require.Error)
	})
}
