package sdata_test

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/sdata"
	"github.com/stretchr/testify/require"
)

func TestWideFrameAddMetric_ValidCases(t *testing.T) {
	t.Run("add two metrics", func(t *testing.T) {
		wf := sdata.NewWideFrameSeries()

		err := wf.SetTime("time", []time.Time{time.UnixMilli(1), time.UnixMilli(2)})
		require.NoError(t, err)

		err = wf.AddMetric("one", data.Labels{"host": "a"}, []float64{1, 2})
		require.NoError(t, err)

		err = wf.AddMetric("one", data.Labels{"host": "b"}, []float64{3, 4})
		require.NoError(t, err)

		expectedFrame := data.NewFrame("",
			data.NewField("time", nil, []time.Time{time.UnixMilli(1), time.UnixMilli(2)}),
			data.NewField("one", data.Labels{"host": "a"}, []float64{1, 2}),
			data.NewField("one", data.Labels{"host": "b"}, []float64{3, 4}),
		).SetMeta(&data.FrameMeta{Type: data.FrameTypeTimeSeriesWide})

		if diff := cmp.Diff(expectedFrame, (*wf)[0], data.FrameTestCompareOptions()...); diff != "" {
			require.FailNow(t, "mismatch (-want +got):\n%s\n", diff)
		}
	})
}

func TestWideFrameSeriesGetMetricRefs(t *testing.T) {
	t.Run("two metrics from wide to multi", func(t *testing.T) {
		wf := sdata.NewWideFrameSeries()

		err := wf.SetTime("time", []time.Time{time.UnixMilli(1), time.UnixMilli(2)})
		require.NoError(t, err)

		err = wf.AddMetric("one", data.Labels{"host": "a"}, []float64{1, 2})
		require.NoError(t, err)

		err = wf.AddMetric("one", data.Labels{"host": "b"}, []float64{3, 4})
		require.NoError(t, err)

		refs, ignoredFields, err := wf.GetMetricRefs()
		require.NoError(t, err)

		expectedRefs := []sdata.TimeSeriesMetricRef{
			{
				ValueField: data.NewField("one", data.Labels{"host": "a"}, []float64{1, 2}),
				TimeField:  data.NewField("time", nil, []time.Time{time.UnixMilli(1), time.UnixMilli(2)}),
			},
			{
				ValueField: data.NewField("one", data.Labels{"host": "b"}, []float64{3, 4}),
				TimeField:  data.NewField("time", nil, []time.Time{time.UnixMilli(1), time.UnixMilli(2)}),
			},
		}

		require.Empty(t, ignoredFields) // TODO more specific []x{} vs nil

		if diff := cmp.Diff(expectedRefs, refs, data.FrameTestCompareOptions()...); diff != "" {
			require.FailNow(t, "mismatch (-want +got):\n%s\n", diff)
		}
	})
}
