package sqlexprtest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

var t0 = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

func typed(t data.FrameType, f *data.Frame) *data.Frame {
	f.Meta = &data.FrameMeta{Type: t}
	return f
}

func codes(r *Report) []Code {
	out := []Code{}
	for _, f := range r.Findings {
		out = append(out, f.Code)
	}
	return out
}

func columnNames(r *Report) []string {
	out := []string{}
	for _, c := range r.Columns {
		out = append(out, c.Name)
	}
	return out
}

func TestCheckTable(t *testing.T) {
	frame := typed(data.FrameTypeTable, data.NewFrame("",
		data.NewField("time", nil, []time.Time{t0}),
		data.NewField("host", nil, []string{"a"}),
		data.NewField("bytes", nil, []int64{1}),
		data.NewField("ok", nil, []*bool{nil}),
	))

	r := Check(data.Frames{frame})

	require.Equal(t, ModeTable, r.Mode)
	assert.Empty(t, r.Findings)
	assert.Equal(t, []Column{
		{Name: "time", Type: data.FieldTypeTime, SQLType: "TIMESTAMP"},
		{Name: "host", Type: data.FieldTypeString, SQLType: "TEXT"},
		{Name: "bytes", Type: data.FieldTypeInt64, SQLType: "BIGINT"},
		{Name: "ok", Type: data.FieldTypeNullableBool, SQLType: "TINYINT(1)"},
	}, r.Columns)
}

func TestCheckLongTypesPassThroughAsTable(t *testing.T) {
	frame := typed(data.FrameTypeTimeSeriesLong, data.NewFrame("",
		data.NewField("time", nil, []time.Time{t0}),
		data.NewField("host", nil, []string{"a"}),
		data.NewField("value", nil, []float64{1}),
	))

	r := Check(data.Frames{frame})

	assert.Equal(t, ModeTable, r.Mode)
	assert.Empty(t, r.Findings)
	assert.Equal(t, []string{"time", "host", "value"}, columnNames(r))
}

func TestCheckEmpty(t *testing.T) {
	t.Run("no frames", func(t *testing.T) {
		r := Check(nil)
		assert.Equal(t, []Code{CodeNoFrames}, codes(r))
		assert.Equal(t, SeverityWarning, r.Worst())
		assert.Empty(t, r.Columns)
	})

	t.Run("frame without fields", func(t *testing.T) {
		r := Check(data.Frames{data.NewFrame("")})
		assert.Equal(t, []Code{CodeNoFields}, codes(r))
	})

	t.Run("typed fields with zero rows keep the schema", func(t *testing.T) {
		frame := typed(data.FrameTypeTable, data.NewFrame("",
			data.NewField("host", nil, []string{}),
			data.NewField("bytes", nil, []int64{}),
		))
		r := Check(data.Frames{frame})
		assert.Empty(t, r.Findings)
		assert.Equal(t, []string{"host", "bytes"}, columnNames(r))
	})
}

func TestCheckMissingType(t *testing.T) {
	plain := func() *data.Frame {
		return data.NewFrame("", data.NewField("value", nil, []float64{1}))
	}

	t.Run("one frame passes as a table", func(t *testing.T) {
		r := Check(data.Frames{plain()})
		assert.Equal(t, ModeTable, r.Mode)
		assert.Equal(t, []Code{CodeMissingType}, codes(r))
		assert.Equal(t, SeverityInfo, r.Worst())
	})

	t.Run("several frames fail", func(t *testing.T) {
		r := Check(data.Frames{plain(), plain()})
		assert.Equal(t, ModeNone, r.Mode)
		assert.Equal(t, []Code{CodeMissingType}, codes(r))
		assert.Equal(t, SeverityError, r.Worst())
	})

	t.Run("labels fail", func(t *testing.T) {
		frame := data.NewFrame("", data.NewField("value", data.Labels{"host": "a"}, []float64{1}))
		r := Check(data.Frames{frame})
		assert.Equal(t, ModeNone, r.Mode)
		assert.Equal(t, SeverityError, r.Worst())
		assert.Equal(t, "value", r.Findings[0].Field)
	})
}

func TestCheckUnsupportedType(t *testing.T) {
	logs := func() *data.Frame {
		return typed(data.FrameTypeLogLines, data.NewFrame("", data.NewField("line", nil, []string{"x"})))
	}

	assert.Empty(t, Check(data.Frames{logs()}).Findings, "one unlabeled frame passes as a table")

	r := Check(data.Frames{logs(), logs()})
	assert.Equal(t, ModeNone, r.Mode)
	assert.Equal(t, []Code{CodeUnsupportedType}, codes(r))
}

func TestCheckTimeSeriesMulti(t *testing.T) {
	series := func(host string, v float64) *data.Frame {
		return typed(data.FrameTypeTimeSeriesMulti, data.NewFrame("",
			data.NewField("time", nil, []time.Time{t0}),
			data.NewField("cpu", data.Labels{"host": host}, []float64{v}),
		))
	}

	r := Check(data.Frames{series("a", 1), series("b", 2)})

	require.Equal(t, ModeFullLong, r.Mode)
	assert.Empty(t, r.Findings)
	assert.Equal(t, []string{"time", "__value__", "__metric_name__", "host"}, columnNames(r))
	assert.Equal(t, "DOUBLE", r.Columns[1].SQLType)
}

func TestCheckNumericWide(t *testing.T) {
	frame := func(rows int) *data.Frame {
		return typed(data.FrameTypeNumericWide, data.NewFrame("",
			data.NewField("cpu", data.Labels{"host": "a"}, make([]float64, rows)),
		))
	}

	r := Check(data.Frames{frame(1)})
	assert.Equal(t, ModeFullLong, r.Mode)
	assert.Equal(t, []string{"__metric_name__", "__value__", "host"}, columnNames(r))

	r = Check(data.Frames{frame(2)})
	assert.Equal(t, ModeNone, r.Mode)
	assert.Equal(t, []Code{CodeInvalidShape}, codes(r))

	r = Check(data.Frames{frame(1), frame(1)})
	assert.Equal(t, ModeNone, r.Mode)
	assert.Equal(t, []Code{CodeInvalidShape}, codes(r))
}

func TestCheckDisplayNameColumn(t *testing.T) {
	field := data.NewField("cpu", nil, []float64{1})
	field.Config = &data.FieldConfig{DisplayNameFromDS: "CPU"}
	frame := typed(data.FrameTypeTimeSeriesWide, data.NewFrame("", data.NewField("time", nil, []time.Time{t0}), field))

	r := Check(data.Frames{frame})

	assert.Equal(t, []string{"time", "__value__", "__metric_name__", "__display_name__"}, columnNames(r))
}

func TestCheckTimeField(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		frame := typed(data.FrameTypeTimeSeriesMulti, data.NewFrame("", data.NewField("cpu", nil, []float64{1})))
		r := Check(data.Frames{frame})
		assert.Equal(t, ModeNone, r.Mode)
		assert.Equal(t, []Code{CodeInvalidShape}, codes(r))
	})

	t.Run("nullable", func(t *testing.T) {
		frame := typed(data.FrameTypeTimeSeriesMulti, data.NewFrame("",
			data.NewField("time", nil, []*time.Time{&t0}),
			data.NewField("cpu", nil, []float64{1}),
		))
		r := Check(data.Frames{frame})
		assert.Equal(t, ModeNone, r.Mode)
		assert.Equal(t, []Code{CodeNullableTime}, codes(r))
		assert.Equal(t, OwnerGrafana, r.Findings[0].Owner)
	})
}

func TestCheckDroppedFields(t *testing.T) {
	frame := typed(data.FrameTypeTimeSeriesWide, data.NewFrame("",
		data.NewField("time", nil, []time.Time{t0}),
		data.NewField("updated", nil, []time.Time{t0}),
		data.NewField("state", nil, []string{"ok"}),
		data.NewField("cpu", nil, []float64{1}),
	))

	r := Check(data.Frames{frame})

	assert.Equal(t, ModeFullLong, r.Mode)
	assert.Equal(t, []Code{CodeDroppedField, CodeDroppedField}, codes(r))
	assert.Equal(t, "updated", r.Findings[0].Field)
	assert.Equal(t, "state", r.Findings[1].Field)
}

func TestCheckMixedTypes(t *testing.T) {
	ts := typed(data.FrameTypeTimeSeriesMulti, data.NewFrame("",
		data.NewField("time", nil, []time.Time{t0}),
		data.NewField("cpu", nil, []float64{1}),
	))
	num := typed(data.FrameTypeNumericMulti, data.NewFrame("",
		data.NewField("time", nil, []time.Time{t0}),
		data.NewField("total", nil, []float64{1}),
	))

	r := Check(data.Frames{ts, num})

	assert.Equal(t, []Code{CodeMixedTypes}, codes(r))
	assert.Equal(t, 1, r.Findings[0].Frame)
}

func TestCheckIntegerPrecision(t *testing.T) {
	big := int64(maxExactInteger + 1)

	t.Run("integer value converted to float64", func(t *testing.T) {
		frame := typed(data.FrameTypeTimeSeriesWide, data.NewFrame("",
			data.NewField("time", nil, []time.Time{t0, t0.Add(time.Second)}),
			data.NewField("count", nil, []int64{1, big}),
		))
		r := Check(data.Frames{frame})
		assert.Equal(t, []Code{CodeIntegerAsFloat}, codes(r))
		assert.Contains(t, r.Findings[0].Message, "9007199254740993")
	})

	t.Run("integer kept in a table", func(t *testing.T) {
		frame := typed(data.FrameTypeTable, data.NewFrame("", data.NewField("count", nil, []int64{big})))
		assert.Empty(t, Check(data.Frames{frame}).Findings)
	})

	t.Run("large whole float", func(t *testing.T) {
		frame := typed(data.FrameTypeTable, data.NewFrame("",
			data.NewField("id", nil, []*float64{nil, new(float64(big))}),
			data.NewField("ratio", nil, []float64{0.5}),
		))
		r := Check(data.Frames{frame})
		assert.Equal(t, []Code{CodeLargeIntegerInFloat}, codes(r))
		assert.Equal(t, "id", r.Findings[0].Field)
	})
}

func TestCheckEnum(t *testing.T) {
	frame := typed(data.FrameTypeTable, data.NewFrame("", data.NewField("state", nil, []data.EnumItemIndex{0})))

	r := Check(data.Frames{frame})

	assert.Equal(t, []Code{CodeEnumField}, codes(r))
	assert.Equal(t, "JSON", r.Columns[0].SQLType)
}

func TestCheckColumnNames(t *testing.T) {
	t.Run("duplicates ignore case", func(t *testing.T) {
		frame := typed(data.FrameTypeTable, data.NewFrame("",
			data.NewField("host", nil, []string{"a"}),
			data.NewField("Host", nil, []string{"b"}),
		))
		r := Check(data.Frames{frame})
		assert.Equal(t, []Code{CodeDuplicateName}, codes(r))
		assert.Equal(t, "Host", r.Findings[0].Field)
	})

	t.Run("label named like a generated column", func(t *testing.T) {
		frame := typed(data.FrameTypeTimeSeriesMulti, data.NewFrame("",
			data.NewField("time", nil, []time.Time{t0}),
			data.NewField("cpu", data.Labels{"time": "nightly", "host": "a"}, []float64{1}),
		))
		r := Check(data.Frames{frame})
		assert.Equal(t, []Code{CodeGeneratedNameClash}, codes(r))
		assert.Equal(t, "time", r.Findings[0].Field)
	})
}

func TestReportString(t *testing.T) {
	frame := data.NewFrame("", data.NewField("value", nil, []float64{1}))
	frame.RefID = "A"
	r := Check(data.Frames{frame})

	assert.Equal(t, `query "A": mode table
  columns: value DOUBLE
  info [missing-type, plugin] frame 0: the frame has no Meta.Type, so it passes through as a table. `+
		`Set Meta.Type (for example table, timeseries-multi, or numeric-wide), so Grafana knows the shape.`, r.String())
}

func TestCheckNumericRows(t *testing.T) {
	t.Run("only the first value survives", func(t *testing.T) {
		a := typed(data.FrameTypeNumericMulti, data.NewFrame("", data.NewField("a", nil, []float64{1})))
		b := typed(data.FrameTypeNumericMulti, data.NewFrame("", data.NewField("b", nil, []float64{1, 2, 3})))
		r := Check(data.Frames{a, b})
		assert.Equal(t, ModeFullLong, r.Mode)
		assert.Equal(t, []Code{CodeInvalidShape}, codes(r))
		assert.Equal(t, SeverityWarning, r.Worst())
		assert.Equal(t, "b", r.Findings[0].Field)
	})

	t.Run("empty field fails", func(t *testing.T) {
		frame := typed(data.FrameTypeNumericWide, data.NewFrame("", data.NewField("a", nil, []float64{})))
		r := Check(data.Frames{frame})
		assert.Equal(t, ModeNone, r.Mode)
		assert.Equal(t, []Code{CodeInvalidShape}, codes(r))
		assert.Equal(t, OwnerGrafana, r.Findings[0].Owner)
	})
}
