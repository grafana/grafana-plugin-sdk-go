package data_test

import (
	"runtime"
	"runtime/debug"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func benchmarkShapes() []arrowTestFrame {
	return []arrowTestFrame{
		{name: "instant", frame: timeSeriesFrame(1, 10)},
		{name: "range_240", frame: timeSeriesFrame(240, 10)},
		{name: "range_100k", frame: timeSeriesFrame(100_000, 10)},
		{name: "range_1m", frame: timeSeriesFrame(1_000_000, 10)},
		{name: "logs_5k", frame: logsFrame(5_000)},
		{name: "logs_100k", frame: logsFrame(100_000)},
		{name: "logs_500k", frame: logsFrame(500_000)},
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

// BenchmarkFramesMarshalArrowPeak reports the peak heap above the input while encoding.
// The GC runs at GOGC=1 so the sampled heap tracks live memory. ns/op includes those collections.
func BenchmarkFramesMarshalArrowPeak(b *testing.B) {
	shapes := []struct {
		name   string
		frames func() data.Frames
	}{
		{"range_1x1m", func() data.Frames { return data.Frames{timeSeriesFrame(1_000_000, 10)} }},
		{"range_100x20k", func() data.Frames {
			return repeatFrames(100, func() *data.Frame { return timeSeriesFrame(20_000, 10) })
		}},
		{"logs_1x500k", func() data.Frames { return data.Frames{logsFrame(500_000)} }},
		{"logs_8x50k", func() data.Frames { return repeatFrames(8, func() *data.Frame { return logsFrame(50_000) }) }},
	}
	defer debug.SetGCPercent(debug.SetGCPercent(1))
	for _, shape := range shapes {
		frames := shape.frames()
		b.Run(shape.name, func(b *testing.B) {
			var peak uint64
			for b.Loop() {
				b.StopTimer()
				runtime.GC()
				var before runtime.MemStats
				runtime.ReadMemStats(&before)
				stop := sampleHeapPeak()
				b.StartTimer()
				encoded, err := frames.MarshalArrow()
				if err != nil {
					b.Fatal(err)
				}
				peak = max(peak, stop()-before.HeapAlloc)
				runtime.KeepAlive(encoded)
			}
			b.ReportMetric(float64(peak)/(1<<20), "peakMiB")
		})
	}
}

func repeatFrames(n int, frame func() *data.Frame) data.Frames {
	frames := make(data.Frames, n)
	for i := range frames {
		frames[i] = frame()
	}
	return frames
}

// sampleHeapPeak samples HeapAlloc until the returned function is called, which returns the maximum seen.
func sampleHeapPeak() func() uint64 {
	var peak atomic.Uint64
	stop, done := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(done)
		var m runtime.MemStats
		for {
			runtime.ReadMemStats(&m)
			if m.HeapAlloc > peak.Load() {
				peak.Store(m.HeapAlloc)
			}
			select {
			case <-stop:
				return
			case <-time.After(200 * time.Microsecond):
			}
		}
	}()
	return func() uint64 {
		close(stop)
		<-done
		return peak.Load()
	}
}
