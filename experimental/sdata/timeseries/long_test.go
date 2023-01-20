package timeseries_test

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/sdata/timeseries"
	"github.com/stretchr/testify/require"
)

func TestLongSeriesGetCollection(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		ls := timeseries.LongFrame{
			data.NewFrame("",
				data.NewField("time", nil, []time.Time{time.UnixMilli(1), time.UnixMilli(1)}),
				data.NewField("host", nil, []string{"a", "b"}),
				data.NewField("iface", nil, []string{"eth0", "eth0"}),
				data.NewField("in_bytes", nil, []float64{1, 2}),
				data.NewField("out_bytes", nil, []int64{3, 4}),
			).SetMeta(&data.FrameMeta{Type: data.FrameTypeTimeSeriesLong, TypeVersion: data.FrameTypeVersion{0, 1}}),
		}

		c, err := ls.GetCollection(false)
		require.NoError(t, err)

		expectedRefs := []timeseries.MetricRef{
			{
				ValueField: data.NewField("in_bytes", data.Labels{"host": "a", "iface": "eth0"}, []float64{1}),
				TimeField:  data.NewField("time", nil, []time.Time{time.UnixMilli(1)}),
			},
			{
				ValueField: data.NewField("in_bytes", data.Labels{"host": "b", "iface": "eth0"}, []float64{2}),
				TimeField:  data.NewField("time", nil, []time.Time{time.UnixMilli(1)}),
			},
			{
				ValueField: data.NewField("out_bytes", data.Labels{"host": "a", "iface": "eth0"}, []int64{3}),
				TimeField:  data.NewField("time", nil, []time.Time{time.UnixMilli(1)}),
			},
			{
				ValueField: data.NewField("out_bytes", data.Labels{"host": "b", "iface": "eth0"}, []int64{4}),
				TimeField:  data.NewField("time", nil, []time.Time{time.UnixMilli(1)}),
			},
		}

		require.Empty(t, c.RemainderIndices) // TODO more specific []x{} vs nil

		require.NoError(t, c.Warning)

		if diff := cmp.Diff(expectedRefs, c.Refs, data.FrameTestCompareOptions()...); diff != "" {
			require.FailNow(t, "mismatch (-want +got):\n", diff)
		}
	})
}
