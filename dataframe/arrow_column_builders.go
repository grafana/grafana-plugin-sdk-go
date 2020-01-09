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

func buildIntColumn(pool memory.Allocator, field arrow.Field, vec *Int64Vector) *array.Column {
	builder := array.NewInt64Builder(pool)
	defer builder.Release()

	for _, v := range (*vec).items {
		builder.Append(v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableIntColumn(pool memory.Allocator, field arrow.Field, vec *nullableInt64Vector) *array.Column {
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

func buildUIntColumn(pool memory.Allocator, field arrow.Field, vec *Uint64Vector) *array.Column {
	builder := array.NewUint64Builder(pool)
	defer builder.Release()

	for _, v := range (*vec).items {
		builder.Append(v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableUIntColumn(pool memory.Allocator, field arrow.Field, vec *nullableUint64Vector) *array.Column {
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

func buildFloatColumn(pool memory.Allocator, field arrow.Field, vec *Float64Vector) *array.Column {
	builder := array.NewFloat64Builder(pool)
	defer builder.Release()

	for _, v := range (*vec).items {
		builder.Append(v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableFloatColumn(pool memory.Allocator, field arrow.Field, vec *nullableFloat64Vector) *array.Column {
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
