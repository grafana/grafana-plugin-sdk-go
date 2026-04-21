// This file contains writeArrowData functions for basic types

package data

import (
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	jsoniter "github.com/json-iterator/go"
)

func writeArrowDataBinary(stream *jsoniter.Stream, col arrow.Array) {
	count := col.Len()

	v := array.NewBinaryData(col.Data())
	stream.WriteArrayStart()
	for i := 0; i < count; i++ {
		if i > 0 {
			stream.WriteRaw(",")
		}
		if col.IsNull(i) {
			stream.WriteNil()
			continue
		}
		stream.WriteRaw(string(v.Value(i)))
	}
	stream.WriteArrayEnd()
}

func writeArrowDataUint8(stream *jsoniter.Stream, col arrow.Array) {
	count := col.Len()

	v := array.NewUint8Data(col.Data())
	stream.WriteArrayStart()
	for i := 0; i < count; i++ {
		if i > 0 {
			stream.WriteRaw(",")
		}
		if col.IsNull(i) {
			stream.WriteNil()
			continue
		}
		stream.WriteUint8(v.Value(i))
	}
	stream.WriteArrayEnd()
}

func writeArrowDataUint16(stream *jsoniter.Stream, col arrow.Array) {
	count := col.Len()

	v := array.NewUint16Data(col.Data())
	stream.WriteArrayStart()
	for i := 0; i < count; i++ {
		if i > 0 {
			stream.WriteRaw(",")
		}
		if col.IsNull(i) {
			stream.WriteNil()
			continue
		}
		stream.WriteUint16(v.Value(i))
	}
	stream.WriteArrayEnd()
}

func writeArrowDataUint32(stream *jsoniter.Stream, col arrow.Array) {
	count := col.Len()

	v := array.NewUint32Data(col.Data())
	stream.WriteArrayStart()
	for i := 0; i < count; i++ {
		if i > 0 {
			stream.WriteRaw(",")
		}
		if col.IsNull(i) {
			stream.WriteNil()
			continue
		}
		stream.WriteUint32(v.Value(i))
	}
	stream.WriteArrayEnd()
}

func writeArrowDataUint64(stream *jsoniter.Stream, col arrow.Array) {
	count := col.Len()

	v := array.NewUint64Data(col.Data())
	stream.WriteArrayStart()
	for i := 0; i < count; i++ {
		if i > 0 {
			stream.WriteRaw(",")
		}
		if col.IsNull(i) {
			stream.WriteNil()
			continue
		}
		stream.WriteUint64(v.Value(i))
	}
	stream.WriteArrayEnd()
}

func writeArrowDataInt8(stream *jsoniter.Stream, col arrow.Array) {
	count := col.Len()

	v := array.NewInt8Data(col.Data())
	stream.WriteArrayStart()
	for i := 0; i < count; i++ {
		if i > 0 {
			stream.WriteRaw(",")
		}
		if col.IsNull(i) {
			stream.WriteNil()
			continue
		}
		stream.WriteInt8(v.Value(i))
	}
	stream.WriteArrayEnd()
}

func writeArrowDataInt16(stream *jsoniter.Stream, col arrow.Array) {
	count := col.Len()

	v := array.NewInt16Data(col.Data())
	stream.WriteArrayStart()
	for i := 0; i < count; i++ {
		if i > 0 {
			stream.WriteRaw(",")
		}
		if col.IsNull(i) {
			stream.WriteNil()
			continue
		}
		stream.WriteInt16(v.Value(i))
	}
	stream.WriteArrayEnd()
}

func writeArrowDataInt32(stream *jsoniter.Stream, col arrow.Array) {
	count := col.Len()

	v := array.NewInt32Data(col.Data())
	stream.WriteArrayStart()
	for i := 0; i < count; i++ {
		if i > 0 {
			stream.WriteRaw(",")
		}
		if col.IsNull(i) {
			stream.WriteNil()
			continue
		}
		stream.WriteInt32(v.Value(i))
	}
	stream.WriteArrayEnd()
}

func writeArrowDataInt64(stream *jsoniter.Stream, col arrow.Array) {
	count := col.Len()

	v := array.NewInt64Data(col.Data())
	stream.WriteArrayStart()
	for i := 0; i < count; i++ {
		if i > 0 {
			stream.WriteRaw(",")
		}
		if col.IsNull(i) {
			stream.WriteNil()
			continue
		}
		stream.WriteInt64(v.Value(i))
	}
	stream.WriteArrayEnd()
}

func writeArrowDataFloat32(stream *jsoniter.Stream, col arrow.Array) *fieldEntityLookup {
	var entities *fieldEntityLookup
	count := col.Len()

	v := array.NewFloat32Data(col.Data())
	stream.WriteArrayStart()
	for i := 0; i < count; i++ {
		if i > 0 {
			stream.WriteRaw(",")
		}
		if col.IsNull(i) {
			stream.WriteNil()
			continue
		}
		val := v.Value(i)
		f64 := float64(val)
		if entityType, found := isSpecialEntity(f64); found {
			if entities == nil {
				entities = getEntityLookup()
			}
			entities.add(entityType, i)
			stream.WriteNil()
		} else {
			stream.WriteFloat32(val)
		}
	}
	stream.WriteArrayEnd()
	return entities
}

func writeArrowDataFloat64(stream *jsoniter.Stream, col arrow.Array) *fieldEntityLookup {
	var entities *fieldEntityLookup
	count := col.Len()

	v := array.NewFloat64Data(col.Data())
	stream.WriteArrayStart()
	for i := 0; i < count; i++ {
		if i > 0 {
			stream.WriteRaw(",")
		}
		if col.IsNull(i) {
			stream.WriteNil()
			continue
		}
		val := v.Value(i)
		if entityType, found := isSpecialEntity(val); found {
			if entities == nil {
				entities = getEntityLookup()
			}
			entities.add(entityType, i)
			stream.WriteNil()
		} else {
			stream.WriteFloat64(val)
		}
	}
	stream.WriteArrayEnd()
	return entities
}

func writeArrowDataString(stream *jsoniter.Stream, col arrow.Array) {
	count := col.Len()

	v := array.NewStringData(col.Data())
	stream.WriteArrayStart()
	for i := 0; i < count; i++ {
		if i > 0 {
			stream.WriteRaw(",")
		}
		if col.IsNull(i) {
			stream.WriteNil()
			continue
		}
		stream.WriteString(v.Value(i))
	}
	stream.WriteArrayEnd()
}

func writeArrowDataBool(stream *jsoniter.Stream, col arrow.Array) {
	count := col.Len()

	v := array.NewBooleanData(col.Data())
	stream.WriteArrayStart()
	for i := 0; i < count; i++ {
		if i > 0 {
			stream.WriteRaw(",")
		}
		if col.IsNull(i) {
			stream.WriteNil()
			continue
		}
		stream.WriteBool(v.Value(i))
	}
	stream.WriteArrayEnd()
}
