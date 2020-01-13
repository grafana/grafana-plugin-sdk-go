package dataframe

import (
	"github.com/apache/arrow/go/arrow"
	"github.com/apache/arrow/go/arrow/array"
	"github.com/apache/arrow/go/arrow/memory"
)

func buildStringColumn(pool memory.Allocator, field arrow.Field, vec *StringVector) *array.Column {
	builder := array.NewStringBuilder(pool)
	defer builder.Release()

	for _, v := range (*vec).items {
		builder.Append(v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableStringColumn(pool memory.Allocator, field arrow.Field, vec *nullableStringVector) *array.Column {
	builder := array.NewStringBuilder(pool)
	defer builder.Release()

	for _, v := range (*vec).items {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildInt8Column(pool memory.Allocator, field arrow.Field, vec *Int8Vector) *array.Column {
	builder := array.NewInt8Builder(pool)
	defer builder.Release()

	for _, v := range (*vec).items {
		builder.Append(v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableInt8Column(pool memory.Allocator, field arrow.Field, vec *nullableInt8Vector) *array.Column {
	builder := array.NewInt8Builder(pool)
	defer builder.Release()

	for _, v := range (*vec).items {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildInt16Column(pool memory.Allocator, field arrow.Field, vec *Int16Vector) *array.Column {
	builder := array.NewInt16Builder(pool)
	defer builder.Release()

	for _, v := range (*vec).items {
		builder.Append(v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableInt16Column(pool memory.Allocator, field arrow.Field, vec *nullableInt16Vector) *array.Column {
	builder := array.NewInt16Builder(pool)
	defer builder.Release()

	for _, v := range (*vec).items {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildInt32Column(pool memory.Allocator, field arrow.Field, vec *Int32Vector) *array.Column {
	builder := array.NewInt32Builder(pool)
	defer builder.Release()

	for _, v := range (*vec).items {
		builder.Append(v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableInt32Column(pool memory.Allocator, field arrow.Field, vec *nullableInt32Vector) *array.Column {
	builder := array.NewInt32Builder(pool)
	defer builder.Release()

	for _, v := range (*vec).items {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildInt64Column(pool memory.Allocator, field arrow.Field, vec *Int64Vector) *array.Column {
	builder := array.NewInt64Builder(pool)
	defer builder.Release()

	for _, v := range (*vec).items {
		builder.Append(v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableInt64Column(pool memory.Allocator, field arrow.Field, vec *nullableInt64Vector) *array.Column {
	builder := array.NewInt64Builder(pool)
	defer builder.Release()

	for _, v := range (*vec).items {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildUInt8Column(pool memory.Allocator, field arrow.Field, vec *Uint8Vector) *array.Column {
	builder := array.NewUint8Builder(pool)
	defer builder.Release()

	for _, v := range (*vec).items {
		builder.Append(v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableUInt8Column(pool memory.Allocator, field arrow.Field, vec *nullableUint8Vector) *array.Column {
	builder := array.NewUint8Builder(pool)
	defer builder.Release()

	for _, v := range (*vec).items {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildUInt16Column(pool memory.Allocator, field arrow.Field, vec *Uint16Vector) *array.Column {
	builder := array.NewUint16Builder(pool)
	defer builder.Release()

	for _, v := range (*vec).items {
		builder.Append(v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableUInt16Column(pool memory.Allocator, field arrow.Field, vec *nullableUint16Vector) *array.Column {
	builder := array.NewUint16Builder(pool)
	defer builder.Release()

	for _, v := range (*vec).items {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildUInt32Column(pool memory.Allocator, field arrow.Field, vec *Uint32Vector) *array.Column {
	builder := array.NewUint32Builder(pool)
	defer builder.Release()

	for _, v := range (*vec).items {
		builder.Append(v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableUInt32Column(pool memory.Allocator, field arrow.Field, vec *nullableUint32Vector) *array.Column {
	builder := array.NewUint32Builder(pool)
	defer builder.Release()

	for _, v := range (*vec).items {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildUInt64Column(pool memory.Allocator, field arrow.Field, vec *Uint64Vector) *array.Column {
	builder := array.NewUint64Builder(pool)
	defer builder.Release()

	for _, v := range (*vec).items {
		builder.Append(v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableUInt64Column(pool memory.Allocator, field arrow.Field, vec *nullableUint64Vector) *array.Column {
	builder := array.NewUint64Builder(pool)
	defer builder.Release()

	for _, v := range (*vec).items {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildFloat32Column(pool memory.Allocator, field arrow.Field, vec *Float32Vector) *array.Column {
	builder := array.NewFloat32Builder(pool)
	defer builder.Release()

	for _, v := range (*vec).items {
		builder.Append(v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableFloat32Column(pool memory.Allocator, field arrow.Field, vec *nullableFloat32Vector) *array.Column {
	builder := array.NewFloat32Builder(pool)
	defer builder.Release()

	for _, v := range (*vec).items {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildFloat64Column(pool memory.Allocator, field arrow.Field, vec *Float64Vector) *array.Column {
	builder := array.NewFloat64Builder(pool)
	defer builder.Release()

	for _, v := range (*vec).items {
		builder.Append(v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableFloat64Column(pool memory.Allocator, field arrow.Field, vec *nullableFloat64Vector) *array.Column {
	builder := array.NewFloat64Builder(pool)
	defer builder.Release()

	for _, v := range (*vec).items {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildBoolColumn(pool memory.Allocator, field arrow.Field, vec *BoolVector) *array.Column {
	builder := array.NewBooleanBuilder(pool)
	defer builder.Release()

	for _, v := range (*vec).items {
		builder.Append(v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableBoolColumn(pool memory.Allocator, field arrow.Field, vec *nullableBoolVector) *array.Column {
	builder := array.NewBooleanBuilder(pool)
	defer builder.Release()

	for _, v := range (*vec).items {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildTimeColumn(pool memory.Allocator, field arrow.Field, vec *TimeTimeVector) *array.Column {
	builder := array.NewTimestampBuilder(pool, &arrow.TimestampType{
		Unit: arrow.Nanosecond,
	})
	defer builder.Release()

	for _, v := range (*vec).items {
		builder.Append(arrow.Timestamp((v).UnixNano()))
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableTimeColumn(pool memory.Allocator, field arrow.Field, vec *nullableTimeTimeVector) *array.Column {
	builder := array.NewTimestampBuilder(pool, &arrow.TimestampType{
		Unit: arrow.Nanosecond,
	})
	defer builder.Release()

	for _, v := range (*vec).items {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(arrow.Timestamp((*v).UnixNano()))
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}
