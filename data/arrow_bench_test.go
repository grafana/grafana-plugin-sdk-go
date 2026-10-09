package data_test

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func benchmarkShapes() []arrowTestFrame {
	return []arrowTestFrame{
		{name: "instant", frame: timeSeriesFrame(1, 10)},
		{name: "range_240", frame: timeSeriesFrame(240, 10)},
	}
}

func BenchmarkFrameMarshalArrow(b *testing.B) {
	for _, shape := range benchmarkShapes() {
		b.Run(shape.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := shape.frame.MarshalArrow(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkFramesMarshalArrow reports the capacity retained per encoded frame next to the allocations.
func BenchmarkFramesMarshalArrow(b *testing.B) {
	frames := make(data.Frames, 1000)
	for i := range frames {
		frames[i] = timeSeriesFrame(1, 10)
	}
	b.ReportAllocs()
	retained := 0
	for b.Loop() {
		encoded, err := frames.MarshalArrow()
		if err != nil {
			b.Fatal(err)
		}
		retained = 0
		for _, enc := range encoded {
			retained += cap(enc)
		}
	}
	b.ReportMetric(float64(retained)/float64(len(frames)), "retainedB/frame")
}

func BenchmarkUnmarshalArrowFrame(b *testing.B) {
	for _, shape := range benchmarkShapes() {
		encoded, err := shape.frame.MarshalArrow()
		if err != nil {
			b.Fatal(err)
		}
		b.Run(shape.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if _, err := data.UnmarshalArrowFrame(encoded); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
