package data

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"
	"unsafe"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	jsoniter "github.com/json-iterator/go"
	"github.com/mattetti/filebuffer"

	sdkjsoniter "github.com/grafana/grafana-plugin-sdk-go/data/utils/jsoniter"
)

const simpleTypeString = "string"
const simpleTypeNumber = "number"
const simpleTypeBool = "boolean"
const simpleTypeTime = "time"
const simpleTypeEnum = "enum"
const simpleTypeOther = "other"

const jsonKeySchema = "schema"
const jsonKeyData = "data"

func init() { //nolint:gochecknoinits
	jsoniter.RegisterTypeEncoder("data.Frame", &dataFrameCodec{})
	jsoniter.RegisterTypeDecoder("data.Frame", &dataFrameCodec{})
}

type dataFrameCodec struct{}

func (codec *dataFrameCodec) IsEmpty(ptr unsafe.Pointer) bool {
	f := (*Frame)(ptr)
	return f.Fields == nil && f.RefID == "" && f.Meta == nil
}

func (codec *dataFrameCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	f := (*Frame)(ptr)
	writeDataFrame(f, stream, true, true)
}

func (codec *dataFrameCodec) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	frame := Frame{}
	err := readDataFrameJSON(&frame, iter)
	if err != nil {
		// keep existing iter error if it exists
		if iter.Error == nil {
			iter.Error = err
		}
		return
	}
	*((*Frame)(ptr)) = frame
}

// FrameInclude - custom type to hold Frame serialization options.
type FrameInclude int

// Known FrameInclude constants.
const (
	// IncludeAll serializes the entire Frame with both Schema and Data.
	IncludeAll FrameInclude = iota + 1
	// IncludeDataOnly only serializes data part of a frame.
	IncludeDataOnly
	// IncludeSchemaOnly only serializes schema part of a frame.
	IncludeSchemaOnly
)

// FrameJSONCache holds a byte representation of the schema separate from the data.
// Methods of FrameJSON are not goroutine-safe.
type FrameJSONCache struct {
	schema json.RawMessage
	data   json.RawMessage
}

// Bytes can return a subset of the cached frame json.  Note that requesting a section
// that was not serialized on creation will return an empty value
func (f *FrameJSONCache) Bytes(args FrameInclude) []byte {
	if f.schema != nil && (args == IncludeAll || args == IncludeSchemaOnly) {
		// Pre-calculate total size to avoid multiple allocations
		size := 1 + len(jsonKeySchema) + 3 + len(f.schema) // {" + schema + ":
		includeData := f.data != nil && (args == IncludeAll || args == IncludeDataOnly)
		if includeData {
			size += 1 + len(jsonKeyData) + 3 + len(f.data) // ," + data + ":
		}
		size++ // closing }

		out := make([]byte, 0, size)
		out = append(out, '{', '"')
		out = append(out, jsonKeySchema...)
		out = append(out, '"', ':')
		out = append(out, f.schema...)

		if includeData {
			out = append(out, ',', '"')
			out = append(out, jsonKeyData...)
			out = append(out, '"', ':')
			out = append(out, f.data...)
		}
		out = append(out, '}')
		return out
	}

	// only data
	if f.data != nil && (args == IncludeAll || args == IncludeDataOnly) {
		size := 1 + len(jsonKeyData) + 3 + len(f.data) + 1
		out := make([]byte, 0, size)
		out = append(out, '{', '"')
		out = append(out, jsonKeyData...)
		out = append(out, '"', ':')
		out = append(out, f.data...)
		out = append(out, '}')
		return out
	}

	return []byte("{}")
}

// SameSchema checks if both structures have the same schema
func (f *FrameJSONCache) SameSchema(dst *FrameJSONCache) bool {
	if f == nil || dst == nil {
		return false
	}
	return bytes.Equal(f.schema, dst.schema)
}

// SetData updates the data bytes with new values
func (f *FrameJSONCache) setData(frame *Frame) error {
	cfg := jsoniter.ConfigCompatibleWithStandardLibrary
	stream := cfg.BorrowStream(nil)
	defer cfg.ReturnStream(stream)

	writeDataFrameData(frame, stream)
	if stream.Error != nil {
		return stream.Error
	}

	buf := stream.Buffer()
	data := make([]byte, len(buf))
	copy(data, buf) // don't hold the internal jsoniter buffer
	f.data = data
	return nil
}

// SetSchema updates the schema bytes with new values
func (f *FrameJSONCache) setSchema(frame *Frame) error {
	cfg := jsoniter.ConfigCompatibleWithStandardLibrary
	stream := cfg.BorrowStream(nil)
	defer cfg.ReturnStream(stream)

	writeDataFrameSchema(frame, stream)
	if stream.Error != nil {
		return stream.Error
	}

	buf := stream.Buffer()
	data := make([]byte, len(buf))
	copy(data, buf) // don't hold the internal jsoniter buffer
	f.schema = data
	return nil
}

// MarshalJSON marshals Frame to JSON.
func (f *FrameJSONCache) MarshalJSON() ([]byte, error) {
	return f.Bytes(IncludeAll), nil
}

// FrameToJSON creates an object that holds schema and data independently.  This is
// useful for explicit control between the data and schema.
// For standard json serialization use `json.Marshal(frame)`
//
// NOTE: the format should be considered experimental until grafana 8 is released.
func FrameToJSON(frame *Frame, include FrameInclude) ([]byte, error) {
	cfg := jsoniter.ConfigCompatibleWithStandardLibrary
	stream := cfg.BorrowStream(nil)
	defer cfg.ReturnStream(stream)

	includeSchema := include == IncludeAll || include == IncludeSchemaOnly
	includeData := include == IncludeAll || include == IncludeDataOnly

	writeDataFrame(frame, stream, includeSchema, includeData)
	if stream.Error != nil {
		return nil, stream.Error
	}

	return append([]byte(nil), stream.Buffer()...), nil
}

// FrameToJSON creates an object that holds schema and data independently.  This is
// useful for explicit control between the data and schema.
// For standard json serialization use `json.Marshal(frame)`
//
// NOTE: the format should be considered experimental until grafana 8 is released.
func FrameToJSONCache(frame *Frame) (FrameJSONCache, error) {
	wrap := FrameJSONCache{}

	err := wrap.setSchema(frame)
	if err != nil {
		return wrap, err
	}

	err = wrap.setData(frame)
	if err != nil {
		return wrap, err
	}

	return wrap, nil
}

type frameSchema struct {
	Name   string         `json:"name,omitempty"`
	Fields []*schemaField `json:"fields,omitempty"`
	RefID  string         `json:"refId,omitempty"`
	Meta   *FrameMeta     `json:"meta,omitempty"`
}

type fieldTypeInfo struct {
	Frame    FieldType `json:"frame,omitempty"`
	Nullable bool      `json:"nullable,omitempty"`
}

// has vector... but without length
type schemaField struct {
	Field
	TypeInfo fieldTypeInfo `json:"typeInfo,omitempty"`
}

func readDataFrameJSON(frame *Frame, iter *jsoniter.Iterator) error {
	for l1Field := iter.ReadObject(); l1Field != ""; l1Field = iter.ReadObject() {
		switch l1Field {
		case jsonKeySchema:
			schema := frameSchema{}
			iter.ReadVal(&schema)
			frame.Name = schema.Name
			frame.RefID = schema.RefID
			frame.Meta = schema.Meta

			// Create a new field for each object
			for _, f := range schema.Fields {
				ft := f.TypeInfo.Frame
				if f.TypeInfo.Nullable {
					ft = ft.NullableType()
				}
				tmp := NewFieldFromFieldType(ft, 0)
				tmp.Name = f.Name
				tmp.Labels = f.Labels
				tmp.Config = f.Config
				frame.Fields = append(frame.Fields, tmp)
			}

		case jsonKeyData:
			err := readFrameData(iter, frame)
			if err != nil {
				return err
			}

		default:
			iter.ReportError("bind l1", "unexpected field: "+l1Field)
		}
	}
	return iter.Error
}

func readDataFramesJSON(frames *Frames, iter *jsoniter.Iterator) error {
	for iter.ReadArray() {
		frame := &Frame{}
		iter.ReadVal(frame)
		if iter.Error != nil {
			return iter.Error
		}
		*frames = append(*frames, frame)
	}
	return nil
}

func readFrameData(iter *jsoniter.Iterator, frame *Frame) error {
	var readValues, readNanos bool
	nanos := make([][]int64, len(frame.Fields))
	for l2Field := iter.ReadObject(); l2Field != ""; l2Field = iter.ReadObject() {
		switch l2Field {
		case "values":
			if !iter.ReadArray() {
				continue // empty fields
			}
			var fieldIndex int
			// Load the first field with a generic interface.
			// The length of the first will be assumed for the other fields
			// and can have a specialized parser
			if frame.Fields == nil {
				return errors.New("fields is nil, malformed key order or frame without schema")
			}

			field := frame.Fields[0]
			vec, err := jsonValuesToVector(iter, field.Type())
			if err != nil {
				return err
			}
			field.vector = vec
			size := vec.Len()

			addNanos := func() {
				if readNanos {
					if nanos[fieldIndex] != nil {
						// Use typed access for time fields to avoid boxing
						if tv, ok := field.vector.(*genericVector[time.Time]); ok {
							for i := 0; i < size; i++ {
								t := tv.AtTyped(i)
								tv.SetTyped(i, t.Add(time.Nanosecond*time.Duration(nanos[fieldIndex][i])))
							}
						} else if tv, ok := field.vector.(*nullableGenericVector[time.Time]); ok {
							for i := 0; i < size; i++ {
								pt := tv.AtTyped(i)
								if pt == nil {
									continue
								}
								t := *pt
								tWithNS := t.Add(time.Nanosecond * time.Duration(nanos[fieldIndex][i]))
								tv.SetTyped(i, &tWithNS)
							}
						} else {
							// Fallback for other types
							for i := 0; i < size; i++ {
								t, ok := field.ConcreteAt(i)
								if !ok {
									continue
								}
								field.Set(i, t.(time.Time).Add(time.Nanosecond*time.Duration(nanos[fieldIndex][i])))
							}
						}
					}
				}
			}

			addNanos()
			fieldIndex++
			for iter.ReadArray() {
				field = frame.Fields[fieldIndex]
				vec, err = readVector(iter, field.Type(), size)
				if err != nil {
					return err
				}

				field.vector = vec
				addNanos()
				fieldIndex++
			}
			readValues = true

		case "entities":
			fieldIndex := 0
			for iter.ReadArray() {
				t := iter.WhatIsNext()
				if t == sdkjsoniter.ObjectValue {
					for l3Field := iter.ReadObject(); l3Field != ""; l3Field = iter.ReadObject() {
						field := frame.Fields[fieldIndex]
						replace := getReplacementValue(l3Field, field.Type())
						for iter.ReadArray() {
							idx := iter.ReadInt()
							setConcreteTypedInVector(field.vector, idx, replace)
						}
					}
				} else {
					iter.ReadAny() // skip nils
				}
				fieldIndex++
			}

		case "nanos":
			fieldIndex := 0
			for iter.ReadArray() {
				field := frame.Fields[fieldIndex]

				t := iter.WhatIsNext()
				if t == sdkjsoniter.ArrayValue {
					for idx := 0; iter.ReadArray(); idx++ {
						ns := iter.ReadInt64()
						if readValues {
							t, ok := field.vector.ConcreteAt(idx)
							if !ok {
								continue
							}
							tWithNS := t.(time.Time).Add(time.Nanosecond * time.Duration(ns))
							setConcreteTypedInVector(field.vector, idx, tWithNS)
							continue
						}
						if idx == 0 {
							nanos[fieldIndex] = append(nanos[fieldIndex], ns)
						}
					}
				} else {
					iter.ReadAny() // skip nils
				}
				fieldIndex++
			}

			readNanos = true
		}
	}
	return nil
}

func getReplacementValue(key string, ft FieldType) interface{} {
	v := math.NaN()
	if key == "Inf" {
		v = math.Inf(1)
	} else if key == "NegInf" {
		v = math.Inf(-1)
	}
	if ft == FieldTypeFloat32 || ft == FieldTypeNullableFloat32 {
		return float32(v)
	}
	return v
}

func float64FromJSON(v interface{}) (float64, error) {
	fV, ok := v.(float64)
	if ok {
		return fV, nil
	}
	iV, ok := v.(int64)
	if ok {
		fV = float64(iV)
		return fV, nil
	}
	iiV, ok := v.(int)
	if ok {
		fV = float64(iiV)
		return fV, nil
	}
	sV, ok := v.(string)
	if ok {
		return strconv.ParseFloat(sV, 64)
	}

	return 0, fmt.Errorf("unable to convert float64 in json [%T]", v)
}

func int64FromJSON(v interface{}) (int64, error) {
	iV, ok := v.(int64)
	if ok {
		return iV, nil
	}
	sV, ok := v.(string)
	if ok {
		return strconv.ParseInt(sV, 0, 64)
	}
	fV, ok := v.(float64)
	if ok {
		return int64(fV), nil
	}

	return 0, fmt.Errorf("unable to convert int64 in json [%T]", v)
}

// in this path, we do not yet know the length and must discover it from the array
// nolint:gocyclo
func jsonValuesToVector(iter *jsoniter.Iterator, ft FieldType) (vector, error) {
	itere := sdkjsoniter.NewIterator(iter)
	// we handle Uint64 differently because the regular method for unmarshalling to []any does not work for uint64 correctly
	// due to jsoniter parsing logic that automatically converts all numbers to float64.
	// We can't use readUint64VectorJSON here because the size of the array is not known and the function requires the length parameter
	switch ft {
	case FieldTypeUint64:
		parseUint64 := func(s string) (uint64, error) {
			return strconv.ParseUint(s, 0, 64)
		}
		u, err := readArrayOfNumbers[uint64](itere, parseUint64, itere.ReadUint64)
		if err != nil {
			return nil, err
		}
		return newGenericVectorWithValues(u), nil

	case FieldTypeNullableUint64:
		parseUint64 := func(s string) (*uint64, error) {
			u, err := strconv.ParseUint(s, 0, 64)
			if err != nil {
				return nil, err
			}
			return &u, nil
		}
		u, err := readArrayOfNumbers[*uint64](itere, parseUint64, itere.ReadUint64Pointer)
		if err != nil {
			return nil, err
		}
		return newNullableGenericVectorWithValues(u), nil

	case FieldTypeInt64:
		vals := newGenericVector[int64](0)
		for iter.ReadArray() {
			v := iter.ReadInt64()
			vals.Append(v)
		}
		return vals, nil

	case FieldTypeNullableInt64:
		vals := newNullableGenericVector[int64](0)
		for iter.ReadArray() {
			t := iter.WhatIsNext()
			if t == sdkjsoniter.NilValue {
				iter.ReadNil()
				vals.Append(nil)
			} else {
				v := iter.ReadInt64()
				vals.Append(&v)
			}
		}
		return vals, nil

	case FieldTypeJSON, FieldTypeNullableJSON:
		vals := newGenericVector[json.RawMessage](0)
		for iter.ReadArray() {
			var v json.RawMessage
			t := iter.WhatIsNext()
			if t == sdkjsoniter.NilValue {
				iter.ReadNil()
			} else {
				iter.ReadVal(&v)
			}
			vals.Append(v)
		}

		// Convert this to the pointer flavor
		if ft == FieldTypeNullableJSON {
			size := vals.Len()
			nullable := newNullableGenericVector[json.RawMessage](size)
			for i := 0; i < size; i++ {
				v := vals.AtTyped(i) // Use typed access to avoid boxing
				nullable.SetTyped(i, &v)
			}
			return nullable, nil
		}

		return vals, nil
	}

	// if it's not uint64 field, handle the array the old way
	convert := func(v interface{}) (interface{}, error) {
		return v, nil
	}

	switch ft.NonNullableType() {
	case FieldTypeTime:
		convert = func(v interface{}) (interface{}, error) {
			fV, ok := v.(float64)
			if !ok {
				return nil, fmt.Errorf("error reading time")
			}
			return time.Unix(0, int64(fV)*int64(time.Millisecond)).UTC(), nil
		}

	case FieldTypeUint8:
		convert = func(v interface{}) (interface{}, error) {
			iV, err := int64FromJSON(v)
			return uint8(iV), err
		}

	case FieldTypeUint16: // enums and uint16 share the same backings
		convert = func(v interface{}) (interface{}, error) {
			iV, err := int64FromJSON(v)
			return uint16(iV), err
		}

	case FieldTypeEnum: // enums and uint16 share the same backings
		convert = func(v interface{}) (interface{}, error) {
			iV, err := int64FromJSON(v)
			return EnumItemIndex(iV), err
		}

	case FieldTypeUint32:
		convert = func(v interface{}) (interface{}, error) {
			iV, err := int64FromJSON(v)
			return uint32(iV), err
		}
	case FieldTypeInt8:
		convert = func(v interface{}) (interface{}, error) {
			iV, err := int64FromJSON(v)
			return int8(iV), err
		}

	case FieldTypeInt16:
		convert = func(v interface{}) (interface{}, error) {
			iV, err := int64FromJSON(v)
			return int16(iV), err
		}

	case FieldTypeInt32:
		convert = func(v interface{}) (interface{}, error) {
			iV, err := int64FromJSON(v)
			return int32(iV), err
		}

	case FieldTypeFloat32:
		convert = func(v interface{}) (interface{}, error) {
			fV, err := float64FromJSON(v)
			return float32(fV), err
		}

	case FieldTypeFloat64:
		convert = func(v interface{}) (interface{}, error) {
			return float64FromJSON(v)
		}

	case FieldTypeString:
		convert = func(v interface{}) (interface{}, error) {
			str, ok := v.(string)
			if ok {
				return str, nil
			}
			return fmt.Sprintf("%v", v), nil
		}

	case FieldTypeBool:
		convert = func(v interface{}) (interface{}, error) {
			val := v.(bool)
			return val, nil
		}

	case FieldTypeJSON:
		convert = func(v interface{}) (interface{}, error) {
			r, ok := v.(json.RawMessage)
			if ok {
				return r, nil
			}
			return nil, fmt.Errorf("unable to convert to json.RawMessage")
		}
	}

	arr := make([]interface{}, 0)
	err := itere.ReadVal(&arr)
	if err != nil {
		return nil, err
	}
	f := NewFieldFromFieldType(ft, len(arr))
	for i, v := range arr {
		if v != nil {
			norm, err := convert(v)
			if err != nil {
				return nil, err
			}
			setConcreteTypedInVector(f.vector, i, norm) // will be pointer for nullable types
		}
	}
	return f.vector, nil
}

func readArrayOfNumbers[T any](iter *sdkjsoniter.Iterator, parse func(string) (T, error), reader func() (T, error)) ([]T, error) {
	var def T
	var result []T
	for {
		next, err := iter.ReadArray()
		if err != nil {
			return nil, err
		}
		if !next {
			break
		}
		nextType, err := iter.WhatIsNext()
		if err != nil {
			return nil, err
		}
		switch nextType {
		case sdkjsoniter.StringValue:
			str, err := iter.ReadString()
			if err != nil {
				return nil, err
			}
			u, err := parse(str)
			if err != nil {
				return nil, iter.ReportError(fmt.Sprintf("readArrayOfNumbers[%T]", def), "cannot parse string")
			}
			result = append(result, u)
		case sdkjsoniter.NilValue:
			_, err := iter.ReadNil()
			if err != nil {
				return nil, err
			}
			// add T's default value. For reference type it will be nil, for value types the default value such as 0, false, ""
			// This is the same logic as in `read<Type>VectorJSON`
			result = append(result, def)
		default: // read as a number, if it is not expected field type, there will be error.
			u, err := reader()
			if err != nil {
				return nil, err
			}
			result = append(result, u)
		}
	}
	return result, nil
}

// nolint:gocyclo
func readVector(iter *jsoniter.Iterator, ft FieldType, size int) (vector, error) {
	switch ft {
	// Time, JSON, and Enum types with custom parsing logic
	case FieldTypeTime:
		// generic time vector
		vec := newGenericVector[time.Time](size)
		for i := 0; i < size; i++ {
			if !iter.ReadArray() {
				return nil, fmt.Errorf("expected array element %d", i)
			}
			t := iter.WhatIsNext()
			if t == jsoniter.NilValue {
				iter.ReadNil()
			} else {
				ms := iter.ReadInt64()
				tv := time.Unix(ms/int64(1e+3), (ms%int64(1e+3))*int64(1e+6)).UTC()
				vec.SetTyped(i, tv)
			}
		}
		if iter.ReadArray() {
			return nil, fmt.Errorf("array size mismatch: expected %d elements", size)
		}
		return vec, iter.Error
	case FieldTypeNullableTime:
		vec := newNullableGenericVector[time.Time](size)
		for i := 0; i < size; i++ {
			if !iter.ReadArray() {
				return nil, fmt.Errorf("expected array element %d", i)
			}
			t := iter.WhatIsNext()
			if t == jsoniter.NilValue {
				iter.ReadNil()
				vec.SetTyped(i, nil)
			} else {
				ms := iter.ReadInt64()
				tv := time.Unix(ms/int64(1e+3), (ms%int64(1e+3))*int64(1e+6)).UTC()
				vec.SetConcreteTyped(i, tv)
			}
		}
		if iter.ReadArray() {
			return nil, fmt.Errorf("array size mismatch: expected %d elements", size)
		}
		return vec, iter.Error
	case FieldTypeJSON:
		vec := newGenericVector[json.RawMessage](size)
		for i := 0; i < size; i++ {
			if !iter.ReadArray() {
				return nil, fmt.Errorf("expected array element %d", i)
			}
			t := iter.WhatIsNext()
			if t == jsoniter.NilValue {
				iter.ReadNil()
			} else {
				var v json.RawMessage
				iter.ReadVal(&v)
				vec.SetTyped(i, v)
			}
		}
		if iter.ReadArray() {
			return nil, fmt.Errorf("array size mismatch: expected %d elements", size)
		}
		return vec, iter.Error
	case FieldTypeNullableJSON:
		vec := newNullableGenericVector[json.RawMessage](size)
		for i := 0; i < size; i++ {
			if !iter.ReadArray() {
				return nil, fmt.Errorf("expected array element %d", i)
			}
			t := iter.WhatIsNext()
			if t == jsoniter.NilValue {
				iter.ReadNil()
				vec.SetTyped(i, nil)
			} else {
				var v json.RawMessage
				iter.ReadVal(&v)
				vec.SetTyped(i, &v)
			}
		}
		if iter.ReadArray() {
			return nil, fmt.Errorf("array size mismatch: expected %d elements", size)
		}
		return vec, iter.Error
	case FieldTypeEnum:
		return readEnumVectorJSON(iter, size)
	case FieldTypeNullableEnum:
		return readNullableEnumVectorJSON(iter, size)

	// Generic vectors - inline implementations
	case FieldTypeUint8:
		return readgenericVectorJSON[uint8](iter, size, iter.ReadUint8)
	case FieldTypeNullableUint8:
		return readnullableGenericVectorJSON[uint8](iter, size, iter.ReadUint8)
	case FieldTypeUint16:
		return readgenericVectorJSON[uint16](iter, size, iter.ReadUint16)
	case FieldTypeNullableUint16:
		return readnullableGenericVectorJSON[uint16](iter, size, iter.ReadUint16)
	case FieldTypeUint32:
		return readgenericVectorJSON[uint32](iter, size, iter.ReadUint32)
	case FieldTypeNullableUint32:
		return readnullableGenericVectorJSON[uint32](iter, size, iter.ReadUint32)
	case FieldTypeUint64:
		return readgenericVectorJSON[uint64](iter, size, iter.ReadUint64)
	case FieldTypeNullableUint64:
		return readnullableGenericVectorJSON[uint64](iter, size, iter.ReadUint64)
	case FieldTypeInt8:
		return readgenericVectorJSON[int8](iter, size, iter.ReadInt8)
	case FieldTypeNullableInt8:
		return readnullableGenericVectorJSON[int8](iter, size, iter.ReadInt8)
	case FieldTypeInt16:
		return readgenericVectorJSON[int16](iter, size, iter.ReadInt16)
	case FieldTypeNullableInt16:
		return readnullableGenericVectorJSON[int16](iter, size, iter.ReadInt16)
	case FieldTypeInt32:
		return readgenericVectorJSON[int32](iter, size, iter.ReadInt32)
	case FieldTypeNullableInt32:
		return readnullableGenericVectorJSON[int32](iter, size, iter.ReadInt32)
	case FieldTypeInt64:
		return readgenericVectorJSON[int64](iter, size, iter.ReadInt64)
	case FieldTypeNullableInt64:
		return readnullableGenericVectorJSON[int64](iter, size, iter.ReadInt64)
	case FieldTypeFloat32:
		return readgenericVectorJSON[float32](iter, size, iter.ReadFloat32)
	case FieldTypeNullableFloat32:
		return readnullableGenericVectorJSON[float32](iter, size, iter.ReadFloat32)
	case FieldTypeFloat64:
		return readgenericVectorJSON[float64](iter, size, iter.ReadFloat64)
	case FieldTypeNullableFloat64:
		return readnullableGenericVectorJSON[float64](iter, size, iter.ReadFloat64)
	case FieldTypeString:
		return readgenericVectorJSON[string](iter, size, iter.ReadString)
	case FieldTypeNullableString:
		return readnullableGenericVectorJSON[string](iter, size, iter.ReadString)
	case FieldTypeBool:
		return readgenericVectorJSON[bool](iter, size, iter.ReadBool)
	case FieldTypeNullableBool:
		return readnullableGenericVectorJSON[bool](iter, size, iter.ReadBool)
	}
	return nil, fmt.Errorf("unsuppoted type: %s", ft.ItemTypeString())
}

// Generic helper for reading non-nullable vectors from JSON
func readgenericVectorJSON[T any](iter *jsoniter.Iterator, size int, readFunc func() T) (*genericVector[T], error) {
	vec := newGenericVector[T](size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			return nil, fmt.Errorf("expected array element %d", i)
		}

		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
		} else {
			v := readFunc()
			vec.SetTyped(i, v)
		}
	}
	if iter.ReadArray() {
		return nil, fmt.Errorf("array size mismatch: expected %d elements", size)
	}
	return vec, iter.Error
}

// Generic helper for reading nullable vectors from JSON
func readnullableGenericVectorJSON[T any](iter *jsoniter.Iterator, size int, readFunc func() T) (*nullableGenericVector[T], error) {
	vec := newNullableGenericVector[T](size)
	for i := 0; i < size; i++ {
		if !iter.ReadArray() {
			return nil, fmt.Errorf("expected array element %d", i)
		}
		t := iter.WhatIsNext()
		if t == jsoniter.NilValue {
			iter.ReadNil()
			vec.SetTyped(i, nil)
		} else {
			v := readFunc()
			vec.SetTyped(i, &v)
		}
	}
	if iter.ReadArray() {
		return nil, fmt.Errorf("array size mismatch: expected %d elements", size)
	}
	return vec, iter.Error
}

// This returns the type name that is used in javascript
func getTypeScriptTypeString(t FieldType) (string, bool) {
	if t.Time() {
		return simpleTypeTime, true
	}
	if t.Numeric() {
		return simpleTypeNumber, true
	}
	switch t {
	case FieldTypeBool, FieldTypeNullableBool:
		return simpleTypeBool, true
	case FieldTypeString, FieldTypeNullableString:
		return simpleTypeString, true
	case FieldTypeEnum, FieldTypeNullableEnum:
		return simpleTypeEnum, true
	case FieldTypeJSON, FieldTypeNullableJSON:
		return simpleTypeOther, true
	}
	return "", false
}

func getFieldTypeForArrow(t arrow.DataType, tsType string) FieldType {
	switch t.ID() {
	case arrow.TIMESTAMP:
		return FieldTypeTime
	case arrow.UINT8:
		return FieldTypeUint8
	case arrow.UINT16:
		if tsType == simpleTypeEnum {
			return FieldTypeEnum
		}
		return FieldTypeUint16
	case arrow.UINT32:
		return FieldTypeUint32
	case arrow.UINT64:
		return FieldTypeUint64
	case arrow.INT8:
		return FieldTypeInt8
	case arrow.INT16:
		return FieldTypeInt16
	case arrow.INT32:
		return FieldTypeInt32
	case arrow.INT64:
		return FieldTypeInt64
	case arrow.FLOAT32:
		return FieldTypeFloat32
	case arrow.FLOAT64:
		return FieldTypeFloat64
	case arrow.STRING:
		return FieldTypeString
	case arrow.BOOL:
		return FieldTypeBool
	case arrow.BINARY:
		return FieldTypeJSON
	}
	return FieldTypeUnknown
}

// export interface FieldValueEntityLookup {
// 	NaN?: number[];
// 	Undef?: number[]; // Missing because of absence or join
// 	Inf?: number[];
// 	NegInf?: number[];
//   }

type fieldEntityLookup struct {
	NaN    []int `json:"NaN,omitempty"`
	Inf    []int `json:"Inf,omitempty"`
	NegInf []int `json:"NegInf,omitempty"`
}

const (
	entityNaN         = "NaN"
	entityPositiveInf = "+Inf"
	entityNegativeInf = "-Inf"
)

// Pre-allocate a small capacity to avoid initial allocations
const entitySliceInitialCap = 8

// Pool for reusing fieldEntityLookup objects
var entityLookupPool = sync.Pool{
	New: func() interface{} {
		return &fieldEntityLookup{}
	},
}

// Pool for reusing string slices when sorting map keys
var stringSlicePool = sync.Pool{
	New: func() interface{} {
		s := make([]string, 0, 16) // Pre-allocate for typical label count
		return &s
	},
}

// getEntityLookup gets a fieldEntityLookup from the pool
func getEntityLookup() *fieldEntityLookup {
	return entityLookupPool.Get().(*fieldEntityLookup)
}

// putEntityLookup returns a fieldEntityLookup to the pool after resetting it
func putEntityLookup(f *fieldEntityLookup) {
	if f == nil {
		return
	}
	// Reset slices but keep capacity
	f.NaN = f.NaN[:0]
	f.Inf = f.Inf[:0]
	f.NegInf = f.NegInf[:0]
	entityLookupPool.Put(f)
}

func (f *fieldEntityLookup) add(str string, idx int) {
	switch str {
	case entityPositiveInf:
		if f.Inf == nil {
			f.Inf = make([]int, 0, entitySliceInitialCap)
		}
		f.Inf = append(f.Inf, idx)
	case entityNegativeInf:
		if f.NegInf == nil {
			f.NegInf = make([]int, 0, entitySliceInitialCap)
		}
		f.NegInf = append(f.NegInf, idx)
	case entityNaN:
		if f.NaN == nil {
			f.NaN = make([]int, 0, entitySliceInitialCap)
		}
		f.NaN = append(f.NaN, idx)
	}
}

func isSpecialEntity(v float64) (string, bool) {
	switch {
	case math.IsNaN(v):
		return entityNaN, true
	case math.IsInf(v, 1):
		return entityPositiveInf, true
	case math.IsInf(v, -1):
		return entityNegativeInf, true
	default:
		return "", false
	}
}

func writeDataFrame(frame *Frame, stream *jsoniter.Stream, includeSchema bool, includeData bool) {
	stream.WriteObjectStart()
	if includeSchema {
		stream.WriteObjectField(jsonKeySchema)
		writeDataFrameSchema(frame, stream)
	}

	if includeData {
		if includeSchema {
			stream.WriteMore()
		}

		stream.WriteObjectField(jsonKeyData)
		writeDataFrameData(frame, stream)
	}
	stream.WriteObjectEnd()
}

func writeDataFrameSchema(frame *Frame, stream *jsoniter.Stream) {
	started := false
	stream.WriteObjectStart()

	if len(frame.Name) > 0 {
		stream.WriteObjectField("name")
		stream.WriteString(frame.Name)
		started = true
	}

	if len(frame.RefID) > 0 {
		if started {
			stream.WriteMore()
		}
		stream.WriteObjectField("refId")
		stream.WriteString(frame.RefID)
		started = true
	}

	if frame.Meta != nil {
		if started {
			stream.WriteMore()
		}
		stream.WriteObjectField("meta")
		stream.WriteVal(frame.Meta)
		started = true
	}

	if started {
		stream.WriteMore()
	}
	stream.WriteObjectField("fields")
	stream.WriteArrayStart()
	for i, f := range frame.Fields {
		if i > 0 {
			stream.WriteMore()
		}
		started = false
		stream.WriteObjectStart()
		if len(f.Name) > 0 {
			stream.WriteObjectField("name")
			stream.WriteString(f.Name)
			started = true
		}

		t, ok := getTypeScriptTypeString(f.Type())
		if ok {
			if started {
				stream.WriteMore()
			}
			stream.WriteObjectField("type")
			stream.WriteString(t)
			started = true
		}

		ft := f.Type()
		nnt := ft.NonNullableType()
		if started {
			stream.WriteMore()
		}
		stream.WriteObjectField("typeInfo")
		stream.WriteObjectStart()
		stream.WriteObjectField("frame")
		stream.WriteString(nnt.ItemTypeString())
		if ft.Nullable() {
			stream.WriteMore()
			stream.WriteObjectField("nullable")
			stream.WriteBool(true)
		}
		stream.WriteObjectEnd()
		started = true

		if f.Labels != nil {
			if started {
				stream.WriteMore()
			}
			stream.WriteObjectField("labels")
			writeLabelsMap(stream, f.Labels)
			started = true
		}

		if f.Config != nil {
			if started {
				stream.WriteMore()
			}
			stream.WriteObjectField("config")
			writeFieldConfig(stream, f.Config)
		}

		stream.WriteObjectEnd()
	}
	stream.WriteArrayEnd()

	stream.WriteObjectEnd()
}

// fieldWriteResult contains the results of writing a field
type fieldWriteResult struct {
	entities     *fieldEntityLookup
	nanos        []int64
	hasNSTime    bool
	usedFallback bool
}

// writeTimeField writes time field data to the stream
func writeTimeField(f *Field, rowCount int, stream *jsoniter.Stream) fieldWriteResult {
	var nsTime []int64
	var hasNSTime bool

	if tv, ok := f.vector.(*genericVector[time.Time]); ok {
		for i := 0; i < rowCount; i++ {
			if i > 0 {
				stream.WriteRaw(",")
			}
			t := tv.AtTyped(i)
			ms := t.UnixMilli()
			stream.WriteInt64(ms)
			msRes := t.Truncate(time.Millisecond)
			ns := t.Sub(msRes).Nanoseconds()
			if ns != 0 {
				if !hasNSTime {
					nsTime = make([]int64, rowCount)
					hasNSTime = true
				}
				nsTime[i] = ns
			}
		}
		return fieldWriteResult{nanos: nsTime, hasNSTime: hasNSTime}
	}

	// Fallback
	for i := 0; i < rowCount; i++ {
		if i > 0 {
			stream.WriteRaw(",")
		}
		if v, ok := f.ConcreteAt(i); ok {
			t := v.(time.Time)
			stream.WriteInt64(t.UnixMilli())
			msRes := t.Truncate(time.Millisecond)
			ns := t.Sub(msRes).Nanoseconds()
			if ns != 0 {
				if !hasNSTime {
					nsTime = make([]int64, rowCount)
					hasNSTime = true
				}
				nsTime[i] = ns
			}
		} else {
			stream.WriteNil()
		}
	}
	return fieldWriteResult{nanos: nsTime, hasNSTime: hasNSTime, usedFallback: true}
}

// writeNullableTimeField writes nullable time field data to the stream
func writeNullableTimeField(f *Field, rowCount int, stream *jsoniter.Stream) fieldWriteResult {
	var nsTime []int64
	var hasNSTime bool

	if tv, ok := f.vector.(*nullableGenericVector[time.Time]); ok {
		for i := 0; i < rowCount; i++ {
			if i > 0 {
				stream.WriteRaw(",")
			}
			pt := tv.AtTyped(i)
			if pt == nil {
				stream.WriteNil()
				continue
			}
			t := *pt
			ms := t.UnixMilli()
			stream.WriteInt64(ms)
			msRes := t.Truncate(time.Millisecond)
			ns := t.Sub(msRes).Nanoseconds()
			if ns != 0 {
				if !hasNSTime {
					nsTime = make([]int64, rowCount)
					hasNSTime = true
				}
				nsTime[i] = ns
			}
		}
		return fieldWriteResult{nanos: nsTime, hasNSTime: hasNSTime}
	}

	// Fallback
	for i := 0; i < rowCount; i++ {
		if i > 0 {
			stream.WriteRaw(",")
		}
		if v, ok := f.ConcreteAt(i); ok {
			t := v.(time.Time)
			stream.WriteInt64(t.UnixMilli())
			msRes := t.Truncate(time.Millisecond)
			ns := t.Sub(msRes).Nanoseconds()
			if ns != 0 {
				if !hasNSTime {
					nsTime = make([]int64, rowCount)
					hasNSTime = true
				}
				nsTime[i] = ns
			}
		} else {
			stream.WriteNil()
		}
	}
	return fieldWriteResult{nanos: nsTime, hasNSTime: hasNSTime, usedFallback: true}
}

// writeFloatField writes float field data to the stream
func writeFloatField(f *Field, rowCount int, stream *jsoniter.Stream) fieldWriteResult {
	var entities *fieldEntityLookup

	switch f.Type() {
	case FieldTypeFloat64:
		if gv, ok := f.vector.(*genericVector[float64]); ok {
			for i := 0; i < rowCount; i++ {
				if i > 0 {
					stream.WriteRaw(",")
				}
				v := gv.AtTyped(i)
				if entityType, found := isSpecialEntity(v); found {
					if entities == nil {
						entities = getEntityLookup()
					}
					entities.add(entityType, i)
					stream.WriteNil()
				} else {
					stream.WriteFloat64(v)
				}
			}
			return fieldWriteResult{entities: entities}
		}
	case FieldTypeNullableFloat64:
		if gv, ok := f.vector.(*nullableGenericVector[float64]); ok {
			for i := 0; i < rowCount; i++ {
				if i > 0 {
					stream.WriteRaw(",")
				}
				pv := gv.AtTyped(i)
				if pv == nil {
					stream.WriteNil()
					continue
				}
				v := *pv
				if entityType, found := isSpecialEntity(v); found {
					if entities == nil {
						entities = getEntityLookup()
					}
					entities.add(entityType, i)
					stream.WriteNil()
				} else {
					stream.WriteFloat64(v)
				}
			}
			return fieldWriteResult{entities: entities}
		}
	case FieldTypeFloat32:
		if gv, ok := f.vector.(*genericVector[float32]); ok {
			for i := 0; i < rowCount; i++ {
				if i > 0 {
					stream.WriteRaw(",")
				}
				v := gv.AtTyped(i)
				if entityType, found := isSpecialEntity(float64(v)); found {
					if entities == nil {
						entities = getEntityLookup()
					}
					entities.add(entityType, i)
					stream.WriteNil()
				} else {
					stream.WriteFloat32(v)
				}
			}
			return fieldWriteResult{entities: entities}
		}
	case FieldTypeNullableFloat32:
		if gv, ok := f.vector.(*nullableGenericVector[float32]); ok {
			for i := 0; i < rowCount; i++ {
				if i > 0 {
					stream.WriteRaw(",")
				}
				pv := gv.AtTyped(i)
				if pv == nil {
					stream.WriteNil()
					continue
				}
				v := *pv
				if entityType, found := isSpecialEntity(float64(v)); found {
					if entities == nil {
						entities = getEntityLookup()
					}
					entities.add(entityType, i)
					stream.WriteNil()
				} else {
					stream.WriteFloat32(v)
				}
			}
			return fieldWriteResult{entities: entities}
		}
	}

	return fieldWriteResult{usedFallback: true}
}

// writeSignedIntField writes signed integer field data to the stream using generics
func writeSignedIntField[T int8 | int16 | int32 | int64](f *Field, rowCount int, stream *jsoniter.Stream) fieldWriteResult {
	if gv, ok := f.vector.(*genericVector[T]); ok {
		for i := 0; i < rowCount; i++ {
			if i > 0 {
				stream.WriteRaw(",")
			}
			stream.WriteInt64(int64(gv.AtTyped(i)))
		}
		return fieldWriteResult{}
	}
	if gv, ok := f.vector.(*nullableGenericVector[T]); ok {
		for i := 0; i < rowCount; i++ {
			if i > 0 {
				stream.WriteRaw(",")
			}
			pv := gv.AtTyped(i)
			if pv == nil {
				stream.WriteNil()
				continue
			}
			stream.WriteInt64(int64(*pv))
		}
		return fieldWriteResult{}
	}
	return fieldWriteResult{usedFallback: true}
}

// writeIntField writes signed integer field data to the stream
func writeIntField(f *Field, rowCount int, stream *jsoniter.Stream) fieldWriteResult {
	switch f.Type() {
	case FieldTypeInt8, FieldTypeNullableInt8:
		return writeSignedIntField[int8](f, rowCount, stream)
	case FieldTypeInt16, FieldTypeNullableInt16:
		return writeSignedIntField[int16](f, rowCount, stream)
	case FieldTypeInt32, FieldTypeNullableInt32:
		return writeSignedIntField[int32](f, rowCount, stream)
	case FieldTypeInt64, FieldTypeNullableInt64:
		return writeSignedIntField[int64](f, rowCount, stream)
	}
	return fieldWriteResult{usedFallback: true}
}

// writeUnsignedIntField writes unsigned integer field data to the stream using generics
func writeUnsignedIntField[T uint8 | uint16 | uint32 | uint64](f *Field, rowCount int, stream *jsoniter.Stream) fieldWriteResult {
	if gv, ok := f.vector.(*genericVector[T]); ok {
		for i := 0; i < rowCount; i++ {
			if i > 0 {
				stream.WriteRaw(",")
			}
			stream.WriteUint64(uint64(gv.AtTyped(i)))
		}
		return fieldWriteResult{}
	}
	if gv, ok := f.vector.(*nullableGenericVector[T]); ok {
		for i := 0; i < rowCount; i++ {
			if i > 0 {
				stream.WriteRaw(",")
			}
			pv := gv.AtTyped(i)
			if pv == nil {
				stream.WriteNil()
				continue
			}
			stream.WriteUint64(uint64(*pv))
		}
		return fieldWriteResult{}
	}
	return fieldWriteResult{usedFallback: true}
}

// writeUintField writes unsigned integer field data to the stream
func writeUintField(f *Field, rowCount int, stream *jsoniter.Stream) fieldWriteResult {
	switch f.Type() {
	case FieldTypeUint8, FieldTypeNullableUint8:
		return writeUnsignedIntField[uint8](f, rowCount, stream)
	case FieldTypeUint16, FieldTypeNullableUint16:
		return writeUnsignedIntField[uint16](f, rowCount, stream)
	case FieldTypeUint32, FieldTypeNullableUint32:
		return writeUnsignedIntField[uint32](f, rowCount, stream)
	case FieldTypeUint64, FieldTypeNullableUint64:
		return writeUnsignedIntField[uint64](f, rowCount, stream)
	}
	return fieldWriteResult{usedFallback: true}
}

// writeStringField writes string field data to the stream
func writeStringField(f *Field, rowCount int, stream *jsoniter.Stream) fieldWriteResult {
	switch f.Type() {
	case FieldTypeString:
		if gv, ok := f.vector.(*genericVector[string]); ok {
			for i := 0; i < rowCount; i++ {
				if i > 0 {
					stream.WriteRaw(",")
				}
				stream.WriteString(gv.AtTyped(i))
			}
			return fieldWriteResult{}
		}
	case FieldTypeNullableString:
		if gv, ok := f.vector.(*nullableGenericVector[string]); ok {
			for i := 0; i < rowCount; i++ {
				if i > 0 {
					stream.WriteRaw(",")
				}
				pv := gv.AtTyped(i)
				if pv == nil {
					stream.WriteNil()
					continue
				}
				stream.WriteString(*pv)
			}
			return fieldWriteResult{}
		}
	}

	return fieldWriteResult{usedFallback: true}
}

// writeBoolField writes bool field data to the stream
func writeBoolField(f *Field, rowCount int, stream *jsoniter.Stream) fieldWriteResult {
	switch f.Type() {
	case FieldTypeBool:
		if gv, ok := f.vector.(*genericVector[bool]); ok {
			for i := 0; i < rowCount; i++ {
				if i > 0 {
					stream.WriteRaw(",")
				}
				stream.WriteBool(gv.AtTyped(i))
			}
			return fieldWriteResult{}
		}
	case FieldTypeNullableBool:
		if gv, ok := f.vector.(*nullableGenericVector[bool]); ok {
			for i := 0; i < rowCount; i++ {
				if i > 0 {
					stream.WriteRaw(",")
				}
				pv := gv.AtTyped(i)
				if pv == nil {
					stream.WriteNil()
					continue
				}
				stream.WriteBool(*pv)
			}
			return fieldWriteResult{}
		}
	}

	return fieldWriteResult{usedFallback: true}
}

// writeJSONField writes JSON field data to the stream
func writeJSONField(f *Field, rowCount int, stream *jsoniter.Stream) fieldWriteResult {
	switch f.Type() {
	case FieldTypeJSON:
		if gv, ok := f.vector.(*genericVector[json.RawMessage]); ok {
			for i := 0; i < rowCount; i++ {
				if i > 0 {
					stream.WriteRaw(",")
				}
				msg := gv.AtTyped(i)
				if len(msg) == 0 || string(msg) == "null" {
					stream.WriteNil()
				} else {
					stream.WriteRaw(string(msg))
				}
			}
			return fieldWriteResult{}
		}
	case FieldTypeNullableJSON:
		if gv, ok := f.vector.(*nullableGenericVector[json.RawMessage]); ok {
			for i := 0; i < rowCount; i++ {
				if i > 0 {
					stream.WriteRaw(",")
				}
				pv := gv.AtTyped(i)
				if pv == nil || len(*pv) == 0 || string(*pv) == "null" {
					stream.WriteNil()
				} else {
					stream.WriteRaw(string(*pv))
				}
			}
			return fieldWriteResult{}
		}
	}

	return fieldWriteResult{usedFallback: true}
}

// writeEnumField writes enum field data to the stream
func writeEnumField(f *Field, rowCount int, stream *jsoniter.Stream) fieldWriteResult {
	switch f.Type() {
	case FieldTypeEnum:
		if gv, ok := f.vector.(*genericVector[EnumItemIndex]); ok {
			for i := 0; i < rowCount; i++ {
				if i > 0 {
					stream.WriteRaw(",")
				}
				stream.WriteUint16(uint16(gv.AtTyped(i)))
			}
			return fieldWriteResult{}
		}
	case FieldTypeNullableEnum:
		if gv, ok := f.vector.(*nullableGenericVector[EnumItemIndex]); ok {
			for i := 0; i < rowCount; i++ {
				if i > 0 {
					stream.WriteRaw(",")
				}
				pv := gv.AtTyped(i)
				if pv == nil {
					stream.WriteNil()
					continue
				}
				stream.WriteUint16(uint16(*pv))
			}
			return fieldWriteResult{}
		}
	}

	return fieldWriteResult{usedFallback: true}
}

func writeDataFrameData(frame *Frame, stream *jsoniter.Stream) {
	rowCount, err := frame.RowLen()
	if err != nil {
		stream.Error = err
		return
	}

	stream.WriteObjectStart()

	entities := make([]*fieldEntityLookup, len(frame.Fields))
	entityCount := 0

	nanos := make([][]int64, len(frame.Fields))
	nsOffSetCount := 0

	stream.WriteObjectField("values")
	stream.WriteArrayStart()
	for fidx, f := range frame.Fields {
		if fidx > 0 {
			stream.WriteMore()
		}

		stream.WriteArrayStart()

		var result fieldWriteResult

		switch f.Type() {
		case FieldTypeTime:
			result = writeTimeField(f, rowCount, stream)
		case FieldTypeNullableTime:
			result = writeNullableTimeField(f, rowCount, stream)
		case FieldTypeFloat64, FieldTypeNullableFloat64, FieldTypeFloat32, FieldTypeNullableFloat32:
			result = writeFloatField(f, rowCount, stream)
		case FieldTypeInt8, FieldTypeNullableInt8, FieldTypeInt16, FieldTypeNullableInt16,
			FieldTypeInt32, FieldTypeNullableInt32, FieldTypeInt64, FieldTypeNullableInt64:
			result = writeIntField(f, rowCount, stream)
		case FieldTypeUint8, FieldTypeNullableUint8, FieldTypeUint16, FieldTypeNullableUint16,
			FieldTypeUint32, FieldTypeNullableUint32, FieldTypeUint64, FieldTypeNullableUint64:
			result = writeUintField(f, rowCount, stream)
		case FieldTypeString, FieldTypeNullableString:
			result = writeStringField(f, rowCount, stream)
		case FieldTypeBool, FieldTypeNullableBool:
			result = writeBoolField(f, rowCount, stream)
		case FieldTypeJSON, FieldTypeNullableJSON:
			result = writeJSONField(f, rowCount, stream)
		case FieldTypeEnum, FieldTypeNullableEnum:
			result = writeEnumField(f, rowCount, stream)
		default:
			result = fieldWriteResult{usedFallback: true}
		}

		// Handle fallback path
		if result.usedFallback {
			for i := 0; i < rowCount; i++ {
				if i > 0 {
					stream.WriteRaw(",")
				}
				if v, ok := f.ConcreteAt(i); ok {
					stream.WriteVal(v)
				} else {
					stream.WriteNil()
				}
			}
		}

		stream.WriteArrayEnd()

		// Handle entities
		if result.entities != nil {
			entities[fidx] = result.entities
			entityCount++
		}

		// Handle nanosecond time offsets
		if result.hasNSTime {
			nanos[fidx] = result.nanos
			nsOffSetCount++
		}
	}
	stream.WriteArrayEnd()

	if entityCount > 0 {
		stream.WriteMore()
		stream.WriteObjectField("entities")
		writeEntitiesArray(stream, entities)
		// Return entities to pool after serialization
		for _, ent := range entities {
			putEntityLookup(ent)
		}
	}

	if nsOffSetCount > 0 {
		stream.WriteMore()
		stream.WriteObjectField("nanos")
		writeNanosArray(stream, nanos)
	}

	stream.WriteObjectEnd()
}

// writeLabelsMap writes a map[string]string without reflection
// This is significantly faster than WriteVal which uses reflection + sorting
func writeLabelsMap(stream *jsoniter.Stream, labels map[string]string) {
	if len(labels) == 0 {
		stream.WriteObjectStart()
		stream.WriteObjectEnd()
		return
	}

	// Option 1: Fast path - no sorting (non-deterministic)
	// Use this if deterministic output is not required
	// Saves ~200-300 MB allocations
	/*
		stream.WriteObjectStart()
		first := true
		for k, v := range labels {
			if !first {
				stream.WriteMore()
			}
			stream.WriteObjectField(k)
			stream.WriteString(v)
			first = false
		}
		stream.WriteObjectEnd()
	*/

	// Option 2: Deterministic path - with sorting
	// Required for consistent JSON output / tests
	// Uses pooled slice to avoid allocation
	keysPtr := stringSlicePool.Get().(*[]string)
	keys := (*keysPtr)[:0] // Reset length but keep capacity

	for k := range labels {
		keys = append(keys, k)
	}

	// Sort for deterministic output
	// Most label maps are small (< 10 keys), so this is fast
	sort.Strings(keys)

	stream.WriteObjectStart()
	for i, k := range keys {
		if i > 0 {
			stream.WriteMore()
		}
		stream.WriteObjectField(k)
		stream.WriteString(labels[k])
	}
	stream.WriteObjectEnd()

	// Return keys slice to pool
	*keysPtr = keys
	stringSlicePool.Put(keysPtr)
}

// writeFieldConfig writes FieldConfig without full reflection
// This manually serializes common simple fields and uses WriteVal for complex nested structures
// nolint:gocyclo
func writeFieldConfig(stream *jsoniter.Stream, config *FieldConfig) {
	stream.WriteObjectStart()
	needsComma := false

	// Simple string fields
	if config.DisplayName != "" {
		stream.WriteObjectField("displayName")
		stream.WriteString(config.DisplayName)
		needsComma = true
	}

	if config.DisplayNameFromDS != "" {
		if needsComma {
			stream.WriteMore()
		}
		stream.WriteObjectField("displayNameFromDS")
		stream.WriteString(config.DisplayNameFromDS)
		needsComma = true
	}

	if config.Path != "" {
		if needsComma {
			stream.WriteMore()
		}
		stream.WriteObjectField("path")
		stream.WriteString(config.Path)
		needsComma = true
	}

	if config.Description != "" {
		if needsComma {
			stream.WriteMore()
		}
		stream.WriteObjectField("description")
		stream.WriteString(config.Description)
		needsComma = true
	}

	// Pointer bool fields
	if config.Filterable != nil {
		if needsComma {
			stream.WriteMore()
		}
		stream.WriteObjectField("filterable")
		stream.WriteBool(*config.Filterable)
		needsComma = true
	}

	if config.Writeable != nil {
		if needsComma {
			stream.WriteMore()
		}
		stream.WriteObjectField("writeable")
		stream.WriteBool(*config.Writeable)
		needsComma = true
	}

	// Numeric fields
	if config.Unit != "" {
		if needsComma {
			stream.WriteMore()
		}
		stream.WriteObjectField("unit")
		stream.WriteString(config.Unit)
		needsComma = true
	}

	if config.Decimals != nil {
		if needsComma {
			stream.WriteMore()
		}
		stream.WriteObjectField("decimals")
		stream.WriteUint16(*config.Decimals)
		needsComma = true
	}

	if config.Min != nil {
		if needsComma {
			stream.WriteMore()
		}
		stream.WriteObjectField("min")
		stream.WriteVal(config.Min) // ConfFloat64 has custom MarshalJSON
		needsComma = true
	}

	if config.Max != nil {
		if needsComma {
			stream.WriteMore()
		}
		stream.WriteObjectField("max")
		stream.WriteVal(config.Max) // ConfFloat64 has custom MarshalJSON
		needsComma = true
	}

	if config.Interval != 0 {
		if needsComma {
			stream.WriteMore()
		}
		stream.WriteObjectField("interval")
		stream.WriteFloat64(config.Interval)
		needsComma = true
	}

	// Complex fields - use WriteVal for these as they're less common
	// and would require hundreds of lines to serialize manually
	if config.Mappings != nil {
		if needsComma {
			stream.WriteMore()
		}
		stream.WriteObjectField("mappings")
		stream.WriteVal(config.Mappings)
		needsComma = true
	}

	if config.Thresholds != nil {
		if needsComma {
			stream.WriteMore()
		}
		stream.WriteObjectField("thresholds")
		stream.WriteVal(config.Thresholds)
		needsComma = true
	}

	if config.Color != nil {
		if needsComma {
			stream.WriteMore()
		}
		stream.WriteObjectField("color")
		stream.WriteVal(config.Color)
		needsComma = true
	}

	if config.Links != nil {
		if needsComma {
			stream.WriteMore()
		}
		stream.WriteObjectField("links")
		stream.WriteVal(config.Links)
		needsComma = true
	}

	if config.NoValue != "" {
		if needsComma {
			stream.WriteMore()
		}
		stream.WriteObjectField("noValue")
		stream.WriteString(config.NoValue)
		needsComma = true
	}

	if config.TypeConfig != nil {
		if needsComma {
			stream.WriteMore()
		}
		stream.WriteObjectField("type")
		stream.WriteVal(config.TypeConfig)
		needsComma = true
	}

	if config.Custom != nil {
		if needsComma {
			stream.WriteMore()
		}
		stream.WriteObjectField("custom")
		stream.WriteVal(config.Custom)
		// needsComma = true last comma is not used
	}

	stream.WriteObjectEnd()
}

// writeEntitiesArray writes entities array without reflection
func writeEntitiesArray(stream *jsoniter.Stream, entities []*fieldEntityLookup) {
	stream.WriteArrayStart()
	for i, ent := range entities {
		if i > 0 {
			stream.WriteMore()
		}
		if ent == nil {
			stream.WriteNil()
			continue
		}
		stream.WriteObjectStart()
		hasField := false
		if len(ent.NaN) > 0 {
			stream.WriteObjectField("NaN")
			stream.WriteArrayStart()
			for j, idx := range ent.NaN {
				if j > 0 {
					stream.WriteMore()
				}
				stream.WriteInt(idx)
			}
			stream.WriteArrayEnd()
			hasField = true
		}
		if len(ent.Inf) > 0 {
			if hasField {
				stream.WriteMore()
			}
			stream.WriteObjectField("Inf")
			stream.WriteArrayStart()
			for j, idx := range ent.Inf {
				if j > 0 {
					stream.WriteMore()
				}
				stream.WriteInt(idx)
			}
			stream.WriteArrayEnd()
			hasField = true
		}
		if len(ent.NegInf) > 0 {
			if hasField {
				stream.WriteMore()
			}
			stream.WriteObjectField("NegInf")
			stream.WriteArrayStart()
			for j, idx := range ent.NegInf {
				if j > 0 {
					stream.WriteMore()
				}
				stream.WriteInt(idx)
			}
			stream.WriteArrayEnd()
		}
		stream.WriteObjectEnd()
	}
	stream.WriteArrayEnd()
}

// writeNanosArray writes nanos array without reflection
func writeNanosArray(stream *jsoniter.Stream, nanos [][]int64) {
	stream.WriteArrayStart()
	for i, nano := range nanos {
		if i > 0 {
			stream.WriteMore()
		}
		if nano == nil {
			stream.WriteNil()
			continue
		}
		stream.WriteArrayStart()
		for j, ns := range nano {
			if j > 0 {
				stream.WriteMore()
			}
			stream.WriteInt64(ns)
		}
		stream.WriteArrayEnd()
	}
	stream.WriteArrayEnd()
}

func writeDataFrames(frames *Frames, stream *jsoniter.Stream) {
	if frames == nil {
		return
	}
	stream.WriteArrayStart()
	for _, frame := range *frames {
		stream.WriteVal(frame)
	}
	stream.WriteArrayEnd()
}

// ArrowBufferToJSON writes a frame to JSON
// NOTE: the format should be considered experimental until grafana 8 is released.
func ArrowBufferToJSON(b []byte, include FrameInclude) ([]byte, error) {
	fB := filebuffer.New(b)
	fR, err := ipc.NewFileReader(fB)
	if err != nil {
		return nil, err
	}
	defer func() { _ = fR.Close() }()

	record, err := fR.Read()
	if errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("no records found")
	}
	if err != nil {
		return nil, err
	}
	// TODO?? multiple records in one file?

	return ArrowToJSON(record, include)
}

// ArrowToJSON writes a frame to JSON
// NOTE: the format should be considered experimental until grafana 8 is released.
func ArrowToJSON(record arrow.Record, include FrameInclude) ([]byte, error) {
	cfg := jsoniter.ConfigCompatibleWithStandardLibrary
	stream := cfg.BorrowStream(nil)
	defer cfg.ReturnStream(stream)

	started := false
	stream.WriteObjectStart()
	if include == IncludeAll || include == IncludeSchemaOnly {
		stream.WriteObjectField("schema")
		writeArrowSchema(stream, record)
		started = true
	}
	if include == IncludeAll || include == IncludeDataOnly {
		if started {
			stream.WriteMore()
		}
		stream.WriteObjectField("data")
		err := writeArrowData(stream, record)
		if err != nil {
			return nil, err
		}
	}

	stream.WriteObjectEnd()

	if stream.Error != nil {
		return nil, stream.Error
	}
	return append([]byte(nil), stream.Buffer()...), nil
}

func writeArrowSchema(stream *jsoniter.Stream, record arrow.Record) {
	started := false
	metaData := record.Schema().Metadata()

	stream.WriteObjectStart()

	name, _ := getMDKey(metadataKeyName, metaData) // No need to check ok, zero value ("") is returned
	refID, _ := getMDKey(metadataKeyRefID, metaData)

	if len(name) > 0 {
		stream.WriteObjectField("name")
		stream.WriteString(name)
		started = true
	}

	if len(refID) > 0 {
		if started {
			stream.WriteMore()
		}
		stream.WriteObjectField("refId")
		stream.WriteString(refID)
		started = true
	}

	if metaAsString, ok := getMDKey("meta", metaData); ok {
		if started {
			stream.WriteMore()
		}
		stream.WriteObjectField("meta")
		stream.WriteRaw(metaAsString)
		started = true
	}

	if started {
		stream.WriteMore()
	}
	stream.WriteObjectField("fields")
	stream.WriteArrayStart()
	for i, f := range record.Schema().Fields() {
		if i > 0 {
			stream.WriteMore()
		}
		started = false
		stream.WriteObjectStart()
		if len(f.Name) > 0 {
			stream.WriteObjectField("name")
			stream.WriteString(f.Name)
			started = true
		}

		tsType, ok := getMDKey(metadataKeyTSType, f.Metadata)
		ft := getFieldTypeForArrow(f.Type, tsType)
		if !ok {
			tsType, ok = getTypeScriptTypeString(ft)
		}

		if ok {
			if started {
				stream.WriteMore()
			}
			stream.WriteObjectField("type")
			stream.WriteString(tsType)

			nnt := ft.NonNullableType()
			stream.WriteMore()
			stream.WriteObjectField("typeInfo")
			stream.WriteObjectStart()
			stream.WriteObjectField("frame")
			stream.WriteString(nnt.ItemTypeString())
			if f.Nullable {
				stream.WriteMore()
				stream.WriteObjectField("nullable")
				stream.WriteBool(true)
			}
			stream.WriteObjectEnd()
		}

		if labelsAsString, ok := getMDKey("labels", f.Metadata); ok {
			stream.WriteMore()
			stream.WriteObjectField("labels")
			stream.WriteRaw(labelsAsString)
		}
		if labelsAsString, ok := getMDKey("config", f.Metadata); ok {
			stream.WriteMore()
			stream.WriteObjectField("config")
			stream.WriteRaw(labelsAsString)
		}

		stream.WriteObjectEnd()
	}
	stream.WriteArrayEnd()

	stream.WriteObjectEnd()
}

func writeArrowData(stream *jsoniter.Stream, record arrow.Record) error {
	fieldCount := len(record.Schema().Fields())

	stream.WriteObjectStart()

	entities := make([]*fieldEntityLookup, fieldCount)
	nanos := make([][]int64, fieldCount)
	var hasNano bool
	entityCount := 0

	stream.WriteObjectField("values")
	stream.WriteArrayStart()
	for fidx := 0; fidx < fieldCount; fidx++ {
		if fidx > 0 {
			stream.WriteMore()
		}
		col := record.Column(fidx)
		var ent *fieldEntityLookup

		switch col.DataType().ID() {
		case arrow.TIMESTAMP:
			nanoOffset := writeArrowDataTIMESTAMP(stream, col)
			if nanoOffset != nil {
				nanos[fidx] = nanoOffset
				hasNano = true
			}

		case arrow.UINT8:
			writeArrowDataUint8(stream, col)
		case arrow.UINT16:
			writeArrowDataUint16(stream, col)
		case arrow.UINT32:
			writeArrowDataUint32(stream, col)
		case arrow.UINT64:
			writeArrowDataUint64(stream, col)
		case arrow.INT8:
			writeArrowDataInt8(stream, col)
		case arrow.INT16:
			writeArrowDataInt16(stream, col)
		case arrow.INT32:
			writeArrowDataInt32(stream, col)
		case arrow.INT64:
			writeArrowDataInt64(stream, col)
		case arrow.FLOAT32:
			ent = writeArrowDataFloat32(stream, col)
		case arrow.FLOAT64:
			ent = writeArrowDataFloat64(stream, col)
		case arrow.STRING:
			writeArrowDataString(stream, col)
		case arrow.BOOL:
			writeArrowDataBool(stream, col)
		case arrow.BINARY:
			writeArrowDataBinary(stream, col)
		default:
			return fmt.Errorf("unsupported arrow type %s for JSON", col.DataType().ID())
		}

		if ent != nil {
			entities[fidx] = ent
			entityCount++
		}
	}
	stream.WriteArrayEnd()

	if entityCount > 0 {
		stream.WriteMore()
		stream.WriteObjectField("entities")
		writeEntitiesArray(stream, entities)
		// Return entities to pool after serialization
		for _, ent := range entities {
			putEntityLookup(ent)
		}
	}

	if hasNano {
		stream.WriteMore()
		stream.WriteObjectField("nanos")
		writeNanosArray(stream, nanos)
	}

	stream.WriteObjectEnd()
	return nil
}

// Custom timestamp extraction... assumes nanoseconds for everything now
func writeArrowDataTIMESTAMP(stream *jsoniter.Stream, col arrow.Array) []int64 {
	count := col.Len()
	var hasNSTime bool
	var nsTime []int64
	v := array.NewTimestampData(col.Data())
	stream.WriteArrayStart()
	for i := 0; i < count; i++ {
		if i > 0 {
			stream.WriteRaw(",")
		}
		if col.IsNull(i) {
			stream.WriteNil()
			continue
		}
		ns := v.Value(i)
		ms := int64(ns) / int64(time.Millisecond) // nanosecond assumption
		stream.WriteInt64(ms)

		nsOffSet := int64(ns) - ms*int64(1e6)
		if nsOffSet != 0 {
			if !hasNSTime {
				nsTime = make([]int64, count)
				hasNSTime = true
			}
			nsTime[i] = nsOffSet
		}

		if stream.Error != nil { // ???
			stream.Error = nil
			stream.WriteNil()
		}
	}
	stream.WriteArrayEnd()
	if hasNSTime {
		return nsTime
	}
	return nil
}
