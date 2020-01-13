package dataframe

import (
	"fmt"
	"time"
)

// Vector represents a collection of Elements.
type Vector interface {
	Set(idx int, i interface{})
	Append(i interface{})
	At(i int) interface{}
	Len() int
	PrimitiveType() VectorPType
	//buildArrowColumn(pool memory.Allocator, field arrow.Field) *array.Column
}

func newVector(t interface{}, n int) (v Vector) {
	switch t.(type) {
	// ints
	case []int8:
		v = newInt8Vector(n, VectorPTypeInt8)
	case []*int8:
		v = newNullableInt8Vector(n, VectorPTypeNullableInt8)
	case []int16:
		v = newInt16Vector(n, VectorPTypeInt16)
	case []*int16:
		v = newNullableInt16Vector(n, VectorPTypeNullableInt16)
	case []int32:
		v = newInt32Vector(n, VectorPTypeInt32)
	case []*int32:
		v = newNullableInt32Vector(n, VectorPTypeNullableInt32)
	case []int64:
		v = newInt64Vector(n, VectorPTypeInt64)
	case []*int64:
		v = newNullableInt64Vector(n, VectorPTypeNullableInt64)

	// uints
	case []uint8:
		v = newUint8Vector(n, VectorPTypeUint8)
	case []*uint8:
		v = newNullableUint8Vector(n, VectorPTypeNullableUint8)
	case []uint16:
		v = newUint16Vector(n, VectorPTypeUint16)
	case []*uint16:
		v = newNullableUint16Vector(n, VectorPTypeNullableUint16)
	case []uint32:
		v = newUint32Vector(n, VectorPTypeUint32)
	case []*uint32:
		v = newNullableUint32Vector(n, VectorPTypeNullableUint32)
	case []uint64:
		v = newUint64Vector(n, VectorPTypeUint64)
	case []*uint64:
		v = newNullableUint64Vector(n, VectorPTypeNullableUInt64)

	// floats
	case []float32:
		v = newFloat32Vector(n, VectorPTypeFloat32)
	case []*float32:
		v = newNullableFloat32Vector(n, VectorPTypeNullableFloat32)
	case []float64:
		v = newFloat64Vector(n, VectorPTypeFloat64)
	case []*float64:
		v = newNullableFloat64Vector(n, VectorPTypeNullableFloat64)

	case []string:
		v = newStringVector(n, VectorPTypeString)
	case []*string:
		v = newNullableStringVector(n, VectorPTypeNullableString)
	case []bool:
		v = newBoolVector(n, VectorPTypeBool)
	case []*bool:
		v = newNullableBoolVector(n, VectorPTypeNullableBool)
	case []time.Time:
		v = newTimeTimeVector(n, VectorPTypeTime)
	case []*time.Time:
		v = newNullableTimeTimeVector(n, VectorPTypeNullableTime)
	default:
		panic(fmt.Sprintf("unsupported vector type of %T", t))
	}
	return
}

// VectorPType indicates the go type underlying the Vector.
type VectorPType int

const (
	// VectorPTypeInt8 indicates the underlying primitive is a []int8.
	VectorPTypeInt8 VectorPType = iota
	// VectorPTypeNullableInt8 indicates the underlying primitive is a []*int8.
	VectorPTypeNullableInt8

	// VectorPTypeInt16 indicates the underlying primitive is a []Int16.
	VectorPTypeInt16
	// VectorPTypeNullableInt16 indicates the underlying primitive is a []*Int16.
	VectorPTypeNullableInt16

	// VectorPTypeInt32 indicates the underlying primitive is a []int32.
	VectorPTypeInt32
	// VectorPTypeNullableInt32 indicates the underlying primitive is a []*int32.
	VectorPTypeNullableInt32

	// VectorPTypeInt64 indicates the underlying primitive is a []int64.
	VectorPTypeInt64
	// VectorPTypeNullableInt64 indicates the underlying primitive is a []*int64.
	VectorPTypeNullableInt64

	// VectorPTypeUint8 indicates the underlying primitive is a []int8.
	VectorPTypeUint8
	// VectorPTypeNullableUint8 indicates the underlying primitive is a []*int8.
	VectorPTypeNullableUint8

	// VectorPTypeUint16 indicates the underlying primitive is a []uint16.
	VectorPTypeUint16
	// VectorPTypeNullableUint16 indicates the underlying primitive is a []*uint16.
	VectorPTypeNullableUint16

	// VectorPTypeUint32 indicates the underlying primitive is a []uint32.
	VectorPTypeUint32
	// VectorPTypeNullableUint32 indicates the underlying primitive is a []*uint32.
	VectorPTypeNullableUint32

	// VectorPTypeUint64 indicates the underlying primitive is a []uint64.
	VectorPTypeUint64
	// VectorPTypeNullableUInt64 indicates the underlying primitive is a []*uint64.
	VectorPTypeNullableUInt64

	// VectorPTypeFloat32 indicates the underlying primitive is a []float32.
	VectorPTypeFloat32
	// VectorPTypeNullableFloat32 indicates the underlying primitive is a []*float32.
	VectorPTypeNullableFloat32

	// VectorPTypeFloat64 indicates the underlying primitive is a []float64.
	VectorPTypeFloat64
	// VectorPTypeNullableFloat64 indicates the underlying primitive is a []*float64.
	VectorPTypeNullableFloat64

	// VectorPTypeString indicates the underlying primitive is a []string.
	VectorPTypeString
	// VectorPTypeNullableString indicates the underlying primitive is a []*string.
	VectorPTypeNullableString

	// VectorPTypeBool indicates the underlying primitive is a []bool.
	VectorPTypeBool
	// VectorPTypeNullableBool indicates the underlying primitive is a []*bool.
	VectorPTypeNullableBool

	// VectorPTypeTime indicates the underlying primitive is a []time.Time.
	VectorPTypeTime
	// VectorPTypeNullableTime indicates the underlying primitive is a []*time.Time.
	VectorPTypeNullableTime
)
