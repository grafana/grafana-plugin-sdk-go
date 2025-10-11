package data

import (
	"encoding/json"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

func buildStringColumn(pool memory.Allocator, field arrow.Field, vec *genericVector[string]) *arrow.Column {
	builder := array.NewStringBuilder(pool)
	defer builder.Release()

	for _, v := range vec.Slice() {
		builder.Append(v)
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}

func buildNullableStringColumn(pool memory.Allocator, field arrow.Field, vec *nullableGenericVector[string]) *arrow.Column {
	builder := array.NewStringBuilder(pool)
	defer builder.Release()

	for _, v := range vec.Slice() {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}

func buildInt8Column(pool memory.Allocator, field arrow.Field, vec *genericVector[int8]) *arrow.Column {
	builder := array.NewInt8Builder(pool)
	defer builder.Release()

	for _, v := range vec.Slice() {
		builder.Append(v)
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}

func buildNullableInt8Column(pool memory.Allocator, field arrow.Field, vec *nullableGenericVector[int8]) *arrow.Column {
	builder := array.NewInt8Builder(pool)
	defer builder.Release()

	for _, v := range vec.Slice() {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}

func buildInt16Column(pool memory.Allocator, field arrow.Field, vec *genericVector[int16]) *arrow.Column {
	builder := array.NewInt16Builder(pool)
	defer builder.Release()

	for _, v := range vec.Slice() {
		builder.Append(v)
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}

func buildNullableInt16Column(pool memory.Allocator, field arrow.Field, vec *nullableGenericVector[int16]) *arrow.Column {
	builder := array.NewInt16Builder(pool)
	defer builder.Release()

	for _, v := range vec.Slice() {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}

func buildInt32Column(pool memory.Allocator, field arrow.Field, vec *genericVector[int32]) *arrow.Column {
	builder := array.NewInt32Builder(pool)
	defer builder.Release()

	for _, v := range vec.Slice() {
		builder.Append(v)
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}

func buildNullableInt32Column(pool memory.Allocator, field arrow.Field, vec *nullableGenericVector[int32]) *arrow.Column {
	builder := array.NewInt32Builder(pool)
	defer builder.Release()

	for _, v := range vec.Slice() {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}

func buildInt64Column(pool memory.Allocator, field arrow.Field, vec *genericVector[int64]) *arrow.Column {
	builder := array.NewInt64Builder(pool)
	defer builder.Release()

	for _, v := range vec.Slice() {
		builder.Append(v)
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}

func buildNullableInt64Column(pool memory.Allocator, field arrow.Field, vec *nullableGenericVector[int64]) *arrow.Column {
	builder := array.NewInt64Builder(pool)
	defer builder.Release()

	for _, v := range vec.Slice() {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}

func buildUInt8Column(pool memory.Allocator, field arrow.Field, vec *genericVector[uint8]) *arrow.Column {
	builder := array.NewUint8Builder(pool)
	defer builder.Release()

	for _, v := range vec.Slice() {
		builder.Append(v)
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}

func buildNullableUInt8Column(pool memory.Allocator, field arrow.Field, vec *nullableGenericVector[uint8]) *arrow.Column {
	builder := array.NewUint8Builder(pool)
	defer builder.Release()

	for _, v := range vec.Slice() {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}

func buildUInt16Column(pool memory.Allocator, field arrow.Field, vec *genericVector[uint16]) *arrow.Column {
	builder := array.NewUint16Builder(pool)
	defer builder.Release()

	for _, v := range vec.Slice() {
		builder.Append(v)
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}

func buildNullableUInt16Column(pool memory.Allocator, field arrow.Field, vec *nullableGenericVector[uint16]) *arrow.Column {
	builder := array.NewUint16Builder(pool)
	defer builder.Release()

	for _, v := range vec.Slice() {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}

func buildUInt32Column(pool memory.Allocator, field arrow.Field, vec *genericVector[uint32]) *arrow.Column {
	builder := array.NewUint32Builder(pool)
	defer builder.Release()

	for _, v := range vec.Slice() {
		builder.Append(v)
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}

func buildNullableUInt32Column(pool memory.Allocator, field arrow.Field, vec *nullableGenericVector[uint32]) *arrow.Column {
	builder := array.NewUint32Builder(pool)
	defer builder.Release()

	for _, v := range vec.Slice() {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}

func buildUInt64Column(pool memory.Allocator, field arrow.Field, vec *genericVector[uint64]) *arrow.Column {
	builder := array.NewUint64Builder(pool)
	defer builder.Release()

	for _, v := range vec.Slice() {
		builder.Append(v)
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}

func buildNullableUInt64Column(pool memory.Allocator, field arrow.Field, vec *nullableGenericVector[uint64]) *arrow.Column {
	builder := array.NewUint64Builder(pool)
	defer builder.Release()

	for _, v := range vec.Slice() {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}

func buildFloat32Column(pool memory.Allocator, field arrow.Field, vec *genericVector[float32]) *arrow.Column {
	builder := array.NewFloat32Builder(pool)
	defer builder.Release()

	for _, v := range vec.Slice() {
		builder.Append(v)
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}

func buildNullableFloat32Column(pool memory.Allocator, field arrow.Field, vec *nullableGenericVector[float32]) *arrow.Column {
	builder := array.NewFloat32Builder(pool)
	defer builder.Release()

	for _, v := range vec.Slice() {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}

func buildFloat64Column(pool memory.Allocator, field arrow.Field, vec *genericVector[float64]) *arrow.Column {
	builder := array.NewFloat64Builder(pool)
	defer builder.Release()

	for _, v := range vec.Slice() {
		builder.Append(v)
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}

func buildNullableFloat64Column(pool memory.Allocator, field arrow.Field, vec *nullableGenericVector[float64]) *arrow.Column {
	builder := array.NewFloat64Builder(pool)
	defer builder.Release()

	for _, v := range vec.Slice() {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}

func buildBoolColumn(pool memory.Allocator, field arrow.Field, vec *genericVector[bool]) *arrow.Column {
	builder := array.NewBooleanBuilder(pool)
	defer builder.Release()

	for _, v := range vec.Slice() {
		builder.Append(v)
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}

func buildNullableBoolColumn(pool memory.Allocator, field arrow.Field, vec *nullableGenericVector[bool]) *arrow.Column {
	builder := array.NewBooleanBuilder(pool)
	defer builder.Release()

	for _, v := range vec.Slice() {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}

func buildTimeColumnGeneric(pool memory.Allocator, field arrow.Field, vec *genericVector[time.Time]) *arrow.Column {
	builder := array.NewTimestampBuilder(pool, &arrow.TimestampType{
		Unit: arrow.Nanosecond,
	})
	defer builder.Release()

	for _, v := range vec.Slice() {
		builder.Append(arrow.Timestamp((v).UnixNano()))
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewTimestampArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}

func buildNullableTimeColumnGeneric(pool memory.Allocator, field arrow.Field, vec *nullableGenericVector[time.Time]) *arrow.Column {
	builder := array.NewTimestampBuilder(pool, &arrow.TimestampType{
		Unit: arrow.Nanosecond,
	})
	defer builder.Release()

	for _, v := range vec.Slice() {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(arrow.Timestamp(v.UnixNano()))
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}

func buildJSONColumnGeneric(pool memory.Allocator, field arrow.Field, vec *genericVector[json.RawMessage]) *arrow.Column {
	builder := array.NewBinaryBuilder(pool, &arrow.BinaryType{})
	defer builder.Release()

	for _, v := range vec.Slice() {
		builder.Append(v)
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}

func buildNullableJSONColumnGeneric(pool memory.Allocator, field arrow.Field, vec *nullableGenericVector[json.RawMessage]) *arrow.Column {
	builder := array.NewBinaryBuilder(pool, &arrow.BinaryType{})
	defer builder.Release()

	for _, v := range vec.Slice() {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}

func buildEnumColumnGeneric(pool memory.Allocator, field arrow.Field, vec *genericVector[EnumItemIndex]) *arrow.Column {
	builder := array.NewUint16Builder(pool)
	defer builder.Release()

	for _, v := range vec.Slice() {
		builder.Append(uint16(v))
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}

func buildNullableEnumColumnGeneric(pool memory.Allocator, field arrow.Field, vec *nullableGenericVector[EnumItemIndex]) *arrow.Column {
	builder := array.NewUint16Builder(pool)
	defer builder.Release()

	for _, v := range vec.Slice() {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(uint16(*v))
	}

	chunked := arrow.NewChunked(field.Type, []arrow.Array{builder.NewArray()})
	defer chunked.Release()

	return arrow.NewColumn(field, chunked)
}
