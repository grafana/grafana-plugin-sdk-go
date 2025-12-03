package data_test

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// Benchmark helpers to create frames of different sizes and complexities

func createSmallFrame() *data.Frame {
	return data.NewFrame("small",
		data.NewField("time", nil, []time.Time{
			time.Unix(1000, 0), time.Unix(2000, 0), time.Unix(3000, 0),
		}),
		data.NewField("value", nil, []float64{1.0, 2.0, 3.0}),
	)
}

func createMediumFrame(rows int) *data.Frame {
	times := make([]time.Time, rows)
	values := make([]float64, rows)
	strings := make([]string, rows)
	ints := make([]int64, rows)

	for i := 0; i < rows; i++ {
		times[i] = time.Unix(int64(i*1000), 0)
		values[i] = float64(i) * 1.5
		strings[i] = "value_" + string(rune(i%26+97))
		ints[i] = int64(i)
	}

	return data.NewFrame("medium",
		data.NewField("time", nil, times),
		data.NewField("value", nil, values),
		data.NewField("string", nil, strings),
		data.NewField("int", nil, ints),
	)
}

func createLargeComplexFrame(rows int) *data.Frame {
	times := make([]*time.Time, rows)
	float64s := make([]*float64, rows)
	float32s := make([]*float32, rows)
	strings := make([]*string, rows)
	int64s := make([]*int64, rows)
	int32s := make([]*int32, rows)
	int16s := make([]*int16, rows)
	int8s := make([]*int8, rows)
	uint64s := make([]*uint64, rows)
	uint32s := make([]*uint32, rows)
	uint16s := make([]*uint16, rows)
	uint8s := make([]*uint8, rows)
	bools := make([]*bool, rows)
	jsons := make([]*json.RawMessage, rows)

	for i := 0; i < rows; i++ {
		if i%10 != 0 { // 10% nulls
			t := time.Unix(int64(i*1000), 0)
			times[i] = &t

			f64 := float64(i) * 1.5
			float64s[i] = &f64

			f32 := float32(i) * 0.5
			float32s[i] = &f32

			s := "value_" + string(rune(i%26+97))
			strings[i] = &s

		i64 := int64(i)
		int64s[i] = &i64

		i32 := int32(i) // #nosec G115 -- benchmark code with controlled input
		int32s[i] = &i32

		i16 := int16(i % 32767) // #nosec G115 -- benchmark code with controlled input
		int16s[i] = &i16

		i8 := int8(i % 127) // #nosec G115 -- benchmark code with controlled input
		int8s[i] = &i8

		u64 := uint64(i) // #nosec G115 -- benchmark code with controlled input
		uint64s[i] = &u64

		u32 := uint32(i) // #nosec G115 -- benchmark code with controlled input
		uint32s[i] = &u32

		u16 := uint16(i % 65535) // #nosec G115 -- benchmark code with controlled input
		uint16s[i] = &u16

		u8 := uint8(i % 255) // #nosec G115 -- benchmark code with controlled input
		uint8s[i] = &u8

			b := i%2 == 0
			bools[i] = &b

			j := json.RawMessage(`{"id":` + string(rune(i%10+48)) + `}`)
			jsons[i] = &j
		}
	}

	frame := data.NewFrame("large_complex",
		data.NewField("time", data.Labels{"source": "benchmark"}, times),
		data.NewField("float64", nil, float64s),
		data.NewField("float32", nil, float32s),
		data.NewField("string", data.Labels{"type": "text"}, strings),
		data.NewField("int64", nil, int64s),
		data.NewField("int32", nil, int32s),
		data.NewField("int16", nil, int16s),
		data.NewField("int8", nil, int8s),
		data.NewField("uint64", nil, uint64s),
		data.NewField("uint32", nil, uint32s),
		data.NewField("uint16", nil, uint16s),
		data.NewField("uint8", nil, uint8s),
		data.NewField("bool", nil, bools),
		data.NewField("json", nil, jsons),
	)

	frame.SetMeta(&data.FrameMeta{
		ExecutedQueryString: "SELECT * FROM benchmarks",
		Custom:              map[string]interface{}{"benchmark": true},
	})

	// Add field configs to some fields
	frame.Fields[1].SetConfig((&data.FieldConfig{
		DisplayName: "Float64 Value",
		Unit:        "percent",
	}).SetMin(0.0).SetMax(float64(rows)))

	return frame
}

func createWideFrame(rows, cols int) *data.Frame {
	fields := make([]*data.Field, cols)
	for c := 0; c < cols; c++ {
		values := make([]float64, rows)
		for r := 0; r < rows; r++ {
			values[r] = float64(r*c) * 0.1
		}
		fields[c] = data.NewField("col_"+string(rune(c%26+97)), nil, values)
	}
	return data.NewFrame("wide", fields...)
}

func createNumericOnlyFrame(rows int) *data.Frame {
	int64s := make([]int64, rows)
	float64s := make([]float64, rows)
	uint64s := make([]uint64, rows)
	int32s := make([]int32, rows)

	for i := 0; i < rows; i++ {
		int64s[i] = int64(i)
		float64s[i] = float64(i) * math.Pi
		uint64s[i] = uint64(i * 2)      // #nosec G115 -- benchmark code with controlled input
		int32s[i] = int32(i % math.MaxInt32) // #nosec G115 -- benchmark code with controlled input
	}

	return data.NewFrame("numeric",
		data.NewField("int64", nil, int64s),
		data.NewField("float64", nil, float64s),
		data.NewField("uint64", nil, uint64s),
		data.NewField("int32", nil, int32s),
	)
}

func createTimeSeriesFrame(rows int) *data.Frame {
	times := make([]time.Time, rows)
	values := make([]float64, rows)
	start := time.Now()

	for i := 0; i < rows; i++ {
		times[i] = start.Add(time.Duration(i) * time.Second)
		values[i] = math.Sin(float64(i) * 0.1)
	}

	return data.NewFrame("timeseries",
		data.NewField("time", nil, times),
		data.NewField("value", nil, values),
	).SetMeta(&data.FrameMeta{
		Type: data.FrameTypeTimeSeriesMany,
	})
}

// Benchmarks for MarshalArrow

func BenchmarkMarshalArrow_Small(b *testing.B) {
	frame := createSmallFrame()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := frame.MarshalArrow()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshalArrow_Medium_100Rows(b *testing.B) {
	frame := createMediumFrame(100)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := frame.MarshalArrow()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshalArrow_Medium_1000Rows(b *testing.B) {
	frame := createMediumFrame(1000)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := frame.MarshalArrow()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshalArrow_Large_10000Rows(b *testing.B) {
	frame := createLargeComplexFrame(10000)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := frame.MarshalArrow()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshalArrow_Large_100000Rows(b *testing.B) {
	frame := createLargeComplexFrame(100000)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := frame.MarshalArrow()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshalArrow_Wide_100x100(b *testing.B) {
	frame := createWideFrame(100, 100)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := frame.MarshalArrow()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshalArrow_NumericOnly_10000Rows(b *testing.B) {
	frame := createNumericOnlyFrame(10000)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := frame.MarshalArrow()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshalArrow_TimeSeries_10000Rows(b *testing.B) {
	frame := createTimeSeriesFrame(10000)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := frame.MarshalArrow()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmarks for UnmarshalArrowFrame

func BenchmarkUnmarshalArrowFrame_Small(b *testing.B) {
	frame := createSmallFrame()
	encoded, err := frame.MarshalArrow()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := data.UnmarshalArrowFrame(encoded)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalArrowFrame_Medium_100Rows(b *testing.B) {
	frame := createMediumFrame(100)
	encoded, err := frame.MarshalArrow()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := data.UnmarshalArrowFrame(encoded)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalArrowFrame_Medium_1000Rows(b *testing.B) {
	frame := createMediumFrame(1000)
	encoded, err := frame.MarshalArrow()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := data.UnmarshalArrowFrame(encoded)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalArrowFrame_Large_10000Rows(b *testing.B) {
	frame := createLargeComplexFrame(10000)
	encoded, err := frame.MarshalArrow()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := data.UnmarshalArrowFrame(encoded)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalArrowFrame_Large_100000Rows(b *testing.B) {
	frame := createLargeComplexFrame(100000)
	encoded, err := frame.MarshalArrow()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := data.UnmarshalArrowFrame(encoded)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalArrowFrame_Wide_100x100(b *testing.B) {
	frame := createWideFrame(100, 100)
	encoded, err := frame.MarshalArrow()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := data.UnmarshalArrowFrame(encoded)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalArrowFrame_NumericOnly_10000Rows(b *testing.B) {
	frame := createNumericOnlyFrame(10000)
	encoded, err := frame.MarshalArrow()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := data.UnmarshalArrowFrame(encoded)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalArrowFrame_TimeSeries_10000Rows(b *testing.B) {
	frame := createTimeSeriesFrame(10000)
	encoded, err := frame.MarshalArrow()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := data.UnmarshalArrowFrame(encoded)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmarks for round-trip (Marshal + Unmarshal)

func BenchmarkArrowRoundTrip_Small(b *testing.B) {
	frame := createSmallFrame()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		encoded, err := frame.MarshalArrow()
		if err != nil {
			b.Fatal(err)
		}
		_, err = data.UnmarshalArrowFrame(encoded)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkArrowRoundTrip_Medium_1000Rows(b *testing.B) {
	frame := createMediumFrame(1000)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		encoded, err := frame.MarshalArrow()
		if err != nil {
			b.Fatal(err)
		}
		_, err = data.UnmarshalArrowFrame(encoded)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkArrowRoundTrip_Large_10000Rows(b *testing.B) {
	frame := createLargeComplexFrame(10000)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		encoded, err := frame.MarshalArrow()
		if err != nil {
			b.Fatal(err)
		}
		_, err = data.UnmarshalArrowFrame(encoded)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmarks for FrameToArrowTable

func BenchmarkFrameToArrowTable_Small(b *testing.B) {
	frame := createSmallFrame()
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		table, err := data.FrameToArrowTable(frame)
		if err != nil {
			b.Fatal(err)
		}
		table.Release()
	}
}

func BenchmarkFrameToArrowTable_Medium_1000Rows(b *testing.B) {
	frame := createMediumFrame(1000)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		table, err := data.FrameToArrowTable(frame)
		if err != nil {
			b.Fatal(err)
		}
		table.Release()
	}
}

func BenchmarkFrameToArrowTable_Large_10000Rows(b *testing.B) {
	frame := createLargeComplexFrame(10000)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		table, err := data.FrameToArrowTable(frame)
		if err != nil {
			b.Fatal(err)
		}
		table.Release()
	}
}

// Benchmarks for FromArrowRecord

func BenchmarkFromArrowRecord_Small(b *testing.B) {
	frame := createSmallFrame()
	encoded, err := frame.MarshalArrow()
	if err != nil {
		b.Fatal(err)
	}

	// Get a record by decoding the arrow data
	decodedFrame, err := data.UnmarshalArrowFrame(encoded)
	if err != nil {
		b.Fatal(err)
	}

	// Re-encode to get the arrow record for benchmarking
	table, err := data.FrameToArrowTable(decodedFrame)
	if err != nil {
		b.Fatal(err)
	}
	defer table.Release()

	// Create a record from the table
	tr := array.NewTableReader(table, -1)
	defer tr.Release()
	if !tr.Next() {
		b.Fatal("no records in table")
	}
	record := tr.RecordBatch()
	defer record.Release()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := data.FromArrowRecord(record)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFromArrowRecord_Medium_1000Rows(b *testing.B) {
	frame := createMediumFrame(1000)
	table, err := data.FrameToArrowTable(frame)
	if err != nil {
		b.Fatal(err)
	}
	defer table.Release()

	tr := array.NewTableReader(table, -1)
	defer tr.Release()
	if !tr.Next() {
		b.Fatal("no records in table")
	}
	record := tr.RecordBatch()
	defer record.Release()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := data.FromArrowRecord(record)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFromArrowRecord_Large_10000Rows(b *testing.B) {
	frame := createLargeComplexFrame(10000)
	table, err := data.FrameToArrowTable(frame)
	if err != nil {
		b.Fatal(err)
	}
	defer table.Release()

	tr := array.NewTableReader(table, -1)
	defer tr.Release()
	if !tr.Next() {
		b.Fatal("no records in table")
	}
	record := tr.RecordBatch()
	defer record.Release()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := data.FromArrowRecord(record)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmarks for multiple frames (Frames.MarshalArrow and UnmarshalArrowFrames)

func BenchmarkFramesMarshalArrow_5Frames_1000Rows(b *testing.B) {
	frames := make(data.Frames, 5)
	for i := 0; i < 5; i++ {
		frames[i] = createMediumFrame(1000)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := frames.MarshalArrow()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFramesMarshalArrow_10Frames_100Rows(b *testing.B) {
	frames := make(data.Frames, 10)
	for i := 0; i < 10; i++ {
		frames[i] = createMediumFrame(100)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := frames.MarshalArrow()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalArrowFrames_5Frames_1000Rows(b *testing.B) {
	frames := make(data.Frames, 5)
	for i := 0; i < 5; i++ {
		frames[i] = createMediumFrame(1000)
	}
	encoded, err := frames.MarshalArrow()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := data.UnmarshalArrowFrames(encoded)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalArrowFrames_10Frames_100Rows(b *testing.B) {
	frames := make(data.Frames, 10)
	for i := 0; i < 10; i++ {
		frames[i] = createMediumFrame(100)
	}
	encoded, err := frames.MarshalArrow()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := data.UnmarshalArrowFrames(encoded)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmarks for different data types

func BenchmarkMarshalArrow_StringHeavy_1000Rows(b *testing.B) {
	strings := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		strings[i] = "this is a longer string value for benchmarking purposes"
	}
	frame := data.NewFrame("strings",
		data.NewField("value", nil, strings),
	)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := frame.MarshalArrow()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalArrowFrame_StringHeavy_1000Rows(b *testing.B) {
	strings := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		strings[i] = "this is a longer string value for benchmarking purposes"
	}
	frame := data.NewFrame("strings",
		data.NewField("value", nil, strings),
	)
	encoded, err := frame.MarshalArrow()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := data.UnmarshalArrowFrame(encoded)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshalArrow_WithMetadataAndLabels_1000Rows(b *testing.B) {
	frame := createMediumFrame(1000)
	frame.SetMeta(&data.FrameMeta{
		ExecutedQueryString: "SELECT * FROM large_table WHERE condition = true",
		Custom: map[string]interface{}{
			"key1": "value1",
			"key2": 12345,
			"key3": true,
		},
		Stats: []data.QueryStat{
			{Value: 123.45, FieldConfig: data.FieldConfig{DisplayName: "stat1"}},
			{Value: 678.90, FieldConfig: data.FieldConfig{DisplayName: "stat2"}},
		},
	})

	for _, field := range frame.Fields {
		field.Labels = data.Labels{
			"source":      "benchmark",
			"environment": "test",
			"version":     "1.0.0",
		}
		field.SetConfig((&data.FieldConfig{
			DisplayName: "Display " + field.Name,
			Unit:        "units",
		}).SetMin(0.0).SetMax(1000.0))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := frame.MarshalArrow()
		if err != nil {
			b.Fatal(err)
		}
	}
}
