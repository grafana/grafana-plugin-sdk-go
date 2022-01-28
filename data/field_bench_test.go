package data_test

import (
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

const LENGTH = 100_000

func BenchmarkNewFieldNullableFloat(b *testing.B) {
	b.ReportAllocs()
	b.Run("NewField *float64 make() known length", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			vals := make([]*float64, LENGTH)
			for i := float64(0); i < LENGTH; i++ {
				vals[int(i)] = &i
			}
			_ = data.NewField("Test", data.Labels{}, vals)
		}
	})

	b.Run("NewField *float64 pre-fill array", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			vals := []*float64{}
			for i := float64(0); i < LENGTH; i++ {
				vals = append(vals, &i)
			}
			_ = data.NewField("Test", data.Labels{}, vals)
		}
	})

	b.Run("NewField *float64 Append", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			f := data.NewField("Test", data.Labels{}, make([]*float64, 0))
			for i := float64(0); i < LENGTH; i++ {
				f.Append(&i)
			}
		}
	})
}

func BenchmarkNewFieldFromTypeNullableFloat(b *testing.B) {
	b.Run("NewFieldFromFieldType *float64 Append", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			f := data.NewFieldFromFieldType(data.FieldTypeNullableFloat64, 0)
			for i := float64(0); i < LENGTH; i++ {
				f.Append(&i)
			}
		}
	})

	b.Run("NewFieldFromFieldType *float64 Set", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			f := data.NewFieldFromFieldType(data.FieldTypeNullableFloat64, LENGTH)
			for i := float64(0); i < LENGTH; i++ {
				f.Set(int(i), &i)
			}
		}
	})
}

func BenchmarkNewFieldFloat(b *testing.B) {
	b.Run("NewField float64 make() known length", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			vals := make([]float64, LENGTH)
			for i := float64(0); i < LENGTH; i++ {
				vals[int(i)] = i
			}
			_ = data.NewField("Test", data.Labels{}, vals)
		}
	})

	b.Run("NewField float64 pre-fill array", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			vals := []float64{}
			for i := float64(0); i < LENGTH; i++ {
				vals = append(vals, i)
			}
			_ = data.NewField("Test", data.Labels{}, vals)
		}
	})

	b.Run("NewField float64 Append", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			f := data.NewField("Test", data.Labels{}, make([]float64, 0))
			for i := float64(0); i < LENGTH; i++ {
				f.Append(i)
			}
		}
	})
}

func BenchmarkNewFieldFromeTypeFloat(b *testing.B) {
	b.Run("NewFieldFromFieldType float64 Append", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			f := data.NewFieldFromFieldType(data.FieldTypeFloat64, 0)
			for i := float64(0); i < LENGTH; i++ {
				f.Append(i)
			}
		}
	})

	b.Run("NewFieldFromFieldType float64 Set", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			f := data.NewFieldFromFieldType(data.FieldTypeFloat64, LENGTH)
			for i := float64(0); i < LENGTH; i++ {
				f.Set(int(i), i)
			}
		}
	})
}

func BenchmarkNewFieldTime(b *testing.B) {
	b.Run("NewField time.Time make() known length", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			vals := make([]time.Time, LENGTH)
			for i := float64(0); i < LENGTH; i++ {
				vals[int(i)] = time.Now()
			}
			_ = data.NewField("Test", data.Labels{}, vals)
		}
	})

	b.Run("NewField time.Time pre-fill array", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			vals := []time.Time{}
			for i := float64(0); i < LENGTH; i++ {
				vals = append(vals, time.Now())
			}
			_ = data.NewField("Test", data.Labels{}, vals)
		}
	})

	b.Run("NewField time.Time Append", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			f := data.NewField("Test", data.Labels{}, make([]time.Time, 0))
			for i := float64(0); i < LENGTH; i++ {
				f.Append(time.Now())
			}
		}
	})
}

func BenchmarkNewFieldFromeTypeTime(b *testing.B) {
	b.Run("NewFieldFromFieldType time.Time Append", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			f := data.NewFieldFromFieldType(data.FieldTypeTime, 0)
			for i := float64(0); i < LENGTH; i++ {
				f.Append(time.Now())
			}
		}
	})

	b.Run("NewFieldFromFieldType time.Time Set", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			f := data.NewFieldFromFieldType(data.FieldTypeTime, LENGTH)
			for i := float64(0); i < LENGTH; i++ {
				f.Set(int(i), time.Now())
			}
		}
	})
}
