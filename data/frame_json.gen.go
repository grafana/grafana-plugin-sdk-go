package data

import (
	"fmt"

	"github.com/apache/arrow/go/arrow/array"
	jsoniter "github.com/json-iterator/go"
)

//------------------------------------------------
// This file is generated from frame_json_test.go
//------------------------------------------------

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
		if stream.Error != nil { // NaN +Inf/-Inf
			txt := fmt.Sprintf("%v", v.Value(i))
			if entities == nil {
				entities = &fieldEntityLookup{}
			}
			entities.add(txt, i)
			stream.Error = nil
			stream.WriteNil()
		}
	}
	stream.WriteArrayEnd()
	return entities
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
		if stream.Error != nil { // NaN +Inf/-Inf
			txt := fmt.Sprintf("%v", v.Value(i))
			if entities == nil {
				entities = &fieldEntityLookup{}
			}
			entities.add(txt, i)
			stream.Error = nil
			stream.WriteNil()
		}
	}
	stream.WriteArrayEnd()
	return entities
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
		if stream.Error != nil { // NaN +Inf/-Inf
			txt := fmt.Sprintf("%v", v.Value(i))
			if entities == nil {
				entities = &fieldEntityLookup{}
			}
			entities.add(txt, i)
			stream.Error = nil
			stream.WriteNil()
		}
	}
	stream.WriteArrayEnd()
	return entities
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
		if stream.Error != nil { // NaN +Inf/-Inf
			txt := fmt.Sprintf("%v", v.Value(i))
			if entities == nil {
				entities = &fieldEntityLookup{}
			}
			entities.add(txt, i)
			stream.Error = nil
			stream.WriteNil()
		}
	}
	stream.WriteArrayEnd()
	return entities
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
		if stream.Error != nil { // NaN +Inf/-Inf
			txt := fmt.Sprintf("%v", v.Value(i))
			if entities == nil {
				entities = &fieldEntityLookup{}
			}
			entities.add(txt, i)
			stream.Error = nil
			stream.WriteNil()
		}
	}
	stream.WriteArrayEnd()
	return entities
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
		if stream.Error != nil { // NaN +Inf/-Inf
			txt := fmt.Sprintf("%v", v.Value(i))
			if entities == nil {
				entities = &fieldEntityLookup{}
			}
			entities.add(txt, i)
			stream.Error = nil
			stream.WriteNil()
		}
	}
	stream.WriteArrayEnd()
	return entities
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
		if stream.Error != nil { // NaN +Inf/-Inf
			txt := fmt.Sprintf("%v", v.Value(i))
			if entities == nil {
				entities = &fieldEntityLookup{}
			}
			entities.add(txt, i)
			stream.Error = nil
			stream.WriteNil()
		}
	}
	stream.WriteArrayEnd()
	return entities
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
		if stream.Error != nil { // NaN +Inf/-Inf
			txt := fmt.Sprintf("%v", v.Value(i))
			if entities == nil {
				entities = &fieldEntityLookup{}
			}
			entities.add(txt, i)
			stream.Error = nil
			stream.WriteNil()
		}
	}
	stream.WriteArrayEnd()
	return entities
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
		stream.WriteFloat32(v.Value(i))
		if stream.Error != nil { // NaN +Inf/-Inf
			txt := fmt.Sprintf("%v", v.Value(i))
			if entities == nil {
				entities = &fieldEntityLookup{}
			}
			entities.add(txt, i)
			stream.Error = nil
			stream.WriteNil()
		}
	}
	stream.WriteArrayEnd()
	return entities
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
		stream.WriteFloat64(v.Value(i))
		if stream.Error != nil { // NaN +Inf/-Inf
			txt := fmt.Sprintf("%v", v.Value(i))
			if entities == nil {
				entities = &fieldEntityLookup{}
			}
			entities.add(txt, i)
			stream.Error = nil
			stream.WriteNil()
		}
	}
	stream.WriteArrayEnd()
	return entities
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
		if stream.Error != nil { // NaN +Inf/-Inf
			txt := fmt.Sprintf("%v", v.Value(i))
			if entities == nil {
				entities = &fieldEntityLookup{}
			}
			entities.add(txt, i)
			stream.Error = nil
			stream.WriteNil()
		}
	}
	stream.WriteArrayEnd()
	return entities
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
		if stream.Error != nil { // NaN +Inf/-Inf
			txt := fmt.Sprintf("%v", v.Value(i))
			if entities == nil {
				entities = &fieldEntityLookup{}
			}
			entities.add(txt, i)
			stream.Error = nil
			stream.WriteNil()
		}
	}
	stream.WriteArrayEnd()
	return entities
}
