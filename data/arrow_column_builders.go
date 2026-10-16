package data

import (
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

func buildStringArray(pool memory.Allocator, vec *stringVector) arrow.Array {
	builder := array.NewStringBuilder(pool)
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		builder.Append(v)
	}

	return builder.NewArray()
}

func buildNullableStringArray(pool memory.Allocator, vec *nullableStringVector) arrow.Array {
	builder := array.NewStringBuilder(pool)
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	return builder.NewArray()
}

func buildInt8Array(pool memory.Allocator, vec *int8Vector) arrow.Array {
	builder := array.NewInt8Builder(pool)
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		builder.Append(v)
	}

	return builder.NewArray()
}

func buildNullableInt8Array(pool memory.Allocator, vec *nullableInt8Vector) arrow.Array {
	builder := array.NewInt8Builder(pool)
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	return builder.NewArray()
}

func buildInt16Array(pool memory.Allocator, vec *int16Vector) arrow.Array {
	builder := array.NewInt16Builder(pool)
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		builder.Append(v)
	}

	return builder.NewArray()
}

func buildNullableInt16Array(pool memory.Allocator, vec *nullableInt16Vector) arrow.Array {
	builder := array.NewInt16Builder(pool)
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	return builder.NewArray()
}

func buildInt32Array(pool memory.Allocator, vec *int32Vector) arrow.Array {
	builder := array.NewInt32Builder(pool)
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		builder.Append(v)
	}

	return builder.NewArray()
}

func buildNullableInt32Array(pool memory.Allocator, vec *nullableInt32Vector) arrow.Array {
	builder := array.NewInt32Builder(pool)
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	return builder.NewArray()
}

func buildInt64Array(pool memory.Allocator, vec *int64Vector) arrow.Array {
	builder := array.NewInt64Builder(pool)
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		builder.Append(v)
	}

	return builder.NewArray()
}

func buildNullableInt64Array(pool memory.Allocator, vec *nullableInt64Vector) arrow.Array {
	builder := array.NewInt64Builder(pool)
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	return builder.NewArray()
}

func buildUInt8Array(pool memory.Allocator, vec *uint8Vector) arrow.Array {
	builder := array.NewUint8Builder(pool)
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		builder.Append(v)
	}

	return builder.NewArray()
}

func buildNullableUInt8Array(pool memory.Allocator, vec *nullableUint8Vector) arrow.Array {
	builder := array.NewUint8Builder(pool)
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	return builder.NewArray()
}

func buildUInt16Array(pool memory.Allocator, vec *uint16Vector) arrow.Array {
	builder := array.NewUint16Builder(pool)
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		builder.Append(v)
	}

	return builder.NewArray()
}

func buildNullableUInt16Array(pool memory.Allocator, vec *nullableUint16Vector) arrow.Array {
	builder := array.NewUint16Builder(pool)
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	return builder.NewArray()
}

func buildUInt32Array(pool memory.Allocator, vec *uint32Vector) arrow.Array {
	builder := array.NewUint32Builder(pool)
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		builder.Append(v)
	}

	return builder.NewArray()
}

func buildNullableUInt32Array(pool memory.Allocator, vec *nullableUint32Vector) arrow.Array {
	builder := array.NewUint32Builder(pool)
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	return builder.NewArray()
}

func buildUInt64Array(pool memory.Allocator, vec *uint64Vector) arrow.Array {
	builder := array.NewUint64Builder(pool)
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		builder.Append(v)
	}

	return builder.NewArray()
}

func buildNullableUInt64Array(pool memory.Allocator, vec *nullableUint64Vector) arrow.Array {
	builder := array.NewUint64Builder(pool)
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	return builder.NewArray()
}

func buildFloat32Array(pool memory.Allocator, vec *float32Vector) arrow.Array {
	builder := array.NewFloat32Builder(pool)
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		builder.Append(v)
	}

	return builder.NewArray()
}

func buildNullableFloat32Array(pool memory.Allocator, vec *nullableFloat32Vector) arrow.Array {
	builder := array.NewFloat32Builder(pool)
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	return builder.NewArray()
}

func buildFloat64Array(pool memory.Allocator, vec *float64Vector) arrow.Array {
	builder := array.NewFloat64Builder(pool)
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		builder.Append(v)
	}

	return builder.NewArray()
}

func buildNullableFloat64Array(pool memory.Allocator, vec *nullableFloat64Vector) arrow.Array {
	builder := array.NewFloat64Builder(pool)
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	return builder.NewArray()
}

func buildBoolArray(pool memory.Allocator, vec *boolVector) arrow.Array {
	builder := array.NewBooleanBuilder(pool)
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		builder.Append(v)
	}

	return builder.NewArray()
}

func buildNullableBoolArray(pool memory.Allocator, vec *nullableBoolVector) arrow.Array {
	builder := array.NewBooleanBuilder(pool)
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	return builder.NewArray()
}

func buildTimeArray(pool memory.Allocator, vec *timeTimeVector) arrow.Array {
	builder := array.NewTimestampBuilder(pool, &arrow.TimestampType{
		Unit: arrow.Nanosecond,
	})
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		builder.Append(arrow.Timestamp((v).UnixNano()))
	}

	return builder.NewTimestampArray()
}

func buildNullableTimeArray(pool memory.Allocator, vec *nullableTimeTimeVector) arrow.Array {
	builder := array.NewTimestampBuilder(pool, &arrow.TimestampType{
		Unit: arrow.Nanosecond,
	})
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(arrow.Timestamp(v.UnixNano()))
	}

	return builder.NewArray()
}

func buildJSONArray(pool memory.Allocator, vec *jsonRawMessageVector) arrow.Array {
	builder := array.NewBinaryBuilder(pool, &arrow.BinaryType{})
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		builder.Append(v)
	}

	return builder.NewArray()
}

func buildNullableJSONArray(pool memory.Allocator, vec *nullableJsonRawMessageVector) arrow.Array {
	builder := array.NewBinaryBuilder(pool, &arrow.BinaryType{})
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append(*v)
	}

	return builder.NewArray()
}

func buildNullableEnumArray(pool memory.Allocator, vec *nullableEnumVector) arrow.Array {
	builder := array.NewUint16Builder(pool)
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		if v == nil {
			builder.AppendNull()
			continue
		}
		builder.Append((uint16)(*v))
	}

	return builder.NewArray()
}

func buildEnumArray(pool memory.Allocator, vec *enumVector) arrow.Array {
	builder := array.NewUint16Builder(pool)
	builder.Reserve(len(*vec))
	defer builder.Release()

	for _, v := range *vec {
		builder.Append(uint16(v))
	}

	return builder.NewArray()
}
