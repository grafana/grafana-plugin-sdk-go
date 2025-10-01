package data

import (
	"encoding/json"
	"fmt"
	"time"
)

// vector represents a Field's collection of Elements.
type vector interface {
	Set(idx int, i interface{})
	Append(i interface{})
	Extend(i int)
	At(i int) interface{}
	NilAt(i int) bool
	Len() int
	Type() FieldType
	PointerAt(i int) interface{}
	CopyAt(i int) interface{}
	ConcreteAt(i int) (val interface{}, ok bool)
	SetConcrete(i int, val interface{})
	Insert(i int, val interface{})
	Delete(i int)
}

// nolint:gocyclo
func vectorFieldType(v vector) FieldType {
	switch v.(type) {
	// time.Time
	case *genericVector[time.Time]:
		return FieldTypeTime
	case *nullableGenericVector[time.Time]:
		return FieldTypeNullableTime

	// json.RawMessage
	case *genericVector[json.RawMessage]:
		return FieldTypeJSON
	case *nullableGenericVector[json.RawMessage]:
		return FieldTypeNullableJSON

	// EnumItemIndex
	case *genericVector[EnumItemIndex]:
		return FieldTypeEnum
	case *nullableGenericVector[EnumItemIndex]:
		return FieldTypeNullableEnum

	case *genericVector[int8]:
		return FieldTypeInt8
	case *nullableGenericVector[int8]:
		return FieldTypeNullableInt8
	case *genericVector[int16]:
		return FieldTypeInt16
	case *nullableGenericVector[int16]:
		return FieldTypeNullableInt16
	case *genericVector[int32]:
		return FieldTypeInt32
	case *nullableGenericVector[int32]:
		return FieldTypeNullableInt32
	case *genericVector[int64]:
		return FieldTypeInt64
	case *nullableGenericVector[int64]:
		return FieldTypeNullableInt64
	case *genericVector[uint8]:
		return FieldTypeUint8
	case *nullableGenericVector[uint8]:
		return FieldTypeNullableUint8
	case *genericVector[uint16]:
		return FieldTypeUint16
	case *nullableGenericVector[uint16]:
		return FieldTypeNullableUint16
	case *genericVector[uint32]:
		return FieldTypeUint32
	case *nullableGenericVector[uint32]:
		return FieldTypeNullableUint32
	case *genericVector[uint64]:
		return FieldTypeUint64
	case *nullableGenericVector[uint64]:
		return FieldTypeNullableUint64
	case *genericVector[float32]:
		return FieldTypeFloat32
	case *nullableGenericVector[float32]:
		return FieldTypeNullableFloat32
	case *genericVector[float64]:
		return FieldTypeFloat64
	case *nullableGenericVector[float64]:
		return FieldTypeNullableFloat64
	case *genericVector[string]:
		return FieldTypeString
	case *nullableGenericVector[string]:
		return FieldTypeNullableString
	case *genericVector[bool]:
		return FieldTypeBool
	case *nullableGenericVector[bool]:
		return FieldTypeNullableBool
	}

	return FieldTypeUnknown
}

func (p FieldType) String() string {
	if p <= 0 {
		return "invalid/unsupported"
	}
	return fmt.Sprintf("[]%v", p.ItemTypeString())
}

// NewFieldFromFieldType creates a new Field of the given FieldType of length n.
// nolint:gocyclo
func NewFieldFromFieldType(p FieldType, n int) *Field {
	f := &Field{}
	switch p {
	// ints (use generic vectors for performance)
	case FieldTypeInt8:
		f.vector = newGenericVector[int8](n)
	case FieldTypeNullableInt8:
		f.vector = newNullableGenericVector[int8](n)

	case FieldTypeInt16:
		f.vector = newGenericVector[int16](n)
	case FieldTypeNullableInt16:
		f.vector = newNullableGenericVector[int16](n)

	case FieldTypeInt32:
		f.vector = newGenericVector[int32](n)
	case FieldTypeNullableInt32:
		f.vector = newNullableGenericVector[int32](n)

	case FieldTypeInt64:
		f.vector = newGenericVector[int64](n)
	case FieldTypeNullableInt64:
		f.vector = newNullableGenericVector[int64](n)

	// uints (use generic vectors for performance)
	case FieldTypeUint8:
		f.vector = newGenericVector[uint8](n)
	case FieldTypeNullableUint8:
		f.vector = newNullableGenericVector[uint8](n)

	case FieldTypeUint16:
		f.vector = newGenericVector[uint16](n)
	case FieldTypeNullableUint16:
		f.vector = newNullableGenericVector[uint16](n)

	case FieldTypeUint32:
		f.vector = newGenericVector[uint32](n)
	case FieldTypeNullableUint32:
		f.vector = newNullableGenericVector[uint32](n)

	case FieldTypeUint64:
		f.vector = newGenericVector[uint64](n)
	case FieldTypeNullableUint64:
		f.vector = newNullableGenericVector[uint64](n)

	// floats (use generic vectors for performance)
	case FieldTypeFloat32:
		f.vector = newGenericVector[float32](n)
	case FieldTypeNullableFloat32:
		f.vector = newNullableGenericVector[float32](n)

	case FieldTypeFloat64:
		f.vector = newGenericVector[float64](n)
	case FieldTypeNullableFloat64:
		f.vector = newNullableGenericVector[float64](n)

	// other basic types (use generic vectors for performance)
	case FieldTypeString:
		f.vector = newGenericVector[string](n)
	case FieldTypeNullableString:
		f.vector = newNullableGenericVector[string](n)

	case FieldTypeBool:
		f.vector = newGenericVector[bool](n)
	case FieldTypeNullableBool:
		f.vector = newNullableGenericVector[bool](n)

		// complex types (now using generic vectors)
	case FieldTypeTime:
		f.vector = newGenericVector[time.Time](n)
	case FieldTypeNullableTime:
		f.vector = newNullableGenericVector[time.Time](n)

	case FieldTypeJSON:
		f.vector = newGenericVector[json.RawMessage](n)
	case FieldTypeNullableJSON:
		f.vector = newNullableGenericVector[json.RawMessage](n)

	case FieldTypeEnum:
		f.vector = newGenericVector[EnumItemIndex](n)
	case FieldTypeNullableEnum:
		f.vector = newNullableGenericVector[EnumItemIndex](n)
	default:
		panic("unsupported FieldType")
	}
	return f
}
