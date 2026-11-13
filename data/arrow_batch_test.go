package data_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func TestMarshalArrowBatchOwnership(t *testing.T) {
	small := data.NewFrame("small", data.NewField("value", data.Labels{"id": "small"}, []float64{1}))
	large := data.NewFrame("large", data.NewField("text", nil, []string{strings.Repeat("x", 128*1024)}))
	empty := data.NewFrame("empty", data.NewField("value", nil, []float64{}))
	for name, frames := range map[string]data.Frames{
		"empty batch":         {},
		"single small":        {small},
		"single large":        {large},
		"reuse":               {small, goldenDF(), empty, small},
		"large between small": {small, large, small, empty, small},
		"large last":          {small, large},
		"empty schema":        {data.NewFrame("no fields"), small},
	} {
		t.Run(name, func(t *testing.T) {
			want := make([][]byte, len(frames))
			for i, f := range frames {
				var err error
				want[i], err = f.MarshalArrow()
				require.NoError(t, err)
			}
			got, err := frames.MarshalArrow()
			require.NoError(t, err)
			require.Equal(t, want, got)
			for _, encoded := range got {
				_, err := data.UnmarshalArrowFrame(encoded)
				require.NoError(t, err)
			}

			// Encoding another batch must not overwrite previously returned data.
			_, err = frames.MarshalArrow()
			require.NoError(t, err)
			require.Equal(t, want, got)
			if len(got) > 1 {
				// The outputs within a batch must not alias one another either.
				got[0][0] ^= 0xff
				require.Equal(t, want[1:], got[1:])
			}
		})
	}
}

func TestMarshalArrowBatchError(t *testing.T) {
	valid := data.NewFrame("valid", data.NewField("value", nil, []float64{1}))
	invalid := data.NewFrame("invalid", data.NewField("a", nil, []float64{1}), data.NewField("b", nil, []float64{}))
	for name, frames := range map[string]data.Frames{
		"nil first":       {nil, valid},
		"nil after valid": {valid, nil},
		"unequal lengths": {valid, invalid},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := frames.MarshalArrow()
			require.Error(t, err)
			require.Nil(t, got)
		})
	}
}

func TestMarshalArrowConcurrentBatches(t *testing.T) {
	frames := data.Frames{goldenDF(), data.NewFrame("value", data.NewField("value", nil, []float64{42}))}
	want, err := frames.MarshalArrow()
	require.NoError(t, err)
	for i := range 16 {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Parallel()
			for range 10 {
				got, err := frames.MarshalArrow()
				require.NoError(t, err)
				require.Equal(t, want, got)
			}
		})
	}
}

func arrowBatchBenchmarkFrames(series, rows int) data.Frames {
	frames := make(data.Frames, series)
	for i := range frames {
		timestamps := make([]time.Time, rows)
		values := make([]float64, rows)
		for j := range rows {
			timestamps[j] = time.Unix(1700000000+int64(j), 0)
			values[j] = float64(i+j) / 3
		}
		frames[i] = data.NewFrame("series", data.NewField("Time", nil, timestamps), data.NewField("Value", data.Labels{"id": fmt.Sprint(i)}, values))
	}
	return frames
}

func BenchmarkMarshalArrowBatch(b *testing.B) {
	for _, tc := range []struct {
		name   string
		frames data.Frames
	}{
		{"single-small", arrowBatchBenchmarkFrames(1, 1)},
		{"instant-1000", arrowBatchBenchmarkFrames(1000, 1)},
		{"range-100x1000", arrowBatchBenchmarkFrames(100, 1000)},
		{"single-large", arrowBatchBenchmarkFrames(1, 10000)},
		{"mixed-small-large-small", data.Frames{arrowBatchBenchmarkFrames(1, 1)[0], arrowBatchBenchmarkFrames(1, 10000)[0], arrowBatchBenchmarkFrames(1, 1)[0]}},
	} {
		b.Run(tc.name, func(b *testing.B) {
			encoded, err := tc.frames.MarshalArrow()
			require.NoError(b, err)
			var output, capacity int
			for _, frame := range encoded {
				output += len(frame)
				capacity += cap(frame)
			}
			b.ReportAllocs()
			for b.Loop() {
				if _, err := tc.frames.MarshalArrow(); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportMetric(float64(output), "output-B/op")
			b.ReportMetric(float64(capacity), "capacity-B/op")
		})
	}
}

func BenchmarkMarshalArrowBatchParallel(b *testing.B) {
	frames := arrowBatchBenchmarkFrames(100, 1)
	want, err := frames.MarshalArrow()
	require.NoError(b, err)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			got, err := frames.MarshalArrow()
			if err != nil {
				b.Error(err)
				return
			}
			for i := range want {
				if !bytes.Equal(want[i], got[i]) {
					b.Error("concurrent batch output changed")
					return
				}
			}
		}
	})
}
