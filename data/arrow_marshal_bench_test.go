package data_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// BenchmarkFramesMarshalArrowSmallFrames covers responses with one small frame
// per series, such as instant queries returning many series.
func BenchmarkFramesMarshalArrowSmallFrames(b *testing.B) {
	ts := time.Unix(1700000000, 0)
	frames := make(data.Frames, 25000)
	for i := range frames {
		frames[i] = data.NewFrame("",
			data.NewField("Time", nil, []time.Time{ts}),
			data.NewField("Value", data.Labels{"job": "kube-state-metrics", "pod": fmt.Sprintf("pod-%d", i)}, []float64{1}),
		)
		frames[i].Meta = &data.FrameMeta{Type: data.FrameTypeNumericMulti}
	}
	b.ReportAllocs()
	for b.Loop() {
		if _, err := frames.MarshalArrow(); err != nil {
			b.Fatal(err)
		}
	}
}
