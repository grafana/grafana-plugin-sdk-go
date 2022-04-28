package data

import (
	"encoding/json"
	"time"
)

// vector represents a Field's collection of Elements.
type vector interface {
	Set(idx int, v interface{})
	Append(v interface{})
	Extend(i int)
	At(i int) interface{}
	Len() int
	Type() FieldType
	PointerAt(i int) interface{}
	CopyAt(i int) interface{}
	ConcreteAt(i int) (val interface{}, ok bool)
	SetConcrete(i int, val interface{})
	Insert(i int, val interface{})
	Delete(i int)
}

type vectorType interface {
	uint8 | uint16 | uint32 | uint64 | int8 | int16 | int32 | int64 | float32 | float64 | string | bool | time.Time | json.RawMessage |
		*uint8 | *uint16 | *uint32 | *uint64 | *int8 | *int16 | *int32 | *int64 | *float32 | *float64 | *string | *bool | *time.Time | *json.RawMessage
}

type genericVector[T vectorType] []T

func newVector[T vectorType](n int) *genericVector[T] {
	v := genericVector[T](make([]T, n))
	return &v
}

func newVectorWithValues[T vectorType](s []T) *genericVector[T] {
	v := make([]T, len(s))
	copy(v, s)
	return (*genericVector[T])(&v)
}

func (v *genericVector[T]) Set(idx int, i interface{}) {
	(*v)[idx] = i.(T)
}

func (v *genericVector[T]) SetConcrete(idx int, i interface{}) {
	v.Set(idx, i)
}

func (v *genericVector[T]) Append(i interface{}) {
	*v = append(*v, i.(T))
}

func (v *genericVector[T]) At(i int) interface{} {
	return (*v)[i]
}

func (v *genericVector[T]) PointerAt(i int) interface{} {
	return &(*v)[i]
}

func (v *genericVector[T]) Len() int {
	return len(*v)
}

func (v *genericVector[T]) CopyAt(i int) interface{} {
	var g T
	g = (*v)[i]
	return g
}

func (v *genericVector[T]) ConcreteAt(i int) (interface{}, bool) {
	return v.At(i).(T), true
}

func (v *genericVector[T]) Type() FieldType {
	var t T
	vt := any(t)
	switch vt.(type) {
	case int8:
		return FieldTypeInt8
	case *int8:
		return FieldTypeNullableInt8

	case int16:
		return FieldTypeInt16
	case *int16:
		return FieldTypeNullableInt16

	case int32:
		return FieldTypeInt32
	case *int32:
		return FieldTypeNullableInt32

	case int64:
		return FieldTypeInt64
	case *int64:
		return FieldTypeNullableInt64

	case uint8:
		return FieldTypeUint8
	case *uint8:
		return FieldTypeNullableUint8

	case uint16:
		return FieldTypeUint16
	case *uint16:
		return FieldTypeNullableUint16

	case uint32:
		return FieldTypeUint32
	case *uint32:
		return FieldTypeNullableUint32

	case uint64:
		return FieldTypeUint64
	case *uint64:
		return FieldTypeNullableUint64

	case float32:
		return FieldTypeFloat32
	case *float32:
		return FieldTypeNullableFloat32

	case float64:
		return FieldTypeFloat64
	case *float64:
		return FieldTypeNullableFloat64

	case string:
		return FieldTypeString
	case *string:
		return FieldTypeNullableString

	case bool:
		return FieldTypeBool
	case *bool:
		return FieldTypeNullableBool

	case time.Time:
		return FieldTypeTime
	case *time.Time:
		return FieldTypeNullableTime

	case json.RawMessage:
		return FieldTypeJSON
	case *json.RawMessage:
		return FieldTypeNullableJSON
	}

	return FieldTypeUnknown
}

func (v *genericVector[T]) Extend(i int) {
	*v = append(*v, make([]T, i)...)
}

func (v *genericVector[T]) Insert(i int, val interface{}) {
	switch {
	case i < v.Len():
		v.Extend(1)
		copy((*v)[i+1:], (*v)[i:])
		v.Set(i, val)
	case i == v.Len():
		v.Append(val)
	case i > v.Len():
		panic("Invalid index; vector length should be greater or equal to that index")
	}
}

func (v *genericVector[T]) Delete(i int) {
	*v = append((*v)[:i], (*v)[i+1:]...)
}
