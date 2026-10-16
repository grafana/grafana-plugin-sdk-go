package data_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func promShapedFrames(numSeries, numPoints int) data.Frames {
	frames := make(data.Frames, numSeries)
	base := time.Now()
	for s := 0; s < numSeries; s++ {
		times := make([]time.Time, numPoints)
		values := make([]float64, numPoints)
		for i := 0; i < numPoints; i++ {
			times[i] = base.Add(time.Duration(i) * 5 * time.Minute)
			values[i] = rand.Float64() * 100
		}
		frames[s] = data.NewFrame("",
			data.NewField("Time", nil, times),
			data.NewField("Value", data.Labels{
				"__name__": "http_requests_total",
				"instance": "10.0.0.1:9090",
				"job":      "node",
				"pod":      "pod-12345",
			}, values),
		)
	}
	return frames
}

func BenchmarkArrowRoundTrip_50x288(b *testing.B)   { benchArrowRoundTrip(b, 50, 288) }
func BenchmarkArrowRoundTrip_300x288(b *testing.B)  { benchArrowRoundTrip(b, 300, 288) }
func BenchmarkArrowRoundTrip_1000x288(b *testing.B) { benchArrowRoundTrip(b, 1000, 288) }

func BenchmarkJSONRoundTrip_50x288(b *testing.B)   { benchJSONRoundTrip(b, 50, 288) }
func BenchmarkJSONRoundTrip_300x288(b *testing.B)  { benchJSONRoundTrip(b, 300, 288) }
func BenchmarkJSONRoundTrip_1000x288(b *testing.B) { benchJSONRoundTrip(b, 1000, 288) }

func benchArrowRoundTrip(b *testing.B, numSeries, numPoints int) {
	frames := promShapedFrames(numSeries, numPoints)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		enc, err := frames.MarshalArrow()
		if err != nil {
			b.Fatal(err)
		}
		if _, err := data.UnmarshalArrowFrames(enc); err != nil {
			b.Fatal(err)
		}
	}
}

func benchJSONRoundTrip(b *testing.B, numSeries, numPoints int) {
	frames := promShapedFrames(numSeries, numPoints)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		enc := make([][]byte, len(frames))
		for j, f := range frames {
			raw, err := f.MarshalJSON()
			if err != nil {
				b.Fatal(err)
			}
			enc[j] = raw
		}
		for _, raw := range enc {
			f := &data.Frame{}
			if err := f.UnmarshalJSON(raw); err != nil {
				b.Fatal(err)
			}
		}
	}
}
