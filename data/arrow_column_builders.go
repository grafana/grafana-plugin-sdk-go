package data

import (
	"encoding/json"
	"time"

	"github.com/apache/arrow/go/arrow"
	"github.com/apache/arrow/go/arrow/array"
	"github.com/apache/arrow/go/arrow/memory"
)

func buildStringColumn(pool memory.Allocator, field arrow.Field, vec *genericVector[string]) *array.Column {
	builder := array.NewStringBuilder(pool)
	defer builder.Release()

	for _, v := range *vec {
		builder.Append(v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableStringColumn(pool memory.Allocator, field arrow.Field, vec *genericVector[*string]) *array.Column {
	builder := array.NewStringBuilder(pool)
	defer builder.Release()

	for _, v := range *vec {
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

func buildInt8Column(pool memory.Allocator, field arrow.Field, vec *genericVector[int8]) *array.Column {
	builder := array.NewInt8Builder(pool)
	defer builder.Release()

	for _, v := range *vec {
		builder.Append(v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableInt8Column(pool memory.Allocator, field arrow.Field, vec *genericVector[*int8]) *array.Column {
	builder := array.NewInt8Builder(pool)
	defer builder.Release()

	for _, v := range *vec {
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

func buildInt16Column(pool memory.Allocator, field arrow.Field, vec *genericVector[int16]) *array.Column {
	builder := array.NewInt16Builder(pool)
	defer builder.Release()

	for _, v := range *vec {
		builder.Append(v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableInt16Column(pool memory.Allocator, field arrow.Field, vec *genericVector[*int16]) *array.Column {
	builder := array.NewInt16Builder(pool)
	defer builder.Release()

	for _, v := range *vec {
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

func buildInt32Column(pool memory.Allocator, field arrow.Field, vec *genericVector[int32]) *array.Column {
	builder := array.NewInt32Builder(pool)
	defer builder.Release()

	for _, v := range *vec {
		builder.Append(v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableInt32Column(pool memory.Allocator, field arrow.Field, vec *genericVector[*int32]) *array.Column {
	builder := array.NewInt32Builder(pool)
	defer builder.Release()

	for _, v := range *vec {
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

func buildInt64Column(pool memory.Allocator, field arrow.Field, vec *genericVector[int64]) *array.Column {
	builder := array.NewInt64Builder(pool)
	defer builder.Release()

	for _, v := range *vec {
		builder.Append(v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableInt64Column(pool memory.Allocator, field arrow.Field, vec *genericVector[*int64]) *array.Column {
	builder := array.NewInt64Builder(pool)
	defer builder.Release()

	for _, v := range *vec {
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

func buildUInt8Column(pool memory.Allocator, field arrow.Field, vec *genericVector[uint8]) *array.Column {
	builder := array.NewUint8Builder(pool)
	defer builder.Release()

	for _, v := range *vec {
		builder.Append(v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableUInt8Column(pool memory.Allocator, field arrow.Field, vec *genericVector[*uint8]) *array.Column {
	builder := array.NewUint8Builder(pool)
	defer builder.Release()

	for _, v := range *vec {
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

func buildUInt16Column(pool memory.Allocator, field arrow.Field, vec *genericVector[uint16]) *array.Column {
	builder := array.NewUint16Builder(pool)
	defer builder.Release()

	for _, v := range *vec {
		builder.Append(v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableUInt16Column(pool memory.Allocator, field arrow.Field, vec *genericVector[*uint16]) *array.Column {
	builder := array.NewUint16Builder(pool)
	defer builder.Release()

	for _, v := range *vec {
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

func buildUInt32Column(pool memory.Allocator, field arrow.Field, vec *genericVector[uint32]) *array.Column {
	builder := array.NewUint32Builder(pool)
	defer builder.Release()

	for _, v := range *vec {
		builder.Append(v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableUInt32Column(pool memory.Allocator, field arrow.Field, vec *genericVector[*uint32]) *array.Column {
	builder := array.NewUint32Builder(pool)
	defer builder.Release()

	for _, v := range *vec {
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

func buildUInt64Column(pool memory.Allocator, field arrow.Field, vec *genericVector[uint64]) *array.Column {
	builder := array.NewUint64Builder(pool)
	defer builder.Release()

	for _, v := range *vec {
		builder.Append(v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableUInt64Column(pool memory.Allocator, field arrow.Field, vec *genericVector[*uint64]) *array.Column {
	builder := array.NewUint64Builder(pool)
	defer builder.Release()

	for _, v := range *vec {
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

func buildFloat32Column(pool memory.Allocator, field arrow.Field, vec *genericVector[float32]) *array.Column {
	builder := array.NewFloat32Builder(pool)
	defer builder.Release()

	for _, v := range *vec {
		builder.Append(v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableFloat32Column(pool memory.Allocator, field arrow.Field, vec *genericVector[*float32]) *array.Column {
	builder := array.NewFloat32Builder(pool)
	defer builder.Release()

	for _, v := range *vec {
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

func buildFloat64Column(pool memory.Allocator, field arrow.Field, vec *genericVector[float64]) *array.Column {
	builder := array.NewFloat64Builder(pool)
	defer builder.Release()

	for _, v := range *vec {
		builder.Append(v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableFloat64Column(pool memory.Allocator, field arrow.Field, vec *genericVector[*float64]) *array.Column {
	builder := array.NewFloat64Builder(pool)
	defer builder.Release()

	for _, v := range *vec {
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

func buildBoolColumn(pool memory.Allocator, field arrow.Field, vec *genericVector[bool]) *array.Column {
	builder := array.NewBooleanBuilder(pool)
	defer builder.Release()

	for _, v := range *vec {
		builder.Append(v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableBoolColumn(pool memory.Allocator, field arrow.Field, vec *genericVector[*bool]) *array.Column {
	builder := array.NewBooleanBuilder(pool)
	defer builder.Release()

	for _, v := range *vec {
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

func buildTimeColumn(pool memory.Allocator, field arrow.Field, vec *genericVector[time.Time]) *array.Column {
	builder := array.NewTimestampBuilder(pool, &arrow.TimestampType{
		Unit: arrow.Nanosecond,
	})
	defer builder.Release()

	for _, v := range *vec {
		builder.Append(arrow.Timestamp((v).UnixNano()))
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableTimeColumn(pool memory.Allocator, field arrow.Field, vec *genericVector[*time.Time]) *array.Column {
	builder := array.NewTimestampBuilder(pool, &arrow.TimestampType{
		Unit: arrow.Nanosecond,
	})
	defer builder.Release()

	for _, v := range *vec {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(arrow.Timestamp(v.UnixNano()))
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildJSONColumn(pool memory.Allocator, field arrow.Field, vec *genericVector[json.RawMessage]) *array.Column {
	builder := array.NewBinaryBuilder(pool, &arrow.BinaryType{})
	defer builder.Release()

	for _, v := range *vec {
		builder.Append(v)
	}

	chunked := array.NewChunked(field.Type, []array.Interface{builder.NewArray()})
	defer chunked.Release()

	return array.NewColumn(field, chunked)
}

func buildNullableJSONColumn(pool memory.Allocator, field arrow.Field, vec *genericVector[*json.RawMessage]) *array.Column {
	builder := array.NewBinaryBuilder(pool, &arrow.BinaryType{})
	defer builder.Release()

	for _, v := range *vec {
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
