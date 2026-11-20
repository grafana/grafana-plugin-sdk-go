package data_test

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func TestJoinContinuation(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	times := func(offsets ...int) []time.Time {
		out := make([]time.Time, len(offsets))
		for i, o := range offsets {
			out[i] = t0.Add(time.Duration(o) * time.Minute)
		}
		return out
	}
	series := func(name string, values any) *data.Field {
		return data.NewField("value", data.Labels{"metric": name}, values)
	}
	continues := func(fillValue *float64) *data.FrameMeta {
		return &data.FrameMeta{Continues: &data.FrameContinuation{FrameID: "1", FillValue: fillValue}}
	}
	earlierMeta := &data.FrameMeta{ExecutedQueryString: "SELECT 1"}
	fill := 7.5

	tests := []struct {
		name    string
		earlier *data.Frame
		next    *data.Frame
		want    *data.Frame
	}{
		{
			name: "new series at the end",
			earlier: data.NewFrame("frame",
				data.NewField("time", nil, times(0, 1)),
				series("a", []*float64{new(1.0), new(2.0)}),
			).SetMeta(earlierMeta),
			next: data.NewFrame("other name",
				data.NewField("time", nil, times(2)),
				series("a", []*float64{new(3.0)}),
				series("b", []*float64{new(4.0)}),
			).SetMeta(continues(nil)),
			want: data.NewFrame("frame",
				data.NewField("time", nil, times(0, 1, 2)),
				series("a", []*float64{new(1.0), new(2.0), new(3.0)}),
				series("b", []*float64{nil, nil, new(4.0)}),
			).SetMeta(earlierMeta),
		},
		{
			name: "new series at the start and in the middle",
			earlier: data.NewFrame("frame",
				data.NewField("time", nil, times(0)),
				series("b", []*float64{new(1.0)}),
				series("d", []*float64{new(2.0)}),
			),
			next: data.NewFrame("frame",
				data.NewField("time", nil, times(1)),
				series("a", []*float64{new(3.0)}),
				series("b", []*float64{new(4.0)}),
				series("c", []*float64{new(5.0)}),
				series("d", []*float64{new(6.0)}),
			).SetMeta(continues(nil)),
			want: data.NewFrame("frame",
				data.NewField("time", nil, times(0, 1)),
				series("a", []*float64{nil, new(3.0)}),
				series("b", []*float64{new(1.0), new(4.0)}),
				series("c", []*float64{nil, new(5.0)}),
				series("d", []*float64{new(2.0), new(6.0)}),
			),
		},
		{
			name: "fill value in nullable and non-nullable fields of other types",
			earlier: data.NewFrame("frame",
				data.NewField("time", nil, times(0, 1)),
				series("a", []float64{1, 2}),
			),
			next: data.NewFrame("frame",
				data.NewField("time", nil, times(2)),
				series("a", []float64{3}),
				series("b", []float64{4}),
				series("c", []*float64{new(5.0)}),
				series("d", []int64{6}),
			).SetMeta(continues(&fill)),
			want: data.NewFrame("frame",
				data.NewField("time", nil, times(0, 1, 2)),
				series("a", []float64{1, 2, 3}),
				series("b", []float64{7.5, 7.5, 4}),
				series("c", []*float64{new(7.5), new(7.5), new(5.0)}),
				series("d", []int64{7, 7, 6}),
			),
		},
		{
			name: "zero value in a non-nullable field without a fill value",
			earlier: data.NewFrame("frame",
				data.NewField("time", nil, times(0)),
				series("a", []float64{1}),
			),
			next: data.NewFrame("frame",
				data.NewField("time", nil, times(1)),
				series("a", []float64{2}),
				series("b", []float64{3}),
			).SetMeta(continues(nil)),
			want: data.NewFrame("frame",
				data.NewField("time", nil, times(0, 1)),
				series("a", []float64{1, 2}),
				series("b", []float64{0, 3}),
			),
		},
		{
			name: "fields with the same name and labels match in order",
			earlier: data.NewFrame("frame",
				data.NewField("value", nil, []int64{1}),
				data.NewField("value", nil, []int64{2}),
			),
			next: data.NewFrame("frame",
				data.NewField("value", nil, []int64{3}),
				data.NewField("value", nil, []int64{4}),
				data.NewField("value", nil, []int64{5}),
			).SetMeta(continues(nil)),
			want: data.NewFrame("frame",
				data.NewField("value", nil, []int64{1, 3}),
				data.NewField("value", nil, []int64{2, 4}),
				data.NewField("value", nil, []int64{0, 5}),
			),
		},
		{
			name: "earlier frame without rows",
			earlier: data.NewFrame("frame",
				data.NewField("time", nil, times()),
				series("a", []*float64{}),
			),
			next: data.NewFrame("frame",
				data.NewField("time", nil, times(0)),
				series("a", []*float64{new(1.0)}),
				series("b", []*float64{new(2.0)}),
			).SetMeta(continues(&fill)),
			want: data.NewFrame("frame",
				data.NewField("time", nil, times(0)),
				series("a", []*float64{new(1.0)}),
				series("b", []*float64{new(2.0)}),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, data.JoinContinuation(tt.earlier, tt.next))
			if diff := cmp.Diff(tt.want, tt.earlier, data.FrameTestCompareOptions()...); diff != "" {
				t.Errorf("joined frame mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestJoinContinuationErrors(t *testing.T) {
	continues := &data.FrameMeta{Continues: &data.FrameContinuation{FrameID: "1"}}
	oneField := func() *data.Frame { return data.NewFrame("frame", data.NewField("a", nil, []int64{1})) }

	tests := []struct {
		name string
		// earlier makes the earlier frame, so that the test can compare it with a frame before the join.
		earlier func() *data.Frame
		next    *data.Frame
		err     string
	}{
		{
			name:    "no continuation in the meta",
			earlier: oneField,
			next:    data.NewFrame("frame", data.NewField("a", nil, []int64{2})),
			err:     "the frame does not continue an earlier frame",
		},
		{
			name:    "other field type",
			earlier: oneField,
			next:    data.NewFrame("frame", data.NewField("a", nil, []float64{2})).SetMeta(continues),
			err:     `field "a" has type []int64 in the earlier frame and []float64 in the continuing frame`,
		},
		{
			name: "field of the earlier frame is missing",
			earlier: func() *data.Frame {
				return data.NewFrame("frame",
					data.NewField("a", nil, []int64{1}),
					data.NewField("b", data.Labels{"metric": "x"}, []int64{2}),
				)
			},
			next: data.NewFrame("frame", data.NewField("a", nil, []int64{3})).SetMeta(continues),
			err:  `field "b" metric=x of the earlier frame is missing in the continuing frame`,
		},
		{
			name: "earlier frame with fields of different lengths",
			earlier: func() *data.Frame {
				return data.NewFrame("frame",
					data.NewField("a", nil, []int64{1}),
					data.NewField("b", nil, []int64{2, 3}),
				)
			},
			next: data.NewFrame("frame", data.NewField("a", nil, []int64{4})).SetMeta(continues),
			err:  "earlier frame:",
		},
		{
			name:    "continuing frame with fields of different lengths",
			earlier: oneField,
			next: data.NewFrame("frame",
				data.NewField("a", nil, []int64{2}),
				data.NewField("b", nil, []int64{3, 4}),
			).SetMeta(continues),
			err: "continuing frame:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			earlier := tt.earlier()
			err := data.JoinContinuation(earlier, tt.next)
			require.ErrorContains(t, err, tt.err)
			if diff := cmp.Diff(tt.earlier(), earlier, data.FrameTestCompareOptions()...); diff != "" {
				t.Errorf("earlier frame changed on an error (-before +after):\n%s", diff)
			}
		})
	}
}

// TestJoinContinuationMatchesLongToWide shows that a long series converted in two parts and joined gives the frame
// of one conversion of all rows, when a series starts in the second part.
func TestJoinContinuationMatchesLongToWide(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	// The series a has a row at each minute from 0 to 5, and the series b starts at minute 3.
	long := func(fromMinute, toMinute int) *data.Frame {
		frame := data.NewFrame("frame",
			data.NewField("time", nil, []time.Time{}),
			data.NewField("metric", nil, []string{}),
			data.NewField("value", nil, []float64{}),
		)
		for m := fromMinute; m < toMinute; m++ {
			frame.AppendRow(t0.Add(time.Duration(m)*time.Minute), "a", float64(m))
			if m >= 3 {
				frame.AppendRow(t0.Add(time.Duration(m)*time.Minute), "b", float64(10*m))
			}
		}
		return frame
	}

	tests := []struct {
		name      string
		fill      *data.FillMissing
		fillValue *float64
	}{
		{name: "no fill"},
		{name: "fill null", fill: &data.FillMissing{Mode: data.FillModeNull}},
		{name: "fill previous", fill: &data.FillMissing{Mode: data.FillModePrevious}},
		{name: "fill value", fill: &data.FillMissing{Mode: data.FillModeValue, Value: 42}, fillValue: new(42.0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, err := data.LongToWide(long(0, 6), tt.fill)
			require.NoError(t, err)

			earlier, err := data.LongToWide(long(0, 3), tt.fill)
			require.NoError(t, err)
			next, err := data.LongToWide(long(3, 6), tt.fill)
			require.NoError(t, err)
			next.Meta = &data.FrameMeta{
				Type:        data.FrameTypeTimeSeriesWide,
				TypeVersion: data.FrameTypeVersion{0, 1},
				Continues:   &data.FrameContinuation{FrameID: "1", FillValue: tt.fillValue},
			}

			require.NoError(t, data.JoinContinuation(earlier, next))
			if diff := cmp.Diff(want, earlier, data.FrameTestCompareOptions()...); diff != "" {
				t.Errorf("joined frame differs from one conversion (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFrameContinuationRoundTrip(t *testing.T) {
	fillValue := 1.5
	frame := data.NewFrame("frame", data.NewField("value", nil, []int64{1})).SetMeta(&data.FrameMeta{
		Continues: &data.FrameContinuation{FrameID: "1", FillValue: &fillValue},
	})

	t.Run("json", func(t *testing.T) {
		b, err := frame.MarshalJSON()
		require.NoError(t, err)
		require.Contains(t, string(b), `"continues":{"frameId":"1","fillValue":1.5}`)
		got := &data.Frame{}
		require.NoError(t, got.UnmarshalJSON(b))
		require.Equal(t, frame.Meta, got.Meta)
	})

	t.Run("arrow", func(t *testing.T) {
		b, err := frame.MarshalArrow()
		require.NoError(t, err)
		got, err := data.UnmarshalArrowFrame(b)
		require.NoError(t, err)
		require.Equal(t, frame.Meta, got.Meta)
	})
}
