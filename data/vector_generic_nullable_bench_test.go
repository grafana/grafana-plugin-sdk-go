package data

import (
	"fmt"
	"testing"
)

// Benchmarks comparing AppendManyWithNulls indirection costs.
// A: callback (current API) — func(int) bool passed in, indirect call per row.
// B: arrow-style []byte validity bitmap inlined.
// C: []bool validity slice inlined.
// Same output in all cases: []*float64 with nils at null positions.

func benchFixture(n int, nullPct int) (vals []float64, bitmap []byte, valid []bool, isNull func(int) bool) {
	vals = make([]float64, n)
	bitmap = make([]byte, (n+7)/8)
	valid = make([]bool, n)
	for i := 0; i < n; i++ {
		vals[i] = float64(i)
		// Deterministic pseudo-random null pattern from nullPct.
		isNullRow := (i*2654435761)%100 < nullPct
		if !isNullRow {
			bitmap[i/8] |= 1 << (uint(i) % 8)
			valid[i] = true
		}
	}
	isNull = func(i int) bool {
		return bitmap[i/8]&(1<<(uint(i)%8)) == 0
	}
	return
}

// Variant A — current callback API.
func appendManyWithNullsCallback[T any](v *nullableGenericVector[T], vals []T, isNull func(int) bool) {
	startIdx := len(v.data)
	v.data = append(v.data, make([]*T, len(vals))...)
	for i, val := range vals {
		if !isNull(i) {
			valCopy := val
			v.data[startIdx+i] = &valCopy
		}
	}
}

// Variant B — arrow-style []byte bitmap inlined.
func appendManyWithNullsBitmap[T any](v *nullableGenericVector[T], vals []T, bitmap []byte) {
	startIdx := len(v.data)
	v.data = append(v.data, make([]*T, len(vals))...)
	for i, val := range vals {
		if bitmap[i/8]&(1<<(uint(i)%8)) != 0 {
			valCopy := val
			v.data[startIdx+i] = &valCopy
		}
	}
}

// Variant C — []bool validity slice inlined.
func appendManyWithNullsBool[T any](v *nullableGenericVector[T], vals []T, valid []bool) {
	startIdx := len(v.data)
	v.data = append(v.data, make([]*T, len(vals))...)
	for i, val := range vals {
		if valid[i] {
			valCopy := val
			v.data[startIdx+i] = &valCopy
		}
	}
}

var benchSizes = []int{1024, 10240}
var benchNullPcts = []int{10, 50, 90}

func BenchmarkAppendManyWithNulls(b *testing.B) {
	for _, n := range benchSizes {
		for _, pct := range benchNullPcts {
			vals, bitmap, valid, isNull := benchFixture(n, pct)

			b.Run(fmt.Sprintf("A_callback/n=%d/null=%d", n, pct), func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					v := newNullableGenericVector[float64](0)
					appendManyWithNullsCallback(v, vals, isNull)
				}
			})
			b.Run(fmt.Sprintf("B_bitmap/n=%d/null=%d", n, pct), func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					v := newNullableGenericVector[float64](0)
					appendManyWithNullsBitmap(v, vals, bitmap)
				}
			})
			b.Run(fmt.Sprintf("C_bool/n=%d/null=%d", n, pct), func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					v := newNullableGenericVector[float64](0)
					appendManyWithNullsBool(v, vals, valid)
				}
			})
		}
	}
}
