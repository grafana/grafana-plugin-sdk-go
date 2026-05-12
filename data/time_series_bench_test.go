package data

import (
	"strconv"
	"testing"
)

func buildTupleLabels(b *testing.B, n int) tupleLabels {
	b.Helper()
	t := make(tupleLabels, 0, n)
	for i := 0; i < n; i++ {
		t = append(t, tupleLabel{
			"label_" + strconv.Itoa(i),
			"value_" + strconv.Itoa(i),
		})
	}
	return t
}

func BenchmarkTupleLabelsMapKey(b *testing.B) {
	for _, n := range []int{1, 3, 10} {
		tl := buildTupleLabels(b, n)
		b.Run(strconv.Itoa(n)+"_labels", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := tl.MapKey(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkLabelsTupleKey(b *testing.B) {
	for _, n := range []int{1, 3, 10} {
		l := make(Labels, n)
		for i := 0; i < n; i++ {
			l["label_"+strconv.Itoa(i)] = "value_" + strconv.Itoa(i)
		}
		b.Run(strconv.Itoa(n)+"_labels", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := labelsTupleKey(l); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
