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
		var mFrameNC numeric.Collection = numeric.NewMultiFrame()
		addMetrics(mFrameNC)

		mFrameRefs, ignored, err := mFrameNC.GetMetricRefs()
		require.Nil(t, err)
		require.Nil(t, ignored)
		require.Equal(t, expectedRefs, mFrameRefs)
	})

	t.Run("wide frame", func(t *testing.T) {
		var wFrameNC numeric.Collection = numeric.NewWideFrame()
		addMetrics(wFrameNC)

		wFrameRefs, ignored, err := wFrameNC.GetMetricRefs()
		require.Nil(t, err)
		require.Nil(t, ignored)
		require.Equal(t, expectedRefs, wFrameRefs)
	})
	t.Run("long frame", func(t *testing.T) {
		lfn := &numeric.LongFrame{
			Frame: data.NewFrame("",
				data.NewField("os.cpu", nil, []float64{1, 2}),
				data.NewField("host", nil, []string{"a", "b"}),
			),
		}
		var lFrameNCR numeric.CollectionReader = lfn

		lFrameRefs, ignored, err := lFrameNCR.GetMetricRefs()
		require.Nil(t, err)
		require.Nil(t, ignored)
		require.Equal(t, expectedRefs, lFrameRefs)
	})
}
