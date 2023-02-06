package numeric_test

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/sdata/numeric"
	"github.com/stretchr/testify/require"
)

func TestSimpleNumeric(t *testing.T) {
	// addMetrics uses the writer interface to add sample metrics
	addMetrics := func(c numeric.CollectionWriter) {
		err := c.AddMetric("os.cpu", data.Labels{"host": "a"}, 1.0)
		require.NoError(t, err)
		err = c.AddMetric("os.cpu", data.Labels{"host": "b"}, 2.0)
		require.NoError(t, err)
	}

	// refs should be same across the formats
	expectedRefs := []numeric.MetricRef{
		{
			ValueField: data.NewField("os.cpu", data.Labels{"host": "a"}, []float64{1}),
		},
		{
			ValueField: data.NewField("os.cpu", data.Labels{"host": "b"}, []float64{2}),
		},
	}

	t.Run("multi frame", func(t *testing.T) {
		var mFrameNC numeric.CollectionRW
		var err error
		mFrameNC, err = numeric.NewMultiFrame(numeric.MultiFrameVersionLatest)
		require.NoError(t, err)

		addMetrics(mFrameNC)

		mc, err := mFrameNC.GetCollection(false)
		require.NoError(t, mc.Warning)
		require.Nil(t, err)
		require.Nil(t, mc.RemainderIndices)
		require.Equal(t, expectedRefs, mc.Refs)
	})

	t.Run("wide frame", func(t *testing.T) {
		var wFrameNC numeric.CollectionRW
		var err error
		wFrameNC, err = numeric.NewWideFrame(numeric.WideFrameVersionLatest)
		require.NoError(t, err)

		addMetrics(wFrameNC)

		wc, err := wFrameNC.GetCollection(false)
		require.NoError(t, wc.Warning)
		require.Nil(t, err)
		require.Nil(t, wc.RemainderIndices)
		require.Equal(t, expectedRefs, wc.Refs)
	})
	t.Run("long frame", func(t *testing.T) {
		lfn := &numeric.LongFrame{
			Frame: data.NewFrame("",
				data.NewField("os.cpu", nil, []float64{1, 2}),
				data.NewField("host", nil, []string{"a", "b"}),
			).SetMeta(&data.FrameMeta{Type: data.FrameTypeNumericLong,
				TypeVersion: numeric.LongFrameVersionLatest}),
		}
		var lcr numeric.CollectionReader = lfn

		lc, err := lcr.GetCollection(false)
		require.NoError(t, lc.Warning)
		require.Nil(t, err)
		require.Nil(t, lc.RemainderIndices)
		require.Equal(t, expectedRefs, lc.Refs)
	})
}
