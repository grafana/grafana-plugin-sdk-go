package data

import (
	"fmt"
	"reflect"
	"time"
)

type typeFunc func(string, reflect.Type, nodes) (*Field, error)

func int8FieldFunc(name string, t reflect.Type, n nodes) (*Field, error) {
	d := make([]int8, len(n))
	for i, v := range n {
		d[i] = v.Val.Interface().(int8)
	}

	return NewField(name, nil, d), nil
}

func int8PtrFieldFunc(name string, t reflect.Type, n nodes) (*Field, error) {
	d := make([]*int8, len(n))
	for i, v := range n {
		d[i] = v.Val.Interface().(*int8)
	}
	return NewField(name, nil, d), nil
}

func int16FieldFunc(name string, t reflect.Type, n nodes) (*Field, error) {
	d := make([]int16, len(n))
	for i, v := range n {
		d[i] = v.Val.Interface().(int16)
	}

	return NewField(name, nil, d), nil
}

func int16PtrFieldFunc(name string, t reflect.Type, n nodes) (*Field, error) {
	d := make([]*int16, len(n))
	for i, v := range n {
		d[i] = v.Val.Interface().(*int16)
	}

	return NewField(name, nil, d), nil
}

func int32FieldFunc(name string, t reflect.Type, n nodes) (*Field, error) {
	d := make([]int32, len(n))
	for i, v := range n {
		d[i] = v.Val.Interface().(int32)
	}

	return NewField(name, nil, d), nil
}

func int32PtrFieldFunc(name string, t reflect.Type, n nodes) (*Field, error) {
	d := make([]*int32, len(n))
	for i, v := range n {
		d[i] = v.Val.Interface().(*int32)
	}

	return NewField(name, nil, d), nil
}

func int64FieldFunc(name string, t reflect.Type, n nodes) (*Field, error) {
	d := make([]int64, len(n))
	for i, v := range n {
		d[i] = v.Val.Interface().(int64)
	}

	return NewField(name, nil, d), nil
}

func int64PtrFieldFunc(name string, t reflect.Type, n nodes) (*Field, error) {
	d := make([]*int64, len(n))
	for i, v := range n {
		d[i] = v.Val.Interface().(*int64)
	}

	return NewField(name, nil, d), nil
}

func uint8FieldFunc(name string, t reflect.Type, n nodes) (*Field, error) {
	d := make([]uint8, len(n))
	for i, v := range n {
		d[i] = v.Val.Interface().(uint8)
	}

	return NewField(name, nil, d), nil
}

func uint8PtrFieldFunc(name string, t reflect.Type, n nodes) (*Field, error) {
	d := make([]*uint8, len(n))
	for i, v := range n {
		d[i] = v.Val.Interface().(*uint8)
	}

	return NewField(name, nil, d), nil
}

func uint16FieldFunc(name string, t reflect.Type, n nodes) (*Field, error) {
	d := make([]uint16, len(n))
	for i, v := range n {
		d[i] = v.Val.Interface().(uint16)
	}

	return NewField(name, nil, d), nil
}

func uint16PtrFieldFunc(name string, t reflect.Type, n nodes) (*Field, error) {
	d := make([]*uint16, len(n))
	for i, v := range n {
		d[i] = v.Val.Interface().(*uint16)
	}

	return NewField(name, nil, d), nil
}

func uint32FieldFunc(name string, t reflect.Type, n nodes) (*Field, error) {
	d := make([]uint32, len(n))
	for i, v := range n {
		d[i] = v.Val.Interface().(uint32)
	}

	return NewField(name, nil, d), nil
}

func uint32PtrFieldFunc(name string, t reflect.Type, n nodes) (*Field, error) {
	d := make([]*uint32, len(n))
	for i, v := range n {
		d[i] = v.Val.Interface().(*uint32)
	}

	return NewField(name, nil, d), nil
}

func uint64FieldFunc(name string, t reflect.Type, n nodes) (*Field, error) {
	d := make([]uint64, len(n))
	for i, v := range n {
		d[i] = v.Val.Interface().(uint64)
	}

	return NewField(name, nil, d), nil
}

func uint64PtrFieldFunc(name string, t reflect.Type, n nodes) (*Field, error) {
	d := make([]*uint64, len(n))
	for i, v := range n {
		d[i] = v.Val.Interface().(*uint64)
	}

	return NewField(name, nil, d), nil
}

func float32FieldFunc(name string, t reflect.Type, n nodes) (*Field, error) {
	d := make([]float32, len(n))
	for i, v := range n {
		d[i] = v.Val.Interface().(float32)
	}

	return NewField(name, nil, d), nil
}

func float32PtrFieldFunc(name string, t reflect.Type, n nodes) (*Field, error) {
	d := make([]*float32, len(n))
	for i, v := range n {
		d[i] = v.Val.Interface().(*float32)
	}

	return NewField(name, nil, d), nil
}

func float64FieldFunc(name string, t reflect.Type, n nodes) (*Field, error) {
	d := make([]float64, len(n))
	for i, v := range n {
		d[i] = v.Val.Interface().(float64)
	}

	return NewField(name, nil, d), nil
}

func float64PtrFieldFunc(name string, t reflect.Type, n nodes) (*Field, error) {
	d := make([]*float64, len(n))
	for i, v := range n {
		d[i] = v.Val.Interface().(*float64)
	}

	return NewField(name, nil, d), nil
}

func stringFieldFunc(name string, t reflect.Type, n nodes) (*Field, error) {
	d := make([]string, len(n))
	for i, v := range n {
		d[i] = v.Val.Interface().(string)
	}

	return NewField(name, nil, d), nil
}

func stringPtrFieldFunc(name string, t reflect.Type, n nodes) (*Field, error) {
	d := make([]*string, len(n))
	for i, v := range n {
		d[i] = v.Val.Interface().(*string)
	}

	return NewField(name, nil, d), nil
}

func boolFieldFunc(name string, t reflect.Type, n nodes) (*Field, error) {
	d := make([]bool, len(n))
	for i, v := range n {
		d[i] = v.Val.Interface().(bool)
	}

	return NewField(name, nil, d), nil
}

func boolPtrFieldFunc(name string, t reflect.Type, n nodes) (*Field, error) {
	d := make([]*bool, len(n))
	for i, v := range n {
		d[i] = v.Val.Interface().(*bool)
	}

	return NewField(name, nil, d), nil
}

func timePtrFieldFunc(name string, t reflect.Type, n nodes) (*Field, error) {
	d := make([]*time.Time, len(n))
	for i, v := range n {
		d[i] = v.Val.Interface().(*time.Time)
	}

	return NewField(name, nil, d), nil
}

func timeFieldFunc(name string, t reflect.Type, n nodes) (*Field, error) {
	d := make([]time.Time, len(n))
	for i, v := range n {
		d[i] = v.Val.Interface().(time.Time)
	}

	return NewField(name, nil, d), nil
}

func structFieldFunc(name string, t reflect.Type, n nodes) (*Field, error) {
	if t == reflect.TypeOf(time.Time{}) {
		return timeFieldFunc(name, t, n)
	}
	return nil, nil
}

func ptrFieldFunc(name string, t reflect.Type, n nodes) (*Field, error) {
	if f, ok := typePtrFuncs[t]; ok {
		return f(name, t, n)
	}
	return nil, fmt.Errorf("%s: %w", t, ErrorUnrecognizedType)
}

var typePtrFuncs = map[reflect.Type]typeFunc{
	reflect.TypeOf(new(int8)):    int8PtrFieldFunc,
	reflect.TypeOf(new(int16)):   int16PtrFieldFunc,
	reflect.TypeOf(new(int32)):   int32PtrFieldFunc,
	reflect.TypeOf(new(int64)):   int64PtrFieldFunc,
	reflect.TypeOf(new(uint8)):   uint8PtrFieldFunc,
	reflect.TypeOf(new(uint16)):  uint16PtrFieldFunc,
	reflect.TypeOf(new(uint32)):  uint32PtrFieldFunc,
	reflect.TypeOf(new(uint64)):  uint64PtrFieldFunc,
	reflect.TypeOf(new(float32)): float32PtrFieldFunc,
	reflect.TypeOf(new(float64)): float64PtrFieldFunc,
	reflect.TypeOf(new(string)):  stringPtrFieldFunc,
	reflect.TypeOf(new(bool)):    boolPtrFieldFunc,
	reflect.TypeOf(&time.Time{}): timePtrFieldFunc,
}

var typeFuncs = map[reflect.Kind]typeFunc{
	reflect.Int8:    int8FieldFunc,
	reflect.Int16:   int16FieldFunc,
	reflect.Int32:   int32FieldFunc,
	reflect.Int64:   int64FieldFunc,
	reflect.Uint8:   uint8FieldFunc,
	reflect.Uint16:  uint16FieldFunc,
	reflect.Uint32:  uint32FieldFunc,
	reflect.Uint64:  uint64FieldFunc,
	reflect.Float32: float32FieldFunc,
	reflect.Float64: float64FieldFunc,
	reflect.String:  stringFieldFunc,
	reflect.Bool:    boolFieldFunc,
	reflect.Struct:  structFieldFunc,
	reflect.Ptr:     ptrFieldFunc,
}
