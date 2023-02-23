package timeseries_test

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/sdata/timeseries"
	"github.com/stretchr/testify/require"
)

func TestWideFrameAddMetric_ValidCases(t *testing.T) {
	t.Run("add two metrics", func(t *testing.T) {
		wf, err := timeseries.NewWideFrame("A", timeseries.WideFrameVersionLatest)
		require.NoError(t, err)

		err = wf.SetTime("time", []time.Time{time.UnixMilli(1), time.UnixMilli(2)})
		require.NoError(t, err)

		err = wf.AddSeries("one", data.Labels{"host": "a"}, []float64{1, 2})
		require.NoError(t, err)

		err = wf.AddSeries("one", data.Labels{"host": "b"}, []float64{3, 4})
		require.NoError(t, err)

		expectedFrame := data.NewFrame("",
			data.NewField("time", nil, []time.Time{time.UnixMilli(1), time.UnixMilli(2)}),
			data.NewField("one", data.Labels{"host": "a"}, []float64{1, 2}),
			data.NewField("one", data.Labels{"host": "b"}, []float64{3, 4}),
		).SetMeta(&data.FrameMeta{Type: data.FrameTypeTimeSeriesWide, TypeVersion: data.FrameTypeVersion{0, 1}})

		expectedFrame.RefID = "A"

		if diff := cmp.Diff(expectedFrame, (*wf)[0], data.FrameTestCompareOptions()...); diff != "" {
			require.FailNow(t, "mismatch (-want +got):\n%s\n", diff)
		}
	})
}

func TestWideFrameSeriesGetMetricRefs(t *testing.T) {
	t.Run("two metrics from wide to multi", func(t *testing.T) {
		wf, err := timeseries.NewWideFrame("A", timeseries.WideFrameVersionLatest)
		require.NoError(t, err)

		err = wf.SetTime("time", []time.Time{time.UnixMilli(1), time.UnixMilli(2)})
		require.NoError(t, err)

		err = wf.AddSeries("one", data.Labels{"host": "a"}, []float64{1, 2})
		require.NoError(t, err)

		err = wf.AddSeries("one", data.Labels{"host": "b"}, []float64{3, 4})
		require.NoError(t, err)

		c, err := wf.GetCollection(false)
		require.NoError(t, err)

		expectedRefs := []timeseries.MetricRef{
			{
				ValueField: data.NewField("one", data.Labels{"host": "a"}, []float64{1, 2}),
				TimeField:  data.NewField("time", nil, []time.Time{time.UnixMilli(1), time.UnixMilli(2)}),
			},
			{
				ValueField: data.NewField("one", data.Labels{"host": "b"}, []float64{3, 4}),
				TimeField:  data.NewField("time", nil, []time.Time{time.UnixMilli(1), time.UnixMilli(2)}),
			},
		}

		require.Empty(t, c.RemainderIndices) // TODO more specific []x{} vs nil
		require.NoError(t, c.Warning)

		if diff := cmp.Diff(expectedRefs, c.Refs, data.FrameTestCompareOptions()...); diff != "" {
			require.FailNow(t, "mismatch (-want +got):\n%s\n", diff)
		}
	})
}
