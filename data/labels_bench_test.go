package data

import (
	"strconv"
	"testing"
)

func buildLabels(n int) Labels {
	l := make(Labels, n)
	for i := 0; i < n; i++ {
		l["label_"+strconv.Itoa(i)] = "value_" + strconv.Itoa(i)
	}
	return l
}

func BenchmarkLabelsMarshalJSON(b *testing.B) {
	for _, n := range []int{1, 3, 10} {
		l := buildLabels(n)
		b.Run(strconv.Itoa(n)+"_labels", func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := l.MarshalJSON(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
