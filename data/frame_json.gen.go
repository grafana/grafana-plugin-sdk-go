package data

import (
	"github.com/apache/arrow/go/arrow/array"
	jsoniter "github.com/json-iterator/go"
)

func writeArrowDataBinary(stream *jsoniter.Stream, col array.Interface) *fieldEntityLookup {
	var entities *fieldEntityLookup
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
	return entities
}

//-------------------------------------------------------------
// The rest of this file is generated from frame_json_test.go
//-------------------------------------------------------------
func writeArrowDataUint8(stream *jsoniter.Stream, col array.Interface) *fieldEntityLookup {
	var entities *fieldEntityLookup
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
	return entities
}

func readUint8VectorJSON(iter *jsoniter.Iterator, size int) (*genericVector[uint8], error) {
	arr := newVector[uint8](size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			iter.ReportError("readUint8VectorJSON", "expected array")
			return nil, iter.Error
		}

		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
		} else {
			v := iter.ReadUint8()
			arr.Set(i, v)
		}
	}

	if iter.ReadArray() {
		iter.ReportError("read", "expected close array")
		return nil, iter.Error
	}
	return arr, nil
}

func readNullableUint8VectorJSON(iter *jsoniter.Iterator, size int) (*genericVector[uint8], error) {
	arr := newVector[uint8](size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			iter.ReportError("readNullableUint8VectorJSON", "expected array")
			return nil, iter.Error
		}
		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
		} else {
			v := iter.ReadUint8()
			arr.Set(i, &v)
		}
	}

	if iter.ReadArray() {
		iter.ReportError("readNullableUint8VectorJSON", "expected close array")
		return nil, iter.Error
	}
	return arr, nil
}

func writeArrowDataUint16(stream *jsoniter.Stream, col array.Interface) *fieldEntityLookup {
	var entities *fieldEntityLookup
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
	return entities
}

func readUint16VectorJSON(iter *jsoniter.Iterator, size int) (*genericVector[uint16], error) {
	arr := newVector[uint16](size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			iter.ReportError("readUint16VectorJSON", "expected array")
			return nil, iter.Error
		}

		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
		} else {
			v := iter.ReadUint16()
			arr.Set(i, v)
		}
	}

	if iter.ReadArray() {
		iter.ReportError("read", "expected close array")
		return nil, iter.Error
	}
	return arr, nil
}

func readNullableUint16VectorJSON(iter *jsoniter.Iterator, size int) (*genericVector[uint16], error) {
	arr := newVector[uint16](size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			iter.ReportError("readNullableUint16VectorJSON", "expected array")
			return nil, iter.Error
		}
		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
		} else {
			v := iter.ReadUint16()
			arr.Set(i, &v)
		}
	}

	if iter.ReadArray() {
		iter.ReportError("readNullableUint16VectorJSON", "expected close array")
		return nil, iter.Error
	}
	return arr, nil
}

func writeArrowDataUint32(stream *jsoniter.Stream, col array.Interface) *fieldEntityLookup {
	var entities *fieldEntityLookup
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
	return entities
}

func readUint32VectorJSON(iter *jsoniter.Iterator, size int) (*genericVector[uint32], error) {
	arr := newVector[uint32](size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			iter.ReportError("readUint32VectorJSON", "expected array")
			return nil, iter.Error
		}

		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
		} else {
			v := iter.ReadUint32()
			arr.Set(i, v)
		}
	}

	if iter.ReadArray() {
		iter.ReportError("read", "expected close array")
		return nil, iter.Error
	}
	return arr, nil
}

func readNullableUint32VectorJSON(iter *jsoniter.Iterator, size int) (*genericVector[uint32], error) {
	arr := newVector[uint32](size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			iter.ReportError("readNullableUint32VectorJSON", "expected array")
			return nil, iter.Error
		}
		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
		} else {
			v := iter.ReadUint32()
			arr.Set(i, &v)
		}
	}

	if iter.ReadArray() {
		iter.ReportError("readNullableUint32VectorJSON", "expected close array")
		return nil, iter.Error
	}
	return arr, nil
}

func writeArrowDataUint64(stream *jsoniter.Stream, col array.Interface) *fieldEntityLookup {
	var entities *fieldEntityLookup
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
	return entities
}

func readUint64VectorJSON(iter *jsoniter.Iterator, size int) (*genericVector[uint64], error) {
	arr := newVector[uint64](size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			iter.ReportError("readUint64VectorJSON", "expected array")
			return nil, iter.Error
		}

		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
		} else {
			v := iter.ReadUint64()
			arr.Set(i, v)
		}
	}

	if iter.ReadArray() {
		iter.ReportError("read", "expected close array")
		return nil, iter.Error
	}
	return arr, nil
}

func readNullableUint64VectorJSON(iter *jsoniter.Iterator, size int) (*genericVector[uint64], error) {
	arr := newVector[uint64](size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			iter.ReportError("readNullableUint64VectorJSON", "expected array")
			return nil, iter.Error
		}
		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
		} else {
			v := iter.ReadUint64()
			arr.Set(i, &v)
		}
	}

	if iter.ReadArray() {
		iter.ReportError("readNullableUint64VectorJSON", "expected close array")
		return nil, iter.Error
	}
	return arr, nil
}

func writeArrowDataInt8(stream *jsoniter.Stream, col array.Interface) *fieldEntityLookup {
	var entities *fieldEntityLookup
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
	return entities
}

func readInt8VectorJSON(iter *jsoniter.Iterator, size int) (*genericVector[int8], error) {
	arr := newVector[int8](size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			iter.ReportError("readInt8VectorJSON", "expected array")
			return nil, iter.Error
		}

		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
		} else {
			v := iter.ReadInt8()
			arr.Set(i, v)
		}
	}

	if iter.ReadArray() {
		iter.ReportError("read", "expected close array")
		return nil, iter.Error
	}
	return arr, nil
}

func readNullableInt8VectorJSON(iter *jsoniter.Iterator, size int) (*genericVector[int8], error) {
	arr := newVector[int8](size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			iter.ReportError("readNullableInt8VectorJSON", "expected array")
			return nil, iter.Error
		}
		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
		} else {
			v := iter.ReadInt8()
			arr.Set(i, &v)
		}
	}

	if iter.ReadArray() {
		iter.ReportError("readNullableInt8VectorJSON", "expected close array")
		return nil, iter.Error
	}
	return arr, nil
}

func writeArrowDataInt16(stream *jsoniter.Stream, col array.Interface) *fieldEntityLookup {
	var entities *fieldEntityLookup
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
	return entities
}

func readInt16VectorJSON(iter *jsoniter.Iterator, size int) (*genericVector[int16], error) {
	arr := newVector[int16](size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			iter.ReportError("readInt16VectorJSON", "expected array")
			return nil, iter.Error
		}

		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
		} else {
			v := iter.ReadInt16()
			arr.Set(i, v)
		}
	}

	if iter.ReadArray() {
		iter.ReportError("read", "expected close array")
		return nil, iter.Error
	}
	return arr, nil
}

func readNullableInt16VectorJSON(iter *jsoniter.Iterator, size int) (*genericVector[int16], error) {
	arr := newVector[int16](size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			iter.ReportError("readNullableInt16VectorJSON", "expected array")
			return nil, iter.Error
		}
		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
		} else {
			v := iter.ReadInt16()
			arr.Set(i, &v)
		}
	}

	if iter.ReadArray() {
		iter.ReportError("readNullableInt16VectorJSON", "expected close array")
		return nil, iter.Error
	}
	return arr, nil
}

func writeArrowDataInt32(stream *jsoniter.Stream, col array.Interface) *fieldEntityLookup {
	var entities *fieldEntityLookup
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
	return entities
}

func readInt32VectorJSON(iter *jsoniter.Iterator, size int) (*genericVector[int32], error) {
	arr := newVector[int32](size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			iter.ReportError("readInt32VectorJSON", "expected array")
			return nil, iter.Error
		}

		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
		} else {
			v := iter.ReadInt32()
			arr.Set(i, v)
		}
	}

	if iter.ReadArray() {
		iter.ReportError("read", "expected close array")
		return nil, iter.Error
	}
	return arr, nil
}

func readNullableInt32VectorJSON(iter *jsoniter.Iterator, size int) (*genericVector[int32], error) {
	arr := newVector[int32](size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			iter.ReportError("readNullableInt32VectorJSON", "expected array")
			return nil, iter.Error
		}
		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
		} else {
			v := iter.ReadInt32()
			arr.Set(i, &v)
		}
	}

	if iter.ReadArray() {
		iter.ReportError("readNullableInt32VectorJSON", "expected close array")
		return nil, iter.Error
	}
	return arr, nil
}

func writeArrowDataInt64(stream *jsoniter.Stream, col array.Interface) *fieldEntityLookup {
	var entities *fieldEntityLookup
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
	return entities
}

func readInt64VectorJSON(iter *jsoniter.Iterator, size int) (*genericVector[int64], error) {
	arr := newVector[int64](size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			iter.ReportError("readInt64VectorJSON", "expected array")
			return nil, iter.Error
		}

		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
		} else {
			v := iter.ReadInt64()
			arr.Set(i, v)
		}
	}

	if iter.ReadArray() {
		iter.ReportError("read", "expected close array")
		return nil, iter.Error
	}
	return arr, nil
}

func readNullableInt64VectorJSON(iter *jsoniter.Iterator, size int) (*genericVector[int64], error) {
	arr := newVector[int64](size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			iter.ReportError("readNullableInt64VectorJSON", "expected array")
			return nil, iter.Error
		}
		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
		} else {
			v := iter.ReadInt64()
			arr.Set(i, &v)
		}
	}

	if iter.ReadArray() {
		iter.ReportError("readNullableInt64VectorJSON", "expected close array")
		return nil, iter.Error
	}
	return arr, nil
}

func writeArrowDataFloat32(stream *jsoniter.Stream, col array.Interface) *fieldEntityLookup {
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
				entities = &fieldEntityLookup{}
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

func readFloat32VectorJSON(iter *jsoniter.Iterator, size int) (*genericVector[float32], error) {
	arr := newVector[float32](size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			iter.ReportError("readFloat32VectorJSON", "expected array")
			return nil, iter.Error
		}

		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
		} else {
			v := iter.ReadFloat32()
			arr.Set(i, v)
		}
	}

	if iter.ReadArray() {
		iter.ReportError("read", "expected close array")
		return nil, iter.Error
	}
	return arr, nil
}

func readNullableFloat32VectorJSON(iter *jsoniter.Iterator, size int) (*genericVector[float32], error) {
	arr := newVector[float32](size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			iter.ReportError("readNullableFloat32VectorJSON", "expected array")
			return nil, iter.Error
		}
		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
		} else {
			v := iter.ReadFloat32()
			arr.Set(i, &v)
		}
	}

	if iter.ReadArray() {
		iter.ReportError("readNullableFloat32VectorJSON", "expected close array")
		return nil, iter.Error
	}
	return arr, nil
}

func writeArrowDataFloat64(stream *jsoniter.Stream, col array.Interface) *fieldEntityLookup {
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
		f64 := float64(val)
		if entityType, found := isSpecialEntity(f64); found {
			if entities == nil {
				entities = &fieldEntityLookup{}
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

func readFloat64VectorJSON(iter *jsoniter.Iterator, size int) (*genericVector[float64], error) {
	arr := newVector[float64](size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			iter.ReportError("readFloat64VectorJSON", "expected array")
			return nil, iter.Error
		}

		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
		} else {
			v := iter.ReadFloat64()
			arr.Set(i, v)
		}
	}

	if iter.ReadArray() {
		iter.ReportError("read", "expected close array")
		return nil, iter.Error
	}
	return arr, nil
}

func readNullableFloat64VectorJSON(iter *jsoniter.Iterator, size int) (*genericVector[float64], error) {
	arr := newVector[float64](size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			iter.ReportError("readNullableFloat64VectorJSON", "expected array")
			return nil, iter.Error
		}
		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
		} else {
			v := iter.ReadFloat64()
			arr.Set(i, &v)
		}
	}

	if iter.ReadArray() {
		iter.ReportError("readNullableFloat64VectorJSON", "expected close array")
		return nil, iter.Error
	}
	return arr, nil
}

func writeArrowDataString(stream *jsoniter.Stream, col array.Interface) *fieldEntityLookup {
	var entities *fieldEntityLookup
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
	return entities
}

func readStringVectorJSON(iter *jsoniter.Iterator, size int) (*genericVector[string], error) {
	arr := newVector[string](size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			iter.ReportError("readStringVectorJSON", "expected array")
			return nil, iter.Error
		}

		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
		} else {
			v := iter.ReadString()
			arr.Set(i, v)
		}
	}

	if iter.ReadArray() {
		iter.ReportError("read", "expected close array")
		return nil, iter.Error
	}
	return arr, nil
}

func readNullableStringVectorJSON(iter *jsoniter.Iterator, size int) (*genericVector[string], error) {
	arr := newVector[string](size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			iter.ReportError("readNullableStringVectorJSON", "expected array")
			return nil, iter.Error
		}
		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
		} else {
			v := iter.ReadString()
			arr.Set(i, &v)
		}
	}

	if iter.ReadArray() {
		iter.ReportError("readNullableStringVectorJSON", "expected close array")
		return nil, iter.Error
	}
	return arr, nil
}

func writeArrowDataBool(stream *jsoniter.Stream, col array.Interface) *fieldEntityLookup {
	var entities *fieldEntityLookup
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
	return entities
}

func readBoolVectorJSON(iter *jsoniter.Iterator, size int) (*genericVector[bool], error) {
	arr := newVector[bool](size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			iter.ReportError("readBoolVectorJSON", "expected array")
			return nil, iter.Error
		}

		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
		} else {
			v := iter.ReadBool()
			arr.Set(i, v)
		}
	}

	if iter.ReadArray() {
		iter.ReportError("read", "expected close array")
		return nil, iter.Error
	}
	return arr, nil
}

func readNullableBoolVectorJSON(iter *jsoniter.Iterator, size int) (*genericVector[bool], error) {
	arr := newVector[bool](size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			iter.ReportError("readNullableBoolVectorJSON", "expected array")
			return nil, iter.Error
		}
		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
		} else {
			v := iter.ReadBool()
			arr.Set(i, &v)
		}
	}

	if iter.ReadArray() {
		iter.ReportError("readNullableBoolVectorJSON", "expected close array")
		return nil, iter.Error
	}
	return arr, nil
}
