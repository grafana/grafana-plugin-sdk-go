package data

import (
	"fmt"
	"math"
	"strconv"
	"time"
)

// Field represents a typed column of data within a Frame.
//
// A Field is essentially a slice of various types with extra properties and methods.
// See NewField() for supported types.
//
// The slice data in the Field is a not exported, so methods on the Field are used to to manipulate its data.
type Field[T vectorType] struct {
	// Name is default identifier of the field. The name does not have to be unique, but the combination
	// of name and Labels should be unique for proper behavior in all situations.
	Name string `json:"name,omitempty"`

	// Labels is an optional set of key=value pairs that in addition to the name, should uniquely
	// identify a Field within a Frame.
	Labels Labels `json:"labels,omitempty"`

	// Config is optional display configuration information for Grafana
	Config *FieldConfig `json:"config,omitempty"`

	// vector is the unexported values. it is unexported so we can change the underlying structure without
	// major breaking changes.
	vector vector[T]
}

// Fields is a slice of Field pointers.
type Fields[T vectorType] []*Field[T]

// NewField returns a instance of *Field[T]. Supported types for values are:
//
// Integers:
//  []int8, []*int8, []int16, []*int16, []int32, []*int32, []int64, []*int64
// Unsigned Integers:
//  []uint8, []*uint8, []uint16, []*uint16, []uint32, []*uint32, []uint64, []*uint64
// Floats:
//  []float32, []*float32, []float64, []*float64
// String, Bool, and Time:
//  []string, []*string, []bool, []*bool, []time.Time, and []*time.Time.
// JSON:
//  []json.RawMessage, []*json.RawMessage
//
// If an unsupported values type is passed, NewField will panic.
// nolint:gocyclo
func NewField[T vectorType](name string, labels Labels, values []T) *Field[T] {
	return &Field[T]{
		Name:   name,
		vector: newVectorWithValues[T](values),
		Labels: labels,
	}
}

// NewFieldWithSize returns a new Field[T] with the specified size.
func NewFieldWithSize[T vectorType](n int) *Field[T] {
	return &Field[T]{
		Name:   "",
		vector: newVector[T](n),
		Labels: make(Labels, 0),
	}

}

// Set sets the Field's value at index idx to val.
// It will panic if idx is out of range or if
// the underlying type of val does not match the element type of the Field.
func (f *Field[T]) Set(idx int, val T) {
	f.vector.Set(idx, val)
}

// SetConcrete sets the Field's value at index idx to val.
// val must be a non-pointer type or a panic will occur.
// If the underlying FieldType is nullable it will set val as a pointer to val. If the FieldType
// is not nullable, then this method behaves the same as the Set method.
// It will panic if the underlying type of val does not match the element concrete type of the Field.
func (f *Field[T]) SetConcrete(idx int, val T) {
	f.vector.SetConcrete(idx, val)
}

// Append appends element e to the Field.
// it will panic if the underlying type of e does not match the element type of the Field.
func (f *Field[T]) Append(e T) {
	f.vector.Append(e)
}

// Extend extends the Field length by i.
// Consider using Frame.Extend() when possible since all Fields within
// a Frame need to be of the same length before marshalling and transmission.
func (f *Field[T]) Extend(i int) {
	f.vector.Extend(i)
}

// At returns the the element at index idx of the Field.
// It will panic if idx is out of range.
func (f *Field[T]) At(idx int) T {
	return f.vector.At(idx)
}

// Len returns the number of elements in the Field.
func (f *Field[T]) Len() int {
	return f.vector.Len()
}

// Type returns the FieldType of the Field, which indicates what type of slice it is.
func (f *Field[T]) Type() FieldType {
	return f.vector.Type()
}

// PointerAt returns a pointer to the value at idx of the Field.
// It will panic if idx is out of range.
func (f *Field[T]) PointerAt(idx int) *T {
	return f.vector.PointerAt(idx)
}

// Insert extends the Field length by 1,
// shifts any existing field values at indices equal or greater to idx by one place
// and inserts val at index idx of the Field.
// If idx is equal to the Field length, then val will be appended.
// It idx exceeds the Field length, this method will panic.
func (f *Field[T]) Insert(idx int, val T) {
	f.vector.Insert(idx, val)
}

// Delete delete element at index idx of the Field.
func (f *Field[T]) Delete(idx int) {
	f.vector.Delete(idx)
}

// CopyAt returns a copy of the value of the specified index idx.
// It will panic if idx is out of range.
func (f *Field[T]) CopyAt(idx int) T {
	return f.vector.CopyAt(idx)
}

// ConcreteAt returns the concrete value at the specified index idx.
// A non-pointer type is returned regardless if the underlying vector is a pointer
// type or not. If the value is a pointer type, and is nil, then the zero value
// is returned and ok will be false.
func (f *Field[T]) ConcreteAt(idx int) (val T, ok bool) {
	return f.vector.ConcreteAt(idx)
}

// Nullable returns if the the Field's elements are nullable.
func (f *Field[T]) Nullable() bool {
	return f.Type().Nullable()
}

// FloatAt returns a float64 at the specified index idx for all supported Field types.
// It will panic if idx is out of range.
//
// If the Field type is numeric and the value at idx is nil, NaN is returned.
// Precision may be lost on large numbers.
//
// If the Field type is a bool then 0 is return if false or nil, and 1 if true.
//
// If the Field type is time.Time, then the millisecond epoch representation of the time
// is returned, or NaN is the value is nil.
//
// If the Field type is a string, then strconv.ParseFloat is called on it and will return
// an error if ParseFloat errors. If the value is nil, NaN is returned.
// nolint:gocyclo
func (f *Field[T]) FloatAt(idx int) (float64, error) {
	v := any(f.At(idx))
	switch v.(type) {
	case int8:
		return float64(v.(int8)), nil
	case *int8:
		iv := v.(*int8)
		if iv == nil {
			return math.NaN(), nil
		}
		return float64(*iv), nil

	case int16:
		return float64(v.(int16)), nil
	case *int16:
		iv := v.(*int16)
		if iv == nil {
			return math.NaN(), nil
		}
		return float64(*iv), nil

	case int32:
		return float64(v.(int32)), nil
	case *int32:
		iv := v.(*int32)
		if iv == nil {
			return math.NaN(), nil
		}
		return float64(*iv), nil

	case int64:
		return float64(v.(int64)), nil
	case *int64:
		iv := v.(*int64)
		if iv == nil {
			return math.NaN(), nil
		}
		return float64(*iv), nil

	case uint8:
		return float64(v.(uint8)), nil
	case *uint8:
		uiv := v.(*uint8)
		if uiv == nil {
			return math.NaN(), nil
		}
		return float64(*uiv), nil

	case uint16:
		return float64(v.(uint16)), nil
	case *uint16:
		uiv := v.(*uint16)
		if uiv == nil {
			return math.NaN(), nil
		}
		return float64(*uiv), nil

	case uint32:
		return float64(v.(uint32)), nil
	case *uint32:
		uiv := v.(*uint32)
		if uiv == nil {
			return math.NaN(), nil
		}
		return float64(*uiv), nil

	// TODO: third param for loss of precision?
	// Maybe something in math/big can help with this (also see https://github.com/golang/go/issues/29463).
	case uint64:
		return float64(v.(uint64)), nil
	case *uint64:
		uiv := v.(*uint64)
		if uiv == nil {
			return math.NaN(), nil
		}
		return float64(*uiv), nil

	case float32:
		return float64(v.(float32)), nil
	case *float32:
		fv := v.(*float32)
		if fv == nil {
			return math.NaN(), nil
		}
		return float64(*fv), nil

	case float64:
		return v.(float64), nil
	case *float64:
		fv := v.(*float64)
		if fv == nil {
			return math.NaN(), nil
		}
		return *fv, nil

	case string:
		s := v.(string)
		ft, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, err
		}
		return ft, nil
	case *string:
		s := v.(*string)
		if s == nil {
			return math.NaN(), nil
		}
		ft, err := strconv.ParseFloat(*s, 64)
		if err != nil {
			return 0, err
		}
		return ft, nil

	case bool:
		if v.(bool) {
			return 1, nil
		}
		return 0, nil

	case *bool:
		b := v.(*bool)
		if b == nil || !*b {
			return 0, nil
		}
		return 1, nil

	case time.Time:
		return float64(v.(time.Time).UnixNano() / int64(time.Millisecond)), nil
	case *time.Time:
		t := v.(*time.Time)
		if t == nil {
			return math.NaN(), nil
		}
		return float64(t.UnixNano() / int64(time.Millisecond)), nil
	}
	return 0, fmt.Errorf("unsupported field type %T", f.Type())
}

// NullableFloatAt it is similar to FloatAt but returns a *float64 at the specified index idx for all supported Field types.
// It will panic if idx is out of range.
func (f *Field[T]) NullableFloatAt(idx int) (*float64, error) {
	fv, err := f.FloatAt(idx)
	if err != nil {
		return nil, err
	}
	return &fv, nil
}
