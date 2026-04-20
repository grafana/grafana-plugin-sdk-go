package data_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func TestNewFieldGenericWithCapacity(t *testing.T) {
	t.Run("length is 0, capacity is respected, append works", func(t *testing.T) {
		f := data.NewFieldGenericWithCapacity[float64]("value", nil, 1000)
		require.Equal(t, 0, f.Len())
		require.Equal(t, data.FieldTypeFloat64, f.Type())

		for i := 0; i < 1000; i++ {
			data.AppendTyped(f, float64(i))
		}
		require.Equal(t, 1000, f.Len())
		require.Equal(t, 999.0, f.At(999))
	})

	t.Run("works for time.Time", func(t *testing.T) {
		f := data.NewFieldGenericWithCapacity[time.Time]("ts", nil, 10)
		require.Equal(t, data.FieldTypeTime, f.Type())
		now := time.Unix(1700000000, 0)
		for i := 0; i < 10; i++ {
			data.AppendTyped(f, now.Add(time.Duration(i)*time.Second))
		}
		require.Equal(t, 10, f.Len())
		require.Equal(t, now, f.At(0))
	})
}

func TestNewFieldGenericNullableWithCapacity(t *testing.T) {
	t.Run("length is 0, capacity is respected, append works", func(t *testing.T) {
		f := data.NewFieldGenericNullableWithCapacity[float64]("value", nil, 100)
		require.Equal(t, 0, f.Len())
		require.Equal(t, data.FieldTypeNullableFloat64, f.Type())

		v := 4.2
		f.Append(&v)
		f.Append((*float64)(nil))
		require.Equal(t, 2, f.Len())
		require.True(t, f.NilAt(1))
	})
}

func TestAtTyped(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		f := data.NewFieldGeneric[int64]("v", nil, []int64{10, 20, 30})
		require.Equal(t, int64(10), data.AtTyped[int64](f, 0))
		require.Equal(t, int64(30), data.AtTyped[int64](f, 2))
	})
	t.Run("panics on wrong type parameter", func(t *testing.T) {
		f := data.NewFieldGeneric[int64]("v", nil, []int64{1})
		require.PanicsWithValue(t, "Field is not backed by genericVector[T]", func() {
			_ = data.AtTyped[int32](f, 0)
		})
	})
	t.Run("panics when backing vector is nullable", func(t *testing.T) {
		v := int64(1)
		f := data.NewFieldGenericNullable[int64]("v", nil, []*int64{&v})
		require.PanicsWithValue(t, "Field is not backed by genericVector[T]", func() {
			_ = data.AtTyped[int64](f, 0)
		})
	})
}

func TestSetTyped(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		f := data.NewFieldGeneric[int64]("v", nil, []int64{0, 0, 0})
		data.SetTyped[int64](f, 1, 99)
		require.Equal(t, int64(99), data.AtTyped[int64](f, 1))
	})
	t.Run("panics on wrong type parameter", func(t *testing.T) {
		f := data.NewFieldGeneric[int64]("v", nil, []int64{0})
		require.PanicsWithValue(t, "Field is not backed by genericVector[T]", func() {
			data.SetTyped[int32](f, 0, 1)
		})
	})
	t.Run("panics when backing vector is nullable", func(t *testing.T) {
		v := int64(1)
		f := data.NewFieldGenericNullable[int64]("v", nil, []*int64{&v})
		require.PanicsWithValue(t, "Field is not backed by genericVector[T]", func() {
			data.SetTyped[int64](f, 0, 2)
		})
	})
}

func TestAppendTyped(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		f := data.NewFieldGenericWithCapacity[int64]("v", nil, 2)
		data.AppendTyped[int64](f, 7)
		require.Equal(t, 1, f.Len())
		require.Equal(t, int64(7), data.AtTyped[int64](f, 0))
	})
	t.Run("panics on wrong type parameter", func(t *testing.T) {
		f := data.NewFieldGeneric[int64]("v", nil, []int64{})
		require.PanicsWithValue(t, "Field is not backed by genericVector[T]", func() {
			data.AppendTyped[int32](f, 1)
		})
	})
	t.Run("panics when backing vector is nullable", func(t *testing.T) {
		f := data.NewFieldGenericNullable[int64]("v", nil, []*int64{})
		require.PanicsWithValue(t, "Field is not backed by genericVector[T]", func() {
			data.AppendTyped[int64](f, 1)
		})
	})
}

func TestAtTypedNullable(t *testing.T) {
	t.Run("happy path — non-nil and nil", func(t *testing.T) {
		v := int64(5)
		f := data.NewFieldGenericNullable[int64]("v", nil, []*int64{&v, nil})
		got := data.AtTypedNullable[int64](f, 0)
		require.NotNil(t, got)
		require.Equal(t, int64(5), *got)
		require.Nil(t, data.AtTypedNullable[int64](f, 1))
	})
	t.Run("panics on wrong type parameter", func(t *testing.T) {
		v := int64(1)
		f := data.NewFieldGenericNullable[int64]("v", nil, []*int64{&v})
		require.PanicsWithValue(t, "Field is not backed by nullableGenericVector[T]", func() {
			_ = data.AtTypedNullable[int32](f, 0)
		})
	})
	t.Run("panics when backing vector is non-nullable", func(t *testing.T) {
		f := data.NewFieldGeneric[int64]("v", nil, []int64{1})
		require.PanicsWithValue(t, "Field is not backed by nullableGenericVector[T]", func() {
			_ = data.AtTypedNullable[int64](f, 0)
		})
	})
}

func TestSetTypedNullable(t *testing.T) {
	t.Run("happy path — set non-nil then nil", func(t *testing.T) {
		f := data.NewFieldGenericNullable[int64]("v", nil, []*int64{nil, nil})
		v := int64(42)
		data.SetTypedNullable[int64](f, 0, &v)
		data.SetTypedNullable[int64](f, 1, nil)
		require.Equal(t, int64(42), *data.AtTypedNullable[int64](f, 0))
		require.Nil(t, data.AtTypedNullable[int64](f, 1))
	})
	t.Run("panics on wrong type parameter", func(t *testing.T) {
		v := int64(1)
		f := data.NewFieldGenericNullable[int64]("v", nil, []*int64{&v})
		require.PanicsWithValue(t, "Field is not backed by nullableGenericVector[T]", func() {
			var x int32 = 1
			data.SetTypedNullable[int32](f, 0, &x)
		})
	})
	t.Run("panics when backing vector is non-nullable", func(t *testing.T) {
		f := data.NewFieldGeneric[int64]("v", nil, []int64{1})
		v := int64(2)
		require.PanicsWithValue(t, "Field is not backed by nullableGenericVector[T]", func() {
			data.SetTypedNullable[int64](f, 0, &v)
		})
	})
}

func TestConcreteAtTyped(t *testing.T) {
	t.Run("non-nullable returns (val, true)", func(t *testing.T) {
		f := data.NewFieldGeneric[int64]("v", nil, []int64{7})
		v, ok := data.ConcreteAtTyped[int64](f, 0)
		require.True(t, ok)
		require.Equal(t, int64(7), v)
	})
	t.Run("nullable non-nil returns (val, true)", func(t *testing.T) {
		v := int64(9)
		f := data.NewFieldGenericNullable[int64]("v", nil, []*int64{&v})
		got, ok := data.ConcreteAtTyped[int64](f, 0)
		require.True(t, ok)
		require.Equal(t, int64(9), got)
	})
	t.Run("nullable nil returns (zero, false)", func(t *testing.T) {
		f := data.NewFieldGenericNullable[int64]("v", nil, []*int64{nil})
		got, ok := data.ConcreteAtTyped[int64](f, 0)
		require.False(t, ok)
		require.Equal(t, int64(0), got)
	})
	t.Run("panics on wrong type parameter", func(t *testing.T) {
		f := data.NewFieldGeneric[int64]("v", nil, []int64{1})
		require.PanicsWithValue(t, "Field is not backed by genericVector[T] or nullableGenericVector[T]", func() {
			_, _ = data.ConcreteAtTyped[int32](f, 0)
		})
	})
}

func TestAppendTypedNullable(t *testing.T) {
	t.Run("appends pointers and nils", func(t *testing.T) {
		f := data.NewFieldGenericNullableWithCapacity[int64]("v", nil, 0)
		v := int64(42)
		data.AppendTypedNullable(f, &v)
		data.AppendTypedNullable[int64](f, nil)
		require.Equal(t, 2, f.Len())
		require.Equal(t, int64(42), *data.AtTypedNullable[int64](f, 0))
		require.Nil(t, data.AtTypedNullable[int64](f, 1))
	})
	t.Run("panics when field is not nullable", func(t *testing.T) {
		f := data.NewFieldGeneric[int64]("v", nil, []int64{1})
		v := int64(2)
		require.PanicsWithValue(t, "Field is not backed by nullableGenericVector[T]", func() {
			data.AppendTypedNullable(f, &v)
		})
	})
	t.Run("panics on wrong element type", func(t *testing.T) {
		f := data.NewFieldGenericNullableWithCapacity[int64]("v", nil, 0)
		v := int32(1)
		require.PanicsWithValue(t, "Field is not backed by nullableGenericVector[T]", func() {
			data.AppendTypedNullable(f, &v)
		})
	})
}

func TestFieldAs(t *testing.T) {
	t.Run("binds type once and routes through typed methods", func(t *testing.T) {
		f := data.NewFieldGeneric[int64]("v", nil, []int64{10, 20, 30})
		tf, ok := data.FieldAs[int64](f)
		require.True(t, ok)
		require.Same(t, f, tf.Field())
		require.Equal(t, 3, tf.Len())
		require.Equal(t, int64(10), tf.At(0))
		tf.Set(1, 99)
		require.Equal(t, int64(99), tf.At(1))
		tf.Append(40)
		tf.AppendMany([]int64{50, 60})
		require.Equal(t, 6, tf.Len())
		require.Equal(t, []int64{10, 99, 30, 40, 50, 60}, tf.Slice())
	})
	t.Run("returns false for wrong element type", func(t *testing.T) {
		f := data.NewFieldGeneric[int64]("v", nil, []int64{1})
		tf, ok := data.FieldAs[int32](f)
		require.False(t, ok)
		require.Nil(t, tf)
	})
	t.Run("returns false for nullable field", func(t *testing.T) {
		f := data.NewFieldGenericNullableWithCapacity[int64]("v", nil, 0)
		tf, ok := data.FieldAs[int64](f)
		require.False(t, ok)
		require.Nil(t, tf)
	})
}

func TestNullableFieldAs(t *testing.T) {
	t.Run("binds type once and routes through typed methods", func(t *testing.T) {
		f := data.NewFieldGenericNullableWithCapacity[int64]("v", nil, 0)
		tf, ok := data.NullableFieldAs[int64](f)
		require.True(t, ok)
		require.Same(t, f, tf.Field())

		v := int64(7)
		tf.Append(&v)
		tf.Append(nil)
		tf.SetConcrete(1, 8)
		require.Equal(t, 2, tf.Len())

		got, present := tf.ConcreteAt(0)
		require.True(t, present)
		require.Equal(t, int64(7), got)
		got, present = tf.ConcreteAt(1)
		require.True(t, present)
		require.Equal(t, int64(8), got)

		a := int64(1)
		b := int64(2)
		tf.AppendMany([]*int64{&a, nil, &b})
		require.Equal(t, 5, tf.Len())
		require.Equal(t, int64(2), *tf.At(4))
	})
	t.Run("returns false for non-nullable field", func(t *testing.T) {
		f := data.NewFieldGeneric[int64]("v", nil, []int64{1})
		tf, ok := data.NullableFieldAs[int64](f)
		require.False(t, ok)
		require.Nil(t, tf)
	})
	t.Run("returns false for wrong element type", func(t *testing.T) {
		f := data.NewFieldGenericNullableWithCapacity[int64]("v", nil, 0)
		tf, ok := data.NullableFieldAs[int32](f)
		require.False(t, ok)
		require.Nil(t, tf)
	})
}

// BenchmarkFieldConstruction_Appended compares unsized-field + Append vs
// pre-sized field + AppendTyped for a realistic Prometheus-matrix-sized series.
// Simulates the pattern profiled in the Prometheus datasource hot path.
func BenchmarkFieldConstruction_Unsized_Append(b *testing.B) {
	const n = 1000
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Mirrors NewField("value", nil, []float64{}) + repeated Append calls
		f := data.NewField("value", nil, []float64{})
		for j := 0; j < n; j++ {
			f.Append(float64(j))
		}
		if f.Len() != n {
			b.Fatal("len")
		}
	}
}

func BenchmarkFieldConstruction_Presized_AppendTyped(b *testing.B) {
	const n = 1000
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		f := data.NewFieldGenericWithCapacity[float64]("value", nil, n)
		for j := 0; j < n; j++ {
			data.AppendTyped(f, float64(j))
		}
		if f.Len() != n {
			b.Fatal("len")
		}
	}
}

func BenchmarkFieldConstruction_Unsized_Time_Append(b *testing.B) {
	const n = 1000
	b.ReportAllocs()
	now := time.Unix(1700000000, 0)
	for i := 0; i < b.N; i++ {
		f := data.NewField("time", nil, []time.Time{})
		for j := 0; j < n; j++ {
			f.Append(now.Add(time.Duration(j) * time.Second))
		}
		if f.Len() != n {
			b.Fatal("len")
		}
	}
}

func BenchmarkFieldConstruction_Presized_Time_AppendTyped(b *testing.B) {
	const n = 1000
	b.ReportAllocs()
	now := time.Unix(1700000000, 0)
	for i := 0; i < b.N; i++ {
		f := data.NewFieldGenericWithCapacity[time.Time]("time", nil, n)
		for j := 0; j < n; j++ {
			data.AppendTyped(f, now.Add(time.Duration(j)*time.Second))
		}
		if f.Len() != n {
			b.Fatal("len")
		}
	}
}

// BenchmarkAppendPaths compares the three ways to append into a pre-sized *Field
// on the hot path:
//
//	interface — f.Append(v) through the vector interface, per-call boxing + assertion
//	typed_fn  — data.AppendTyped(f, v) free function, per-call type assertion
//	typed_tf  — tf.Append(v) via *TypedField[T], assertion amortized at construction
//
// The test isolates only the append loop; Field construction is excluded via
// b.StopTimer so the number reflects per-row append cost for a realistic column.
func BenchmarkAppendPaths(b *testing.B) {
	const n = 10000

	b.Run("float64/interface", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			f := data.NewFieldGenericWithCapacity[float64]("v", nil, n)
			b.StartTimer()
			for j := 0; j < n; j++ {
				f.Append(float64(j))
			}
		}
	})
	b.Run("float64/typed_fn", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			f := data.NewFieldGenericWithCapacity[float64]("v", nil, n)
			b.StartTimer()
			for j := 0; j < n; j++ {
				data.AppendTyped(f, float64(j))
			}
		}
	})
	b.Run("float64/typed_tf", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			f := data.NewFieldGenericWithCapacity[float64]("v", nil, n)
			tf, _ := data.FieldAs[float64](f)
			b.StartTimer()
			for j := 0; j < n; j++ {
				tf.Append(float64(j))
			}
		}
	})

	b.Run("time.Time/interface", func(b *testing.B) {
		b.ReportAllocs()
		now := time.Unix(1700000000, 0)
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			f := data.NewFieldGenericWithCapacity[time.Time]("v", nil, n)
			b.StartTimer()
			for j := 0; j < n; j++ {
				f.Append(now.Add(time.Duration(j) * time.Second))
			}
		}
	})
	b.Run("time.Time/typed_fn", func(b *testing.B) {
		b.ReportAllocs()
		now := time.Unix(1700000000, 0)
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			f := data.NewFieldGenericWithCapacity[time.Time]("v", nil, n)
			b.StartTimer()
			for j := 0; j < n; j++ {
				data.AppendTyped(f, now.Add(time.Duration(j)*time.Second))
			}
		}
	})
	b.Run("time.Time/typed_tf", func(b *testing.B) {
		b.ReportAllocs()
		now := time.Unix(1700000000, 0)
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			f := data.NewFieldGenericWithCapacity[time.Time]("v", nil, n)
			tf, _ := data.FieldAs[time.Time](f)
			b.StartTimer()
			for j := 0; j < n; j++ {
				tf.Append(now.Add(time.Duration(j) * time.Second))
			}
		}
	})

	b.Run("nullable_float64/typed_fn", func(b *testing.B) {
		b.ReportAllocs()
		v := 1.5
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			f := data.NewFieldGenericNullableWithCapacity[float64]("v", nil, n)
			b.StartTimer()
			for j := 0; j < n; j++ {
				data.AppendTypedNullable(f, &v)
			}
		}
	})
	b.Run("nullable_float64/typed_tf", func(b *testing.B) {
		b.ReportAllocs()
		v := 1.5
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			f := data.NewFieldGenericNullableWithCapacity[float64]("v", nil, n)
			tf, _ := data.NullableFieldAs[float64](f)
			b.StartTimer()
			for j := 0; j < n; j++ {
				tf.Append(&v)
			}
		}
	})
}
