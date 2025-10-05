package data_test

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// BenchmarkFrameToJSON benchmarks the main public API function with different include options
func BenchmarkFrameToJSON_IncludeAll(b *testing.B) {
	f := goldenDF()
	warm, err := data.FrameToJSON(f, data.IncludeAll)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(warm)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := data.FrameToJSON(f, data.IncludeAll)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFrameToJSON_SchemaOnly(b *testing.B) {
	f := goldenDF()
	warm, err := data.FrameToJSON(f, data.IncludeSchemaOnly)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(warm)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := data.FrameToJSON(f, data.IncludeSchemaOnly)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFrameToJSON_DataOnly(b *testing.B) {
	f := goldenDF()
	warm, err := data.FrameToJSON(f, data.IncludeDataOnly)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(warm)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := data.FrameToJSON(f, data.IncludeDataOnly)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFrameJSONCache_Create benchmarks the cache creation (existing benchmark was testing this)
func BenchmarkFrameJSONCache_Create(b *testing.B) {
	f := goldenDF()
	warm, err := data.FrameToJSONCache(f)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(warm.Bytes(data.IncludeAll))))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := data.FrameToJSONCache(f)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFrameJSONCache_Bytes benchmarks the actual intended usage pattern
func BenchmarkFrameJSONCache_Bytes(b *testing.B) {
	f := goldenDF()
	cache, err := data.FrameToJSONCache(f)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("IncludeAll", func(b *testing.B) {
		result := cache.Bytes(data.IncludeAll)
		b.SetBytes(int64(len(result)))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = cache.Bytes(data.IncludeAll)
		}
	})

	b.Run("SchemaOnly", func(b *testing.B) {
		result := cache.Bytes(data.IncludeSchemaOnly)
		b.SetBytes(int64(len(result)))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = cache.Bytes(data.IncludeSchemaOnly)
		}
	})

	b.Run("DataOnly", func(b *testing.B) {
		result := cache.Bytes(data.IncludeDataOnly)
		b.SetBytes(int64(len(result)))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = cache.Bytes(data.IncludeDataOnly)
		}
	})
}

// BenchmarkFrameUnmarshalJSON benchmarks deserialization - CRITICAL MISSING BENCHMARK
func BenchmarkFrameUnmarshalJSON(b *testing.B) {
	f := goldenDF()
	jsonData, err := json.Marshal(f)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(jsonData)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var frame data.Frame
		err := json.Unmarshal(jsonData, &frame)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFrameUnmarshalJSON_FromFrameToJSON benchmarks unmarshal from FrameToJSON output
func BenchmarkFrameUnmarshalJSON_FromFrameToJSON(b *testing.B) {
	f := goldenDF()
	jsonData, err := data.FrameToJSON(f, data.IncludeAll)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(jsonData)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var frame data.Frame
		err := json.Unmarshal(jsonData, &frame)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFrameMarshalJSON_Sizes benchmarks different frame sizes to see scaling behavior
func BenchmarkFrameMarshalJSON_Sizes(b *testing.B) {
	sizes := []int{10, 100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Rows_%d", size), func(b *testing.B) {
			f := data.NewFrame("test",
				data.NewField("time", nil, makeTimeSlice(size)),
				data.NewField("value", nil, makeFloat64Slice(size)),
				data.NewField("name", nil, makeStringSlice(size)),
			)

			warm, _ := json.Marshal(f)
			b.SetBytes(int64(len(warm)))
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := json.Marshal(f)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkFrameUnmarshalJSON_Sizes benchmarks deserialization at different scales
func BenchmarkFrameUnmarshalJSON_Sizes(b *testing.B) {
	sizes := []int{10, 100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Rows_%d", size), func(b *testing.B) {
			f := data.NewFrame("test",
				data.NewField("time", nil, makeTimeSlice(size)),
				data.NewField("value", nil, makeFloat64Slice(size)),
				data.NewField("name", nil, makeStringSlice(size)),
			)

			jsonData, err := json.Marshal(f)
			if err != nil {
				b.Fatal(err)
			}
			b.SetBytes(int64(len(jsonData)))
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				var frame data.Frame
				err := json.Unmarshal(jsonData, &frame)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkFrameMarshalJSON_FieldTypes benchmarks specific field types to isolate optimized paths
// nolint:gocyclo
func BenchmarkFrameMarshalJSON_FieldTypes(b *testing.B) {
	size := 1000

	b.Run("TimeNoNanos", func(b *testing.B) {
		times := make([]time.Time, size)
		for i := range times {
			times[i] = time.Unix(int64(i), 0) // No nanosecond precision
		}
		f := data.NewFrame("test", data.NewField("time", nil, times))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(f)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("TimeWithNanos", func(b *testing.B) {
		times := make([]time.Time, size)
		for i := range times {
			times[i] = time.Unix(int64(i), int64((i%1000)*1000)) // Has nanosecond precision
		}
		f := data.NewFrame("test", data.NewField("time", nil, times))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(f)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("NullableTime", func(b *testing.B) {
		times := make([]*time.Time, size)
		for i := range times {
			if i%10 != 0 { // 10% null values
				t := time.Unix(int64(i), 0)
				times[i] = &t
			}
		}
		f := data.NewFrame("test", data.NewField("time", nil, times))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(f)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Float64Clean", func(b *testing.B) {
		values := make([]float64, size)
		for i := range values {
			values[i] = float64(i) * 1.5
		}
		f := data.NewFrame("test", data.NewField("value", nil, values))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(f)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Float64WithSpecials", func(b *testing.B) {
		values := make([]float64, size)
		for i := range values {
			switch i % 10 {
			case 0:
				values[i] = math.NaN()
			case 1:
				values[i] = math.Inf(1)
			case 2:
				values[i] = math.Inf(-1)
			default:
				values[i] = float64(i) * 1.5
			}
		}
		f := data.NewFrame("test", data.NewField("value", nil, values))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(f)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Float32WithSpecials", func(b *testing.B) {
		values := make([]float32, size)
		for i := range values {
			switch i % 10 {
			case 0:
				values[i] = float32(math.NaN())
			case 1:
				values[i] = float32(math.Inf(1))
			case 2:
				values[i] = float32(math.Inf(-1))
			default:
				values[i] = float32(i) * 1.5
			}
		}
		f := data.NewFrame("test", data.NewField("value", nil, values))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(f)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Int64", func(b *testing.B) {
		values := make([]int64, size)
		for i := range values {
			values[i] = int64(i)
		}
		f := data.NewFrame("test", data.NewField("value", nil, values))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(f)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("NullableInt64", func(b *testing.B) {
		values := make([]*int64, size)
		for i := range values {
			if i%10 != 0 { // 10% null values
				v := int64(i)
				values[i] = &v
			}
		}
		f := data.NewFrame("test", data.NewField("value", nil, values))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(f)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Uint64", func(b *testing.B) {
		values := make([]uint64, size)
		for i := range values {
			values[i] = uint64(i)
		}
		f := data.NewFrame("test", data.NewField("value", nil, values))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(f)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("NullableUint64", func(b *testing.B) {
		values := make([]*uint64, size)
		for i := range values {
			if i%10 != 0 { // 10% null values
				v := uint64(i)
				values[i] = &v
			}
		}
		f := data.NewFrame("test", data.NewField("value", nil, values))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(f)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("String", func(b *testing.B) {
		values := make([]string, size)
		for i := range values {
			values[i] = fmt.Sprintf("value_%d", i)
		}
		f := data.NewFrame("test", data.NewField("value", nil, values))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(f)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("NullableString", func(b *testing.B) {
		values := make([]*string, size)
		for i := range values {
			if i%10 != 0 { // 10% null values
				v := fmt.Sprintf("value_%d", i)
				values[i] = &v
			}
		}
		f := data.NewFrame("test", data.NewField("value", nil, values))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(f)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Bool", func(b *testing.B) {
		values := make([]bool, size)
		for i := range values {
			values[i] = i%2 == 0
		}
		f := data.NewFrame("test", data.NewField("value", nil, values))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(f)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("JSON", func(b *testing.B) {
		values := make([]json.RawMessage, size)
		for i := range values {
			values[i] = json.RawMessage(fmt.Sprintf(`{"index":%d}`, i))
		}
		f := data.NewFrame("test", data.NewField("value", nil, values))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(f)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Enum", func(b *testing.B) {
		values := make([]data.EnumItemIndex, size)
		for i := range values {
			values[i] = data.EnumItemIndex(i % 5)
		}
		f := data.NewFrame("test", data.NewField("value", nil, values))
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(f)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkFrameMarshalJSON_Parallel tests concurrent marshaling (pool contention)
func BenchmarkFrameMarshalJSON_Parallel(b *testing.B) {
	f := goldenDF()
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := json.Marshal(f)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkFrameMarshalJSON_ParallelLarge tests concurrent marshaling with larger frames
func BenchmarkFrameMarshalJSON_ParallelLarge(b *testing.B) {
	f := data.NewFrame("test",
		data.NewField("time", nil, makeTimeSlice(1000)),
		data.NewField("value", nil, makeFloat64Slice(1000)),
		data.NewField("name", nil, makeStringSlice(1000)),
	)
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := json.Marshal(f)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkFrameUnmarshalJSON_Parallel tests concurrent unmarshaling
func BenchmarkFrameUnmarshalJSON_Parallel(b *testing.B) {
	f := goldenDF()
	jsonData, err := json.Marshal(f)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var frame data.Frame
			err := json.Unmarshal(jsonData, &frame)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkFrameMarshalJSON_WithLabels tests frames with labels (map sorting)
func BenchmarkFrameMarshalJSON_WithLabels(b *testing.B) {
	size := 1000

	b.Run("NoLabels", func(b *testing.B) {
		f := data.NewFrame("test",
			data.NewField("time", nil, makeTimeSlice(size)),
			data.NewField("value", nil, makeFloat64Slice(size)),
		)
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(f)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("SmallLabels", func(b *testing.B) {
		labels := data.Labels{"job": "api", "instance": "server1"}
		f := data.NewFrame("test",
			data.NewField("time", labels, makeTimeSlice(size)),
			data.NewField("value", labels, makeFloat64Slice(size)),
		)
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(f)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ManyLabels", func(b *testing.B) {
		labels := data.Labels{
			"job":        "api",
			"instance":   "server1",
			"region":     "us-west",
			"datacenter": "dc1",
			"cluster":    "prod",
			"namespace":  "default",
			"pod":        "pod-123",
			"container":  "main",
		}
		f := data.NewFrame("test",
			data.NewField("time", labels, makeTimeSlice(size)),
			data.NewField("value", labels, makeFloat64Slice(size)),
		)
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(f)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkFrameMarshalJSON_WithMeta tests frames with metadata
func BenchmarkFrameMarshalJSON_WithMeta(b *testing.B) {
	size := 1000

	b.Run("NoMeta", func(b *testing.B) {
		f := data.NewFrame("test",
			data.NewField("time", nil, makeTimeSlice(size)),
			data.NewField("value", nil, makeFloat64Slice(size)),
		)
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(f)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("WithMeta", func(b *testing.B) {
		f := data.NewFrame("test",
			data.NewField("time", nil, makeTimeSlice(size)),
			data.NewField("value", nil, makeFloat64Slice(size)),
		)
		f.Meta = &data.FrameMeta{
			ExecutedQueryString: "SELECT * FROM table WHERE time > now() - 1h",
			Custom: map[string]interface{}{
				"key1": "value1",
				"key2": 123,
				"key3": true,
			},
		}
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(f)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkArrowToJSON benchmarks Arrow to JSON conversion
func BenchmarkArrowToJSON(b *testing.B) {
	f := goldenDF()
	arrowData, err := f.MarshalArrow()
	if err != nil {
		b.Fatal(err)
	}

	warm, err := data.ArrowBufferToJSON(arrowData, data.IncludeAll)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(warm)))
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := data.ArrowBufferToJSON(arrowData, data.IncludeAll)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFrameRoundtrip benchmarks complete marshal->unmarshal cycle
func BenchmarkFrameRoundtrip(b *testing.B) {
	f := goldenDF()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		jsonData, err := json.Marshal(f)
		if err != nil {
			b.Fatal(err)
		}
		var frame data.Frame
		err = json.Unmarshal(jsonData, &frame)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Helper functions for creating test data

func makeTimeSlice(n int) []time.Time {
	result := make([]time.Time, n)
	base := time.Unix(1600000000, 0)
	for i := range result {
		result[i] = base.Add(time.Duration(i) * time.Second)
	}
	return result
}

func makeFloat64Slice(n int) []float64 {
	result := make([]float64, n)
	for i := range result {
		result[i] = float64(i) * 1.5
	}
	return result
}

func makeStringSlice(n int) []string {
	result := make([]string, n)
	for i := range result {
		result[i] = fmt.Sprintf("value_%d", i)
	}
	return result
}
