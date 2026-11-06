package sqlexprtest

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMalformedFramesFollowConversionRoute(t *testing.T) {
	table := func(fields ...*data.Field) *data.Frame {
		return typed(data.FrameTypeTable, data.NewFrame("", fields...))
	}
	series := func(typ data.FrameType, nTime, nValue int) *data.Frame {
		return typed(typ, data.NewFrame("", data.NewField("time", nil, make([]time.Time, nTime)), data.NewField("v", nil, make([]float64, nValue))))
	}
	numeric := func(typ data.FrameType, n int) *data.Frame {
		return typed(typ, data.NewFrame("", data.NewField("v", nil, make([]float64, n))))
	}
	cases := []struct {
		name     string
		frames   data.Frames
		mode     Mode
		rows     int
		code     Code
		severity Severity
		columns  int
	}{
		{"first nil", data.Frames{nil}, ModeNone, 0, CodeNilFrame, SeverityError, 0},
		{"later nil table is not visited", data.Frames{table(), nil}, ModeNone, 0, CodeMultipleTableFrames, SeverityError, 0},
		{"later nil wide is not visited", data.Frames{numeric(data.FrameTypeNumericWide, 1), nil}, ModeNone, 0, CodeWideFrameCount, SeverityError, 0},
		{"later nil numeric multi", data.Frames{numeric(data.FrameTypeNumericMulti, 1), nil}, ModeNone, 0, CodeNilFrame, SeverityError, 0},
		{"later nil series multi", data.Frames{series(data.FrameTypeTimeSeriesMulti, 1, 1), nil}, ModeNone, 0, CodeNilFrame, SeverityError, 0},
		{"nil table field", data.Frames{table(nil)}, ModeNone, 0, CodeNilField, SeverityError, 0},
		{"table storage fails during schema inspection", data.Frames{table(&data.Field{Name: "v"})}, ModeTable, 0, CodeInvalidField, SeverityError, 1},
		{"later table storage", data.Frames{table(data.NewField("n", nil, []float64{1}), &data.Field{Name: "v"})}, ModeTable, 1, CodeInvalidField, SeverityError, 2},
		{"first table storage before a longer field", data.Frames{table(&data.Field{Name: "v"}, data.NewField("n", nil, []float64{1, 2}))}, ModeTable, 0, CodeInvalidField, SeverityError, 2},
		{"table shorter field", data.Frames{table(data.NewField("a", nil, []float64{1, 2}), data.NewField("b", nil, []float64{3}))}, ModeTable, 2, CodeUnequalFieldLengths, SeverityError, 2},
		{"table longer field", data.Frames{table(data.NewField("a", nil, []float64{1}), data.NewField("b", nil, []float64{2, 3}))}, ModeTable, 1, CodeUnequalFieldLengths, SeverityWarning, 2},
		{"multi too many values", data.Frames{series(data.FrameTypeTimeSeriesMulti, 1, 2)}, ModeNone, 0, CodeUnequalFieldLengths, SeverityError, 0},
		{"multi extra timestamps", data.Frames{series(data.FrameTypeTimeSeriesMulti, 2, 1)}, ModeFullLong, 1, CodeUnequalFieldLengths, SeverityWarning, 3},
		{"wide too few values", data.Frames{series(data.FrameTypeTimeSeriesWide, 2, 1)}, ModeNone, 0, CodeUnequalFieldLengths, SeverityError, 0},
		{"wide extra values", data.Frames{series(data.FrameTypeTimeSeriesWide, 1, 2)}, ModeFullLong, 1, CodeUnequalFieldLengths, SeverityWarning, 3},
		{"wide row rejection before nil field", data.Frames{typed(data.FrameTypeNumericWide, data.NewFrame("", data.NewField("a", nil, []float64{1, 2}), nil))}, ModeNone, 0, CodeNumericRowCount, SeverityError, 0},
		{"multi merge visits nil before row rejection", data.Frames{typed(data.FrameTypeNumericMulti, data.NewFrame("", data.NewField("a", nil, []float64{1, 2}), nil))}, ModeNone, 0, CodeNilField, SeverityError, 0},
		{"missing time precedes later nil frame", data.Frames{numeric(data.FrameTypeTimeSeriesMulti, 1), nil}, ModeNone, 0, CodeMissingTimeField, SeverityError, 0},
		{"multi length error precedes later nil field", data.Frames{typed(data.FrameTypeTimeSeriesMulti, data.NewFrame("", data.NewField("time", nil, []time.Time{t0}), data.NewField("v", nil, []float64{1, 2}), nil))}, ModeNone, 0, CodeUnequalFieldLengths, SeverityError, 0},
		{"wide metadata visits nil before row read", data.Frames{typed(data.FrameTypeTimeSeriesWide, data.NewFrame("", data.NewField("time", nil, []time.Time{t0, t0}), data.NewField("v", nil, []float64{1}), nil))}, ModeNone, 0, CodeNilField, SeverityError, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var r *Report
			require.NotPanics(t, func() { r = Check(tc.frames) })
			assert.Equal(t, tc.mode, r.Mode)
			assert.Equal(t, tc.rows, r.Rows)
			assert.Equal(t, []Code{tc.code}, codes(r))
			assert.Equal(t, tc.severity, r.Worst())
			assert.Len(t, r.Columns, tc.columns)
			if tc.mode == ModeNone {
				assert.NotContains(t, r.String(), ", rows ")
			}
		})
	}
	for _, typ := range []data.FrameType{data.FrameTypeNumericWide, data.FrameTypeNumericMulti, data.FrameTypeTimeSeriesWide, data.FrameTypeTimeSeriesMulti} {
		for _, f := range []*data.Field{nil, {Name: "v"}} {
			t.Run(string(typ)+" malformed field", func(t *testing.T) {
				r := Check(data.Frames{typed(typ, data.NewFrame("", f))})
				assert.Equal(t, ModeNone, r.Mode)
				assert.Equal(t, SeverityError, r.Worst())
				if f == nil {
					assert.Equal(t, []Code{CodeNilField}, codes(r))
				} else {
					assert.Equal(t, []Code{CodeInvalidField}, codes(r))
				}
			})
		}
	}
}

func TestConvertedRows(t *testing.T) {
	for _, typ := range []data.FrameType{data.FrameTypeNumericWide, data.FrameTypeNumericMulti, data.FrameTypeTimeSeriesWide, data.FrameTypeTimeSeriesMulti} {
		t.Run(string(typ), func(t *testing.T) {
			fields := []*data.Field{data.NewField("a", nil, []*float64{nil}), data.NewField("b", nil, []int64{1})}
			if typ == data.FrameTypeTimeSeriesWide || typ == data.FrameTypeTimeSeriesMulti {
				fields = append([]*data.Field{data.NewField("t", nil, []time.Time{t0})}, fields...)
			}
			r := Check(data.Frames{typed(typ, data.NewFrame("", fields...))})
			assert.Equal(t, ModeFullLong, r.Mode)
			assert.Equal(t, 2, r.Rows)
			assert.Empty(t, r.Findings)
			// With no numeric fields, conversion still constructs its generated schema.
			fields = []*data.Field{data.NewField("s", nil, []string{"x"})}
			if typ == data.FrameTypeTimeSeriesWide || typ == data.FrameTypeTimeSeriesMulti {
				fields = append(fields, data.NewField("t", nil, []time.Time{t0}))
			}
			r = Check(data.Frames{typed(typ, data.NewFrame("", fields...))})
			assert.Equal(t, ModeFullLong, r.Mode)
			assert.Zero(t, r.Rows)
			assert.Equal(t, []Code{CodeDroppedField}, codes(r))
		})
	}
}

func TestFloatPrecisionBoundaries(t *testing.T) {
	for _, sign := range []float64{-1, 1} {
		for _, tc := range []struct {
			name     string
			values   any
			want     bool
			boundary string
		}{
			{"float32 below", []float32{float32(sign * ((1 << 24) - 1))}, false, "2^24"},
			{"float32 at", []float32{float32(sign * (1 << 24))}, true, "2^24"},
			{"float32 above nullable", []*float32{nil, new(float32(sign * ((1 << 24) + 2)))}, true, "2^24"},
			{"float64 below", []float64{sign * ((1 << 53) - 1)}, false, "2^53"},
			{"float64 at", []float64{sign * (1 << 53)}, true, "2^53"},
			{"float64 above nullable", []*float64{nil, new(sign * ((1 << 53) + 2))}, true, "2^53"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				r := Check(data.Frames{typed(data.FrameTypeTable, data.NewFrame("", data.NewField("v", nil, tc.values)))})
				if tc.want {
					require.Equal(t, []Code{CodeLargeIntegerInFloat}, codes(r))
					assert.Contains(t, r.Findings[0].Message, tc.boundary)
					assert.Equal(t, SeverityInfo, r.Worst())
				} else {
					assert.Empty(t, r.Findings)
				}
			})
		}
	}
}

func TestValueNormalizations(t *testing.T) {
	for _, values := range []any{[]float32{float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1))}, []float64{math.NaN(), math.Inf(1), math.Inf(-1)}, []*float32{nil, new(float32(math.Inf(1)))}, []*float64{nil, new(math.NaN())}} {
		r := Check(data.Frames{typed(data.FrameTypeTable, data.NewFrame("", data.NewField("v", nil, values)))})
		require.Equal(t, []Code{CodeNonfiniteAsNull}, codes(r))
		assert.Equal(t, SeverityInfo, r.Worst())
		assert.Equal(t, OwnerGrafana, r.Findings[0].Owner)
	}
	for _, values := range []any{[]json.RawMessage{nil, json.RawMessage(`{"x":1}`), json.RawMessage(`null`)}, []*json.RawMessage{nil, new(json.RawMessage(nil)), new(json.RawMessage(`null`))}} {
		assert.Empty(t, Check(data.Frames{typed(data.FrameTypeTable, data.NewFrame("", data.NewField("j", nil, values)))}).Findings)
	}
	r := Check(data.Frames{typed(data.FrameTypeTable, data.NewFrame("", data.NewField("j", nil, []json.RawMessage{json.RawMessage(`broken`), json.RawMessage{}})))})
	require.Equal(t, []Code{CodeInvalidJSON}, codes(r))
	assert.Equal(t, SeverityWarning, r.Worst())
	assert.Equal(t, OwnerPlugin, r.Findings[0].Owner)
}

func TestOnlyConsumedValuesAreDiagnosed(t *testing.T) {
	frame := typed(data.FrameTypeNumericMulti, data.NewFrame("", data.NewField("first", nil, []float64{1}), data.NewField("later", nil, []float64{2, math.NaN(), 1 << 54})))
	assert.Equal(t, []Code{CodeNumericValuesTruncated}, codes(Check(data.Frames{frame})))
	frame = typed(data.FrameTypeTable, data.NewFrame("", data.NewField("first", nil, []float64{1}), data.NewField("later", nil, []json.RawMessage{json.RawMessage(`null`), json.RawMessage(`broken`)})))
	assert.Equal(t, []Code{CodeUnequalFieldLengths}, codes(Check(data.Frames{frame})))
	frame = typed(data.FrameTypeTimeSeriesMulti, data.NewFrame("", data.NewField("time", nil, []time.Time{t0}), data.NewField("large", nil, []float64{1 << 54}), data.NewField("short", nil, []float64{1, 2})))
	assert.Equal(t, []Code{CodeUnequalFieldLengths}, codes(Check(data.Frames{frame})), "fatal shape suppresses optional value scans even in earlier fields")
}

func TestSelectedTimeLabelsAndAdditionalTimeFields(t *testing.T) {
	frame := typed(data.FrameTypeTimeSeriesWide, data.NewFrame("", data.NewField("nullable", nil, []*time.Time{&t0}), data.NewField("selected", data.Labels{"zone": "west"}, []time.Time{t0}), data.NewField("extra", data.Labels{"ignored": "x"}, []time.Time{t0}), data.NewField("v", nil, []float64{1})))
	r := Check(data.Frames{frame})
	require.Equal(t, []Code{CodeDroppedField, CodeDroppedLabel, CodeDroppedField}, codes(r))
	assert.Equal(t, "selected", r.Findings[1].Field)
	assert.Contains(t, r.Findings[1].Message, "field 1")
	assert.Equal(t, []string{"time", "__value__", "__metric_name__"}, columnNames(r))
	assert.Equal(t, 1, r.Rows)
}
