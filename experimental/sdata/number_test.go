package sdata_test

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/sdata"
	"github.com/stretchr/testify/require"
)

func TestSimpleNumeric(t *testing.T) {
	addMetrics := func(c sdata.NumericCollection) {
		err := c.AddMetric("os.cpu", data.Labels{"host": "a"}, 1.0)
		require.NoError(t, err)
		err = c.AddMetric("os.cpu", data.Labels{"host": "b"}, 2.0)
		require.NoError(t, err)

	}

	expectedRefs := []sdata.NumericMetricRef{
		{
			data.NewField("os.cpu", data.Labels{"host": "a"}, []float64{1}),
		},
		{
			data.NewField("os.cpu", data.Labels{"host": "b"}, []float64{2}),
		},
	}

	var mFrameNC sdata.NumericCollection = &sdata.MultiFrameNumeric{}
	addMetrics(mFrameNC)

	mFrameRefs := mFrameNC.GetMetricRefs()
	require.Equal(t, expectedRefs, mFrameRefs)

	var wFrameNC sdata.NumericCollection = &sdata.WideFrameNumeric{}
	addMetrics(wFrameNC)

	wFrameRefs := wFrameNC.GetMetricRefs()
	require.Equal(t, expectedRefs, wFrameRefs)

	lfn := &sdata.LongFrameNumeric{
		Frame: data.NewFrame("",
			data.NewField("os.cpu", nil, []float64{1, 2}),
			data.NewField("host", nil, []string{"a", "b"}),
		),
	}
	var lFrameNCR sdata.NumericCollectionReader = lfn

	lFrameRefs := lFrameNCR.GetMetricRefs()
	require.Equal(t, expectedRefs, lFrameRefs, "both are dynamic output expected to be equal")

}
