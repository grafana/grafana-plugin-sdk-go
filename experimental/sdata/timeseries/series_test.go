package timeseries_test

import (
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/sdata/timeseries"
	"github.com/stretchr/testify/require"
)

func TestSeriesCollectionReaderInterface(t *testing.T) {
	timeSlice := []time.Time{time.Unix(1234567890, 0), time.Unix(1234567891, 0)}

	metricName := "os.cpu"
	valuesA := []float64{1, 2}
	valuesB := []float64{3, 4}

	// refs should be same across the formats
	expectedRefs := []timeseries.MetricRef{
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
		sc, err := timeseries.NewMultiFrame("A", timeseries.WideFrameVersionLatest)
		require.NoError(t, err)

		err = sc.AddSeries(metricName, data.Labels{"host": "a"}, timeSlice, valuesA)
		require.NoError(t, err)

		err = sc.AddSeries(metricName, data.Labels{"host": "b"}, timeSlice, valuesB)
		require.NoError(t, err)

		var r timeseries.CollectionReader = sc

		c, err := r.GetCollection(true)

		require.NoError(t, c.Warning)
		require.Nil(t, err)
		require.Nil(t, c.RemainderIndices)
		require.Equal(t, expectedRefs, c.Refs)
	})

	t.Run("wide frame", func(t *testing.T) {
		sc, err := timeseries.NewWideFrame("B", timeseries.WideFrameVersionLatest)
		require.NoError(t, err)

		err = sc.SetTime("time", timeSlice)
		require.NoError(t, err)

		err = sc.AddSeries(metricName, data.Labels{"host": "a"}, valuesA)
		require.NoError(t, err)

		err = sc.AddSeries(metricName, data.Labels{"host": "b"}, valuesB)
		require.NoError(t, err)

		var r timeseries.CollectionReader = sc

		c, err := r.GetCollection(true)
		require.Nil(t, err)

		require.NoError(t, c.Warning)
		require.Nil(t, c.RemainderIndices)
		require.Equal(t, expectedRefs, c.Refs)
	})

	t.Run("long frame", func(t *testing.T) {
		ls := &timeseries.LongFrame{data.NewFrame("",
			data.NewField("time", nil, []time.Time{timeSlice[0], timeSlice[0],
				timeSlice[1], timeSlice[1]}),
			data.NewField("os.cpu", nil, []float64{valuesA[0], valuesB[0],
				valuesA[1], valuesB[1]}),
			data.NewField("host", nil, []string{"a", "b", "a", "b"}),
		).SetMeta(&data.FrameMeta{Type: data.FrameTypeTimeSeriesLong, TypeVersion: data.FrameTypeVersion{0, 1}}),
		}

		var r timeseries.CollectionReader = ls

		c, err := r.GetCollection(true)
		require.Nil(t, err)

		require.NoError(t, c.Warning)
		require.Nil(t, c.RemainderIndices)
		require.Equal(t, expectedRefs, c.Refs)
	})
}

func addFields(frame *data.Frame, fields ...*data.Field) *data.Frame {
	frame.Fields = append(frame.Fields, fields...)
	return frame
}

func TestEmptyFromNew(t *testing.T) {
	var multi, wide, long timeseries.CollectionReader
	var err error

	multi, err = timeseries.NewMultiFrame("A", timeseries.MultiFrameVersionLatest)
	require.NoError(t, err)

	wide, err = timeseries.NewWideFrame("B", timeseries.WideFrameVersionLatest)
	require.NoError(t, err)

	long, err = timeseries.NewLongFrame("C", timeseries.LongFrameVersionLatest)
	require.NoError(t, err)

	emptyReqs := func(c timeseries.Collection, err error) {
		require.NoError(t, err)
		require.Nil(t, c.RemainderIndices)
		require.NotNil(t, c.Refs)
		require.Len(t, c.Refs, 0)
		require.NoError(t, c.Warning)
	}

	viaFrames := func(r timeseries.CollectionReader) {
		t.Run("should work when losing go type via Frames()", func(t *testing.T) {
			frames := r.Frames()
			r, err := timeseries.CollectionReaderFromFrames(frames)
			require.NoError(t, err)

			c, err := r.GetCollection(true)
			emptyReqs(c, err)
		})
	}

	t.Run("multi", func(t *testing.T) {
		c, err := multi.GetCollection(true)
		emptyReqs(c, err)
		viaFrames(multi)
	})

	t.Run("wide", func(t *testing.T) {
		c, err := wide.GetCollection(true)
		emptyReqs(c, err)
		viaFrames(wide)
	})

	t.Run("long", func(t *testing.T) {
		c, err := long.GetCollection(true)
		emptyReqs(c, err)
		viaFrames(long)
	})
}
