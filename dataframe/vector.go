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
	case []int64:
		v = newInt64Vector(n, VectorPTypeInt64)
	case []*int64:
		v = newNullableInt64Vector(n, VectorPTypeNullableInt64)
	case []uint64:
		v = newUint64Vector(n, VectorPTypeUint64)
	case []*uint64:
		v = newNullableUint64Vector(n, VectorPTypeNullableUInt64)
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
	// VectorPTypeInt64 indicates the underlying primitive is a []int64.
	VectorPTypeInt64 VectorPType = iota
	// VectorPTypeNullableInt64 indicates the underlying primitive is a []*int64.
	VectorPTypeNullableInt64

	// VectorPTypeUint64 indicates the underlying primitive is a []uint64.
	VectorPTypeUint64
	// VectorPTypeNullableUInt64 indicates the underlying primitive is a []*uint64.
	VectorPTypeNullableUInt64

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
