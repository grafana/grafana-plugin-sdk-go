package data

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
)

// writeDataFrameDataReference is the boxing implementation that writeDataFrameData replaced.
func writeDataFrameDataReference(frame *Frame, stream *jsoniter.Stream) {
	rowCount, err := frame.RowLen()
	if err != nil {
		stream.Error = err
		return
	}

	stream.WriteObjectStart()

	entities := make([]*fieldEntityLookup, len(frame.Fields))
	entityCount := 0

	nanos := make([][]int64, len(frame.Fields))
	nsOffSetCount := 0

	stream.WriteObjectField("values")
	stream.WriteArrayStart()
	for fidx, f := range frame.Fields {
		if fidx > 0 {
			stream.WriteMore()
		}
		isTime := f.Type().Time()
		nsTime := make([]int64, rowCount)
		var hasNSTime bool
		isFloat := f.Type() == FieldTypeFloat64 || f.Type() == FieldTypeNullableFloat64 ||
			f.Type() == FieldTypeFloat32 || f.Type() == FieldTypeNullableFloat32

		stream.WriteArrayStart()
		for i := range rowCount {
			if i > 0 {
				stream.WriteRaw(",")
			}
			if v, ok := f.ConcreteAt(i); ok {
				switch {
				case isTime:
					t := v.(time.Time)
					stream.WriteVal(t.UnixMilli())
					msRes := t.Truncate(time.Millisecond)
					ns := t.Sub(msRes).Nanoseconds()
					if ns != 0 {
						hasNSTime = true
						nsTime[i] = ns
					}
				case isFloat:
					// For float and nullable float we check whether a value is a special
					// entity (NaN, -Inf, +Inf) not supported by JSON spec, we then encode this
					// information into a separate field to restore on a consumer side (setting
					// null to the entity position in data). Since we are using f.ConcreteAt
					// above the value is always float64 or float32 types, and never a *float64
					// or *float32.
					var f64 float64
					switch vt := v.(type) {
					case float64:
						f64 = vt
					case float32:
						f64 = float64(vt)
					default:
						stream.Error = fmt.Errorf("unsupported float type: %T", v)
						return
					}
					if entityType, found := isSpecialEntity(f64); found {
						if entities[fidx] == nil {
							entities[fidx] = &fieldEntityLookup{}
						}
						entities[fidx].add(entityType, i)
						entityCount++
						stream.WriteNil()
					} else {
						stream.WriteVal(v)
					}
				default:
					stream.WriteVal(v)
				}
			} else {
				stream.WriteNil()
			}
		}
		stream.WriteArrayEnd()
		if hasNSTime {
			nanos[fidx] = nsTime
			nsOffSetCount++
		}
	}
	stream.WriteArrayEnd()

	if entityCount > 0 {
		stream.WriteMore()
		stream.WriteObjectField("entities")
		stream.WriteVal(entities)
	}

	if nsOffSetCount > 0 {
		stream.WriteMore()
		stream.WriteObjectField("nanos")
		stream.WriteVal(nanos)
	}

	stream.WriteObjectEnd()
}

//nolint:gocyclo,gosec // test data generator, overflow conversions are intended
func randomFrame(rng *rand.Rand, rows int) *Frame {
	types := []FieldType{
		FieldTypeInt8, FieldTypeNullableInt8, FieldTypeInt16, FieldTypeInt32, FieldTypeNullableInt32,
		FieldTypeInt64, FieldTypeNullableInt64, FieldTypeUint8, FieldTypeUint64, FieldTypeNullableUint64,
		FieldTypeFloat32, FieldTypeNullableFloat32, FieldTypeFloat64, FieldTypeNullableFloat64,
		FieldTypeString, FieldTypeNullableString, FieldTypeBool, FieldTypeNullableBool,
		FieldTypeTime, FieldTypeNullableTime, FieldTypeJSON, FieldTypeNullableJSON,
	}
	specialFloats := []float64{math.NaN(), math.Inf(1), math.Inf(-1), 0, math.Copysign(0, -1), 1e21, 1e-7, math.MaxFloat64, math.SmallestNonzeroFloat64}
	frame := &Frame{Name: "diff"}
	nFields := 1 + rng.Intn(6)
	for range nFields {
		ft := types[rng.Intn(len(types))]
		f := NewFieldFromFieldType(ft, rows)
		f.Name = "f"
		for i := range rows {
			if ft.Nullable() && rng.Intn(5) == 0 {
				continue
			}
			var v any
			switch ft.NonNullableType() {
			case FieldTypeInt8:
				v = int8(rng.Intn(256) - 128)
			case FieldTypeInt16:
				v = int16(rng.Intn(65536) - 32768)
			case FieldTypeInt32:
				v = int32(rng.Uint32())
			case FieldTypeInt64:
				v = int64(rng.Uint64())
			case FieldTypeUint8:
				v = uint8(rng.Intn(256))
			case FieldTypeUint64:
				v = rng.Uint64()
			case FieldTypeFloat32:
				v = float32(rng.NormFloat64() * 1000)
				if rng.Intn(8) == 0 {
					v = float32(specialFloats[rng.Intn(len(specialFloats))])
				}
			case FieldTypeFloat64:
				v = rng.NormFloat64() * 1e6
				if rng.Intn(8) == 0 {
					v = specialFloats[rng.Intn(len(specialFloats))]
				}
			case FieldTypeString:
				v = []string{"", "a", "<b>&amp;</b>", "quote\"\\", "\u2028 unicode ❤️", "tab\t\n"}[rng.Intn(6)]
			case FieldTypeBool:
				v = rng.Intn(2) == 0
			case FieldTypeTime:
				v = time.Unix(rng.Int63n(4e9)-1e9, rng.Int63n(1e9)).UTC()
				if rng.Intn(2) == 0 {
					v = time.UnixMilli(rng.Int63n(4e12) - 1e12)
				}
			case FieldTypeJSON:
				v = json.RawMessage(`{"a":1}`)
			}
			if ft.Nullable() {
				switch x := v.(type) {
				case int8:
					v = &x
				case int16:
					v = &x
				case int32:
					v = &x
				case int64:
					v = &x
				case uint8:
					v = &x
				case uint64:
					v = &x
				case float32:
					v = &x
				case float64:
					v = &x
				case string:
					v = &x
				case bool:
					v = &x
				case time.Time:
					v = &x
				case json.RawMessage:
					v = &x
				}
			}
			f.Set(i, v)
		}
		frame.Fields = append(frame.Fields, f)
	}
	return frame
}

func TestWriteDataFrameDataMatchesReference(t *testing.T) {
	rng := rand.New(rand.NewSource(1)) //nolint:gosec // deterministic test data
	cfg := jsoniter.ConfigCompatibleWithStandardLibrary
	for _, rows := range []int{0, 1, 2, 17, 500} {
		for n := range 100 {
			frame := randomFrame(rng, rows)

			want := cfg.BorrowStream(nil)
			writeDataFrameDataReference(frame, want)
			got := cfg.BorrowStream(nil)
			writeDataFrameData(frame, got)

			if want.Error != nil || got.Error != nil {
				t.Fatalf("rows=%d iteration=%d: reference error %v, new error %v", rows, n, want.Error, got.Error)
			}
			if string(want.Buffer()) != string(got.Buffer()) {
				t.Fatalf("rows=%d iteration=%d: output differs\nwant %s\ngot  %s", rows, n, want.Buffer(), got.Buffer())
			}
			cfg.ReturnStream(want)
			cfg.ReturnStream(got)
		}
	}
}

func TestWriteDataFrameDataOtherConfigMatchesReference(t *testing.T) {
	rng := rand.New(rand.NewSource(2)) //nolint:gosec // deterministic test data
	cfg := jsoniter.Config{EscapeHTML: false, SortMapKeys: true}.Froze()
	for n := range 100 {
		frame := randomFrame(rng, 50)

		want := cfg.BorrowStream(nil)
		writeDataFrameDataReference(frame, want)
		got := cfg.BorrowStream(nil)
		writeDataFrameData(frame, got)

		if string(want.Buffer()) != string(got.Buffer()) {
			t.Fatalf("iteration=%d: output differs\nwant %s\ngot  %s", n, want.Buffer(), got.Buffer())
		}
		cfg.ReturnStream(want)
		cfg.ReturnStream(got)
	}
}

func TestWriteDataFrameDataTypedFieldAllocs(t *testing.T) {
	const rows = 1000
	ts := make([]time.Time, rows)
	vals := make([]float64, rows)
	for i := range ts {
		ts[i] = time.UnixMilli(int64(i) * 1000)
		vals[i] = float64(i) / 3
	}
	frame := NewFrame("allocs", NewField("time", nil, ts), NewField("value", nil, vals))
	cfg := jsoniter.ConfigCompatibleWithStandardLibrary
	stream := cfg.BorrowStream(nil)
	defer cfg.ReturnStream(stream)

	got := testing.AllocsPerRun(20, func() {
		stream.Reset(nil)
		writeDataFrameData(frame, stream)
	})
	if got > 12 {
		t.Errorf("writeDataFrameData allocated %v times per run for a 2 field frame, want at most 12", got)
	}
}
