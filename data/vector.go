package data

import (
	"encoding/json"
	"fmt"
	"time"
)

type vectorType interface {
	uint8 | uint16 | uint32 | uint64 | int8 | int16 | int32 | int64 | float32 | float64 | string | bool | time.Time |
		*uint8 | *uint16 | *uint32 | *uint64 | *int8 | *int16 | *int32 | *int64 | *float32 | *float64 | *string | *bool | *time.Time
}

// vector represents a Field's collection of Elements.
type vector[T vectorType] interface {
	Set(idx int, v T)
	Append(v T)
	Extend(i int)
	At(i int) T
	Len() int
	Type() FieldType
	PointerAt(i int) *T
	CopyAt(i int) T
	ConcreteAt(i int) (val T, ok bool)
	SetConcrete(i int, val T)
	Insert(i int, val T)
	Delete(i int)
}

func vectorFieldType[T vectorType](v vector[T]) FieldType {
	var vt T
	switch any(vt).(type) {
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

func (p FieldType) String() string {
	if p <= 0 {
		return "invalid/unsupported"
	}
	return fmt.Sprintf("[]%v", p.ItemTypeString())
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

func (v *genericVector[T]) Set(idx int, i T) {
	(*v)[idx] = i
}

func (v *genericVector[T]) SetConcrete(idx int, i T) {
	v.Set(idx, i)
}

func (v *genericVector[T]) Append(i T) {
	*v = append(*v, i)
}

func (v *genericVector[T]) At(i int) T {
	return (*v)[i]
}

func (v *genericVector[T]) PointerAt(i int) *T {
	return &(*v)[i]
}

func (v *genericVector[T]) Len() int {
	return len(*v)
}

func (v *genericVector[T]) CopyAt(i int) T {
	var g T
	g = (*v)[i]
	return g
}

func (v *genericVector[T]) ConcreteAt(i int) (T, bool) {
	return v.At(i), true
}

func (v *genericVector[T]) Type() FieldType {
	return vectorFieldType[T](v)
}

func (v *genericVector[T]) Extend(i int) {
	*v = append(*v, make([]T, i)...)
}

func (v *genericVector[T]) Insert(i int, val T) {
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
