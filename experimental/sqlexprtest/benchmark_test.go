package sqlexprtest_test

import (
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/sqlexprtest"
)

func BenchmarkSQLInputCheck(b *testing.B) {
	b.Run("table_100000_rows", func(b *testing.B) {
		frame := data.NewFrame("", data.NewField("value", nil, make([]float64, 100000))).SetMeta(&data.FrameMeta{Type: data.FrameTypeTable})
		b.ReportAllocs()
		for b.Loop() {
			r := sqlexprtest.Check(data.Frames{frame})
			if r.Rows != 100000 {
				b.Fatal(r)
			}
		}
	})
	b.Run("wide_50_series_1000_timestamps", func(b *testing.B) {
		frame := data.NewFrame("", data.NewField("time", nil, make([]time.Time, 1000))).SetMeta(&data.FrameMeta{Type: data.FrameTypeTimeSeriesWide})
		for range 50 {
			frame.Fields = append(frame.Fields, data.NewField("value", nil, make([]float64, 1000)))
		}
		b.ReportAllocs()
		for b.Loop() {
			r := sqlexprtest.Check(data.Frames{frame})
			if r.Rows != 50000 || r.Cells() != 150000 {
				b.Fatal(r)
			}
		}
	})
}
