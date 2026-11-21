package data_test

import (
	"errors"
	"math/rand"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// laLongFrame builds a long frame shaped like Log Analytics output: a time column,
// one nullable string label and one nullable float value, sorted by time.
func laLongFrame(series, pointsPerSeries, timeSlots int, shared bool, seed int64) *data.Frame {
	r := rand.New(rand.NewSource(seed)) //nolint:gosec // deterministic test data
	type row struct {
		t int
		s int
	}
	rows := make([]row, 0, series*pointsPerSeries)
	for s := range series {
		for p := range pointsPerSeries {
			switch {
			case shared:
				rows = append(rows, row{p, s})
			case timeSlots > 0:
				rows = append(rows, row{r.Intn(timeSlots), s})
			default:
				rows = append(rows, row{s*pointsPerSeries + p, s})
			}
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].t < rows[j].t })
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ts := make([]time.Time, len(rows))
	labels := make([]*string, len(rows))
	values := make([]*float64, len(rows))
	names := make([]string, series)
	for i := range names {
		names[i] = "series-" + strconv.Itoa(i)
	}
	for i, rw := range rows {
		ts[i] = base.Add(time.Duration(rw.t) * time.Second)
		labels[i] = &names[rw.s]
		v := float64(i)
		values[i] = &v
	}
	return data.NewFrame("PrimaryResult",
		data.NewField("TimeGenerated", nil, ts),
		data.NewField("Series", nil, labels),
		data.NewField("Value", nil, values),
	)
}

var sinkFrame *data.Frame

func BenchmarkLongToWideWithLimitShapes(b *testing.B) {
	const limit = 25_000_000
	cases := []struct {
		name  string
		frame *data.Frame
	}{
		{"over_limit_sparse_10k_series_500k_rows", laLongFrame(10_000, 50, 0, false, 1)},
		{"under_limit_dense_2k_series_250_times", laLongFrame(2_000, 250, 0, true, 2)},
		{"under_limit_near_cap_1k_series_20k_slots", laLongFrame(1_000, 500, 20_000, false, 3)},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				f, _ := data.LongToWideWithLimit(c.frame, nil, limit)
				sinkFrame = f
			}
		})
	}
}

func TestLongToWideWithLimitFailsBeforeBuilding(t *testing.T) {
	// 2,000 series at unique times: the wide Frame has 100,000 rows and 2,001 Fields, far over the limit.
	long := laLongFrame(2_000, 50, 0, false, 1)
	const limit = 1_000_000
	if _, err := data.LongToWideWithLimit(long, nil, limit); !errors.Is(err, data.ErrorWideFrameTooLarge) {
		t.Fatalf("LongToWideWithLimit() error = %v, want ErrorWideFrameTooLarge", err)
	}
	res := testing.Benchmark(func(b *testing.B) {
		b.Helper()
		b.ReportAllocs()
		for b.Loop() {
			_, _ = data.LongToWideWithLimit(long, nil, limit)
		}
	})
	if got := res.AllocedBytesPerOp(); got > 1<<20 {
		t.Errorf("an over-limit conversion allocates %d bytes, want at most 1 MiB: the limit must be checked before the wide Frame is built", got)
	}
}

func randomLongFrame(r *rand.Rand) *data.Frame {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	nRows := 1 + r.Intn(300)
	slots := 1 + r.Intn(60)
	nFactors := 1 + r.Intn(2)
	nValues := 1 + r.Intn(2)
	boolFactor := r.Intn(4) == 0
	times := make([]int, nRows)
	for i := range times {
		times[i] = r.Intn(slots)
	}
	sort.Ints(times)
	ts := make([]time.Time, nRows)
	for i, s := range times {
		ts[i] = base.Add(time.Duration(s) * time.Minute)
	}
	fields := []*data.Field{data.NewField("time", nil, ts)}
	for f := range nFactors {
		if boolFactor && f == 0 {
			vals := make([]bool, nRows)
			for i := range vals {
				vals[i] = r.Intn(2) == 0
			}
			fields = append(fields, data.NewField("b"+strconv.Itoa(f), nil, vals))
			continue
		}
		vals := make([]*string, nRows)
		for i := range vals {
			s := "v" + strconv.Itoa(r.Intn(1+r.Intn(12)))
			vals[i] = &s
		}
		fields = append(fields, data.NewField("f"+strconv.Itoa(f), nil, vals))
	}
	for v := range nValues {
		vals := make([]*float64, nRows)
		for i := range vals {
			if r.Intn(10) > 0 {
				x := r.Float64()
				vals[i] = &x
			}
		}
		fields = append(fields, data.NewField("val"+strconv.Itoa(v), nil, vals))
	}
	return data.NewFrame("long", fields...)
}

func TestLongToWideWithLimitUnderTheLimitIsUnchanged(t *testing.T) {
	r := rand.New(rand.NewSource(42)) //nolint:gosec // deterministic test data
	fills := []*data.FillMissing{nil, {Mode: data.FillModeNull}, {Mode: data.FillModeValue, Value: -1}, {Mode: data.FillModePrevious}}
	for trial := range 500 {
		long := randomLongFrame(r)
		fill := fills[trial%len(fills)]
		want, err := data.LongToWide(long, fill)
		if err != nil {
			t.Fatalf("trial %d: LongToWide() error = %v", trial, err)
		}
		rows, _ := want.RowLen()
		cells := int64(rows * len(want.Fields))

		got, err := data.LongToWideWithLimit(long, fill, cells)
		if err != nil {
			t.Fatalf("trial %d: a limit equal to the wide size (%d cells) failed: %v", trial, cells, err)
		}
		if diff := cmp.Diff(want, got, data.FrameTestCompareOptions()...); diff != "" {
			t.Fatalf("trial %d: the wide Frame differs (-want +got):\n%s", trial, diff)
		}
		if cells > 1 {
			if _, err := data.LongToWideWithLimit(long, fill, cells-1); !errors.Is(err, data.ErrorWideFrameTooLarge) {
				t.Fatalf("trial %d: a limit one below the wide size (%d cells) gave %v, want ErrorWideFrameTooLarge", trial, cells, err)
			}
		}
	}
}

func TestLongToWideWithLimitInputsTheCheckSkips(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s := func(v string) *string { return &v }
	tests := []struct {
		name string
		long *data.Frame
	}{
		{"unsorted times", data.NewFrame("", data.NewField("t", nil, []time.Time{base.Add(time.Minute), base}), data.NewField("f", nil, []string{"a", "b"}), data.NewField("v", nil, []float64{1, 2}))},
		{"null time", data.NewFrame("", data.NewField("t", nil, []*time.Time{&base, nil}), data.NewField("f", nil, []string{"a", "b"}), data.NewField("v", nil, []float64{1, 2}))},
		{"null factor", data.NewFrame("", data.NewField("t", nil, []time.Time{base, base}), data.NewField("f", nil, []*string{s("a"), nil}), data.NewField("v", nil, []float64{1, 2}))},
		{"int factor", data.NewFrame("", data.NewField("t", nil, []time.Time{base, base}), data.NewField("f", nil, []int64{1, 2}), data.NewField("v", nil, []float64{1, 2}))},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			want, wantErr := data.LongToWide(tc.long, nil)
			got, gotErr := data.LongToWideWithLimit(tc.long, nil, 1_000_000)
			if (wantErr == nil) != (gotErr == nil) || (wantErr != nil && wantErr.Error() != gotErr.Error()) {
				t.Fatalf("error = %v, want %v", gotErr, wantErr)
			}
			if wantErr == nil {
				if diff := cmp.Diff(want, got, data.FrameTestCompareOptions()...); diff != "" {
					t.Fatalf("the wide Frame differs (-want +got):\n%s", diff)
				}
			}
		})
	}
}
