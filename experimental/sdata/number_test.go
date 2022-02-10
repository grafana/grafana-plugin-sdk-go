package sdata_test

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/sdata"
	"github.com/stretchr/testify/require"
)

func TestSimpleNumeric(t *testing.T) {
	// addMetrics uses the writer interface to add sample metrics
	addMetrics := func(c sdata.NumericCollectionWriter) {
		err := c.AddMetric("os.cpu", data.Labels{"host": "a"}, 1.0)
		require.NoError(t, err)
		err = c.AddMetric("os.cpu", data.Labels{"host": "b"}, 2.0)
		require.NoError(t, err)

	}

	// refs should be same across the formats
	expectedRefs := []sdata.NumericMetricRef{
		{
			data.NewField("os.cpu", data.Labels{"host": "a"}, []float64{1}),
		},
		{
			data.NewField("os.cpu", data.Labels{"host": "b"}, []float64{2}),
		},
	}

	// multiframe
	var mFrameNC sdata.NumericCollection = &sdata.MultiFrameNumeric{}
	addMetrics(mFrameNC)

	mFrameRefs := mFrameNC.GetMetricRefs()
	require.Equal(t, expectedRefs, mFrameRefs)

	// wideframe
	var wFrameNC sdata.NumericCollection = &sdata.WideFrameNumeric{}
	addMetrics(wFrameNC)

	wFrameRefs := wFrameNC.GetMetricRefs()
	require.Equal(t, expectedRefs, wFrameRefs)

	// longframe
	lfn := &sdata.LongFrameNumeric{
		Frame: data.NewFrame("",
			data.NewField("os.cpu", nil, []float64{1, 2}),
			data.NewField("host", nil, []string{"a", "b"}),
		),
	}
	var lFrameNCR sdata.NumericCollectionReader = lfn

	lFrameRefs := lFrameNCR.GetMetricRefs()
	require.Equal(t, expectedRefs, lFrameRefs)
}
