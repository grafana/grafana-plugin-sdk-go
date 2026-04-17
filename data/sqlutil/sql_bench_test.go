package sqlutil_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/data/sqlutil"
)

func newInt64Field() *data.Field {
	return data.NewField("v", nil, []int64{})
}

// Benchmarks for the FrameFromRows hot path. Each benchmark targets one of
// the three optimizations so the contribution of each can be read independently
// from `benchstat`. To compare against the pre-optimization shape, run on the
// commit before this PR and benchstat the two outputs.

// BenchmarkDefaultConverterFunc isolates Fix 2 (precomputing reflect.PointerTo).
// The "cached" variant is the in-tree DefaultConverterFunc; the "uncached"
// variant inlines the pre-optimization shape so the benchmark is self-
// contained and can be diffed with benchstat in a single run.
func BenchmarkDefaultConverterFunc(b *testing.B) {
	t64 := reflect.TypeOf(int64(0))
	v := int64(42)

	b.Run("cached", func(b *testing.B) {
		fn := sqlutil.DefaultConverterFunc(t64)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = fn(&v)
		}
	})

	b.Run("uncached", func(b *testing.B) {
		fn := func(in interface{}) (interface{}, error) { //nolint:unparam // mirrors the shape of DefaultConverterFunc for apples-to-apples comparison
			if reflect.TypeOf(in) == reflect.PointerTo(t64) {
				return reflect.ValueOf(in).Elem().Interface(), nil
			}
			return in, nil
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = fn(&v)
		}
	})
}

// BenchmarkFrameFromRows is the headline benchmark. It runs three variants
// across a row × column matrix:
//   - baseline: a local implementation that mimics the pre-optimization
//     hot loop (fresh scan buffer per row, no presizing).
//   - reuse: in-tree FrameFromRows (scan/converted buffers reused across
//     rows, but fields not presized).
//   - presized: in-tree FrameFromRowsWithCapacity (everything stacked).
//
// Diffing reuse vs baseline shows Fix 3's contribution.
// Diffing presized vs reuse shows Fix 1's contribution.
// Fix 2's contribution is measured in BenchmarkDefaultConverterFunc.
func BenchmarkFrameFromRows(b *testing.B) {
	matrix := []struct {
		rows int
		cols int
	}{
		{100, 5},
		{1000, 5},
		{10000, 5},
		{1000, 20},
		{10000, 20},
	}

	for _, m := range matrix {
		name := fmt.Sprintf("rows=%d/cols=%d", m.rows, m.cols)

		b.Run(name+"/baseline", func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				rows := makeWideRows(b, m.cols, m.rows) //nolint:rowserrcheck // frameFromRowsNoBufferReuse checks rows.Err() internally
				b.StartTimer()
				if err := frameFromRowsNoBufferReuse(rows, m.rows); err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				_ = rows.Close() //nolint:sqlclosecheck // deferred Close would stack across b.N iterations and skew allocs
				b.StartTimer()
			}
		})

		b.Run(name+"/reuse", func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				rows := makeWideRows(b, m.cols, m.rows) //nolint:rowserrcheck // sqlutil.FrameFromRows checks rows.Err() internally
				b.StartTimer()
				_, err := sqlutil.FrameFromRows(rows, -1)
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				_ = rows.Close() //nolint:sqlclosecheck // deferred Close would stack across b.N iterations and skew allocs
				b.StartTimer()
			}
		})

		b.Run(name+"/presized", func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				rows := makeWideRows(b, m.cols, m.rows) //nolint:rowserrcheck // sqlutil.FrameFromRowsWithCapacity checks rows.Err() internally
				b.StartTimer()
				_, err := sqlutil.FrameFromRowsWithCapacity(rows, -1, m.rows)
				if err != nil {
					b.Fatal(err)
				}
				b.StopTimer()
				_ = rows.Close() //nolint:sqlclosecheck // deferred Close would stack across b.N iterations and skew allocs
				b.StartTimer()
			}
		})
	}
}

// BenchmarkFieldGrow_VsAppendGrowth isolates Fix 1 at the primitive level.
// "presized" calls Grow(n) once before n appends; "grow" lets append's
// doubling strategy reallocate as it grows.
func BenchmarkFieldGrow_VsAppendGrowth(b *testing.B) {
	const n = 10000

	b.Run("presized", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			f := newInt64Field()
			f.Grow(n)
			for j := int64(0); j < n; j++ {
				f.Append(j)
			}
		}
	})

	b.Run("grow", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			f := newInt64Field()
			for j := int64(0); j < n; j++ {
				f.Append(j)
			}
		}
	})
}
