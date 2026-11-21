package data_test

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func benchTimeSeriesFrame(rows int) *data.Frame {
	ts := make([]time.Time, rows)
	vals := make([]float64, rows)
	start := time.UnixMilli(1_700_000_000_000)
	for i := range rows {
		ts[i] = start.Add(time.Duration(i) * 15 * time.Second)
		vals[i] = float64(i) * 0.37
	}
	return data.NewFrame("series",
		data.NewField("time", nil, ts),
		data.NewField("value", data.Labels{"job": "api"}, vals),
	)
}

func benchWideFrame(rows int) *data.Frame {
	ts := make([]time.Time, rows)
	f1 := make([]*float64, rows)
	f2 := make([]*float64, rows)
	i64 := make([]int64, rows)
	strs := make([]string, rows)
	bools := make([]bool, rows)
	start := time.UnixMilli(1_700_000_000_000)
	for i := range rows {
		ts[i] = start.Add(time.Duration(i) * time.Second)
		if i%10 != 0 {
			v := float64(i) / 3
			f1[i] = &v
		}
		if i%7 != 0 {
			v := math.Sqrt(float64(i))
			f2[i] = &v
		}
		i64[i] = int64(i) * 1000
		strs[i] = fmt.Sprintf("host-%d", i%16)
		bools[i] = i%2 == 0
	}
	return data.NewFrame("wide",
		data.NewField("time", nil, ts),
		data.NewField("mean", nil, f1),
		data.NewField("max", nil, f2),
		data.NewField("count", nil, i64),
		data.NewField("host", nil, strs),
		data.NewField("up", nil, bools),
	)
}

func BenchmarkFrameToJSONShapes(b *testing.B) {
	shapes := []struct {
		name  string
		build func(int) *data.Frame
	}{
		{"timeseries", benchTimeSeriesFrame},
		{"wide", benchWideFrame},
	}
	for _, s := range shapes {
		for _, rows := range []int{100, 10_000} {
			f := s.build(rows)
			b.Run(fmt.Sprintf("%s/rows=%d", s.name, rows), func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					if _, err := data.FrameToJSONCache(f); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
