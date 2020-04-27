// This file was automatically generated by genny.
// Any changes will be lost if this file is regenerated.
// see https://github.com/cheekybits/genny

package data

import "time"

//go:Uint8erate uint8ny -in=$GOFILE -out=vector.Uint8.go uint8 "Uint8=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,string,bool,time.Time,time.Duration"

type uint8Vector []uint8

func newUint8Vector(n int) *uint8Vector {
	v := uint8Vector(make([]uint8, n))
	return &v
}

func (v *uint8Vector) Set(idx int, i interface{}) {
	(*v)[idx] = i.(uint8)
}

func (v *uint8Vector) SetConcreteAt(idx int, i interface{}) {
	v.Set(idx, i)
}

func (v *uint8Vector) Append(i interface{}) {
	(*v) = append((*v), i.(uint8))
}

func (v *uint8Vector) At(i int) interface{} {
	return (*v)[i]
}

func (v *uint8Vector) PointerAt(i int) interface{} {
	return &(*v)[i]
}

func (v *uint8Vector) Len() int {
	return len((*v))
}

func (v *uint8Vector) CopyAt(i int) interface{} {
	var g uint8
	g = (*v)[i]
	return g
}

func (v *uint8Vector) ConcreteAt(i int) (interface{}, bool) {
	return v.At(i), true
}

func (v *uint8Vector) Type() FieldType {
	return vectorFieldType(v)
}

func (v *uint8Vector) Extend(i int) {
	(*v) = append((*v), make([]uint8, i)...)
}

//go:Uint16erate uint16ny -in=$GOFILE -out=vector.Uint16.go uint16 "Uint16=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,string,bool,time.Time,time.Duration"

type uint16Vector []uint16

func newUint16Vector(n int) *uint16Vector {
	v := uint16Vector(make([]uint16, n))
	return &v
}

func (v *uint16Vector) Set(idx int, i interface{}) {
	(*v)[idx] = i.(uint16)
}

func (v *uint16Vector) SetConcreteAt(idx int, i interface{}) {
	v.Set(idx, i)
}

func (v *uint16Vector) Append(i interface{}) {
	(*v) = append((*v), i.(uint16))
}

func (v *uint16Vector) At(i int) interface{} {
	return (*v)[i]
}

func (v *uint16Vector) PointerAt(i int) interface{} {
	return &(*v)[i]
}

func (v *uint16Vector) Len() int {
	return len((*v))
}

func (v *uint16Vector) CopyAt(i int) interface{} {
	var g uint16
	g = (*v)[i]
	return g
}

func (v *uint16Vector) ConcreteAt(i int) (interface{}, bool) {
	return v.At(i), true
}

func (v *uint16Vector) Type() FieldType {
	return vectorFieldType(v)
}

func (v *uint16Vector) Extend(i int) {
	(*v) = append((*v), make([]uint16, i)...)
}

//go:Uint32erate uint32ny -in=$GOFILE -out=vector.Uint32.go uint32 "Uint32=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,string,bool,time.Time,time.Duration"

type uint32Vector []uint32

func newUint32Vector(n int) *uint32Vector {
	v := uint32Vector(make([]uint32, n))
	return &v
}

func (v *uint32Vector) Set(idx int, i interface{}) {
	(*v)[idx] = i.(uint32)
}

func (v *uint32Vector) SetConcreteAt(idx int, i interface{}) {
	v.Set(idx, i)
}

func (v *uint32Vector) Append(i interface{}) {
	(*v) = append((*v), i.(uint32))
}

func (v *uint32Vector) At(i int) interface{} {
	return (*v)[i]
}

func (v *uint32Vector) PointerAt(i int) interface{} {
	return &(*v)[i]
}

func (v *uint32Vector) Len() int {
	return len((*v))
}

func (v *uint32Vector) CopyAt(i int) interface{} {
	var g uint32
	g = (*v)[i]
	return g
}

func (v *uint32Vector) ConcreteAt(i int) (interface{}, bool) {
	return v.At(i), true
}

func (v *uint32Vector) Type() FieldType {
	return vectorFieldType(v)
}

func (v *uint32Vector) Extend(i int) {
	(*v) = append((*v), make([]uint32, i)...)
}

//go:Uint64erate uint64ny -in=$GOFILE -out=vector.Uint64.go uint64 "Uint64=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,string,bool,time.Time,time.Duration"

type uint64Vector []uint64

func newUint64Vector(n int) *uint64Vector {
	v := uint64Vector(make([]uint64, n))
	return &v
}

func (v *uint64Vector) Set(idx int, i interface{}) {
	(*v)[idx] = i.(uint64)
}

func (v *uint64Vector) SetConcreteAt(idx int, i interface{}) {
	v.Set(idx, i)
}

func (v *uint64Vector) Append(i interface{}) {
	(*v) = append((*v), i.(uint64))
}

func (v *uint64Vector) At(i int) interface{} {
	return (*v)[i]
}

func (v *uint64Vector) PointerAt(i int) interface{} {
	return &(*v)[i]
}

func (v *uint64Vector) Len() int {
	return len((*v))
}

func (v *uint64Vector) CopyAt(i int) interface{} {
	var g uint64
	g = (*v)[i]
	return g
}

func (v *uint64Vector) ConcreteAt(i int) (interface{}, bool) {
	return v.At(i), true
}

func (v *uint64Vector) Type() FieldType {
	return vectorFieldType(v)
}

func (v *uint64Vector) Extend(i int) {
	(*v) = append((*v), make([]uint64, i)...)
}

//go:Int8erate int8ny -in=$GOFILE -out=vector.Int8.go int8 "Int8=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,string,bool,time.Time,time.Duration"

type int8Vector []int8

func newInt8Vector(n int) *int8Vector {
	v := int8Vector(make([]int8, n))
	return &v
}

func (v *int8Vector) Set(idx int, i interface{}) {
	(*v)[idx] = i.(int8)
}

func (v *int8Vector) SetConcreteAt(idx int, i interface{}) {
	v.Set(idx, i)
}

func (v *int8Vector) Append(i interface{}) {
	(*v) = append((*v), i.(int8))
}

func (v *int8Vector) At(i int) interface{} {
	return (*v)[i]
}

func (v *int8Vector) PointerAt(i int) interface{} {
	return &(*v)[i]
}

func (v *int8Vector) Len() int {
	return len((*v))
}

func (v *int8Vector) CopyAt(i int) interface{} {
	var g int8
	g = (*v)[i]
	return g
}

func (v *int8Vector) ConcreteAt(i int) (interface{}, bool) {
	return v.At(i), true
}

func (v *int8Vector) Type() FieldType {
	return vectorFieldType(v)
}

func (v *int8Vector) Extend(i int) {
	(*v) = append((*v), make([]int8, i)...)
}

//go:Int16erate int16ny -in=$GOFILE -out=vector.Int16.go int16 "Int16=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,string,bool,time.Time,time.Duration"

type int16Vector []int16

func newInt16Vector(n int) *int16Vector {
	v := int16Vector(make([]int16, n))
	return &v
}

func (v *int16Vector) Set(idx int, i interface{}) {
	(*v)[idx] = i.(int16)
}

func (v *int16Vector) SetConcreteAt(idx int, i interface{}) {
	v.Set(idx, i)
}

func (v *int16Vector) Append(i interface{}) {
	(*v) = append((*v), i.(int16))
}

func (v *int16Vector) At(i int) interface{} {
	return (*v)[i]
}

func (v *int16Vector) PointerAt(i int) interface{} {
	return &(*v)[i]
}

func (v *int16Vector) Len() int {
	return len((*v))
}

func (v *int16Vector) CopyAt(i int) interface{} {
	var g int16
	g = (*v)[i]
	return g
}

func (v *int16Vector) ConcreteAt(i int) (interface{}, bool) {
	return v.At(i), true
}

func (v *int16Vector) Type() FieldType {
	return vectorFieldType(v)
}

func (v *int16Vector) Extend(i int) {
	(*v) = append((*v), make([]int16, i)...)
}

//go:Int32erate int32ny -in=$GOFILE -out=vector.Int32.go int32 "Int32=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,string,bool,time.Time,time.Duration"

type int32Vector []int32

func newInt32Vector(n int) *int32Vector {
	v := int32Vector(make([]int32, n))
	return &v
}

func (v *int32Vector) Set(idx int, i interface{}) {
	(*v)[idx] = i.(int32)
}

func (v *int32Vector) SetConcreteAt(idx int, i interface{}) {
	v.Set(idx, i)
}

func (v *int32Vector) Append(i interface{}) {
	(*v) = append((*v), i.(int32))
}

func (v *int32Vector) At(i int) interface{} {
	return (*v)[i]
}

func (v *int32Vector) PointerAt(i int) interface{} {
	return &(*v)[i]
}

func (v *int32Vector) Len() int {
	return len((*v))
}

func (v *int32Vector) CopyAt(i int) interface{} {
	var g int32
	g = (*v)[i]
	return g
}

func (v *int32Vector) ConcreteAt(i int) (interface{}, bool) {
	return v.At(i), true
}

func (v *int32Vector) Type() FieldType {
	return vectorFieldType(v)
}

func (v *int32Vector) Extend(i int) {
	(*v) = append((*v), make([]int32, i)...)
}

//go:Int64erate int64ny -in=$GOFILE -out=vector.Int64.go int64 "Int64=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,string,bool,time.Time,time.Duration"

type int64Vector []int64

func newInt64Vector(n int) *int64Vector {
	v := int64Vector(make([]int64, n))
	return &v
}

func (v *int64Vector) Set(idx int, i interface{}) {
	(*v)[idx] = i.(int64)
}

func (v *int64Vector) SetConcreteAt(idx int, i interface{}) {
	v.Set(idx, i)
}

func (v *int64Vector) Append(i interface{}) {
	(*v) = append((*v), i.(int64))
}

func (v *int64Vector) At(i int) interface{} {
	return (*v)[i]
}

func (v *int64Vector) PointerAt(i int) interface{} {
	return &(*v)[i]
}

func (v *int64Vector) Len() int {
	return len((*v))
}

func (v *int64Vector) CopyAt(i int) interface{} {
	var g int64
	g = (*v)[i]
	return g
}

func (v *int64Vector) ConcreteAt(i int) (interface{}, bool) {
	return v.At(i), true
}

func (v *int64Vector) Type() FieldType {
	return vectorFieldType(v)
}

func (v *int64Vector) Extend(i int) {
	(*v) = append((*v), make([]int64, i)...)
}

//go:Float32erate float32ny -in=$GOFILE -out=vector.Float32.go float32 "Float32=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,string,bool,time.Time,time.Duration"

type float32Vector []float32

func newFloat32Vector(n int) *float32Vector {
	v := float32Vector(make([]float32, n))
	return &v
}

func (v *float32Vector) Set(idx int, i interface{}) {
	(*v)[idx] = i.(float32)
}

func (v *float32Vector) SetConcreteAt(idx int, i interface{}) {
	v.Set(idx, i)
}

func (v *float32Vector) Append(i interface{}) {
	(*v) = append((*v), i.(float32))
}

func (v *float32Vector) At(i int) interface{} {
	return (*v)[i]
}

func (v *float32Vector) PointerAt(i int) interface{} {
	return &(*v)[i]
}

func (v *float32Vector) Len() int {
	return len((*v))
}

func (v *float32Vector) CopyAt(i int) interface{} {
	var g float32
	g = (*v)[i]
	return g
}

func (v *float32Vector) ConcreteAt(i int) (interface{}, bool) {
	return v.At(i), true
}

func (v *float32Vector) Type() FieldType {
	return vectorFieldType(v)
}

func (v *float32Vector) Extend(i int) {
	(*v) = append((*v), make([]float32, i)...)
}

//go:Float64erate float64ny -in=$GOFILE -out=vector.Float64.go float64 "Float64=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,string,bool,time.Time,time.Duration"

type float64Vector []float64

func newFloat64Vector(n int) *float64Vector {
	v := float64Vector(make([]float64, n))
	return &v
}

func (v *float64Vector) Set(idx int, i interface{}) {
	(*v)[idx] = i.(float64)
}

func (v *float64Vector) SetConcreteAt(idx int, i interface{}) {
	v.Set(idx, i)
}

func (v *float64Vector) Append(i interface{}) {
	(*v) = append((*v), i.(float64))
}

func (v *float64Vector) At(i int) interface{} {
	return (*v)[i]
}

func (v *float64Vector) PointerAt(i int) interface{} {
	return &(*v)[i]
}

func (v *float64Vector) Len() int {
	return len((*v))
}

func (v *float64Vector) CopyAt(i int) interface{} {
	var g float64
	g = (*v)[i]
	return g
}

func (v *float64Vector) ConcreteAt(i int) (interface{}, bool) {
	return v.At(i), true
}

func (v *float64Vector) Type() FieldType {
	return vectorFieldType(v)
}

func (v *float64Vector) Extend(i int) {
	(*v) = append((*v), make([]float64, i)...)
}

//go:Stringerate stringny -in=$GOFILE -out=vector.String.go string "String=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,string,bool,time.Time,time.Duration"

type stringVector []string

func newStringVector(n int) *stringVector {
	v := stringVector(make([]string, n))
	return &v
}

func (v *stringVector) Set(idx int, i interface{}) {
	(*v)[idx] = i.(string)
}

func (v *stringVector) SetConcreteAt(idx int, i interface{}) {
	v.Set(idx, i)
}

func (v *stringVector) Append(i interface{}) {
	(*v) = append((*v), i.(string))
}

func (v *stringVector) At(i int) interface{} {
	return (*v)[i]
}

func (v *stringVector) PointerAt(i int) interface{} {
	return &(*v)[i]
}

func (v *stringVector) Len() int {
	return len((*v))
}

func (v *stringVector) CopyAt(i int) interface{} {
	var g string
	g = (*v)[i]
	return g
}

func (v *stringVector) ConcreteAt(i int) (interface{}, bool) {
	return v.At(i), true
}

func (v *stringVector) Type() FieldType {
	return vectorFieldType(v)
}

func (v *stringVector) Extend(i int) {
	(*v) = append((*v), make([]string, i)...)
}

//go:Boolerate boolny -in=$GOFILE -out=vector.Bool.go bool "Bool=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,string,bool,time.Time,time.Duration"

type boolVector []bool

func newBoolVector(n int) *boolVector {
	v := boolVector(make([]bool, n))
	return &v
}

func (v *boolVector) Set(idx int, i interface{}) {
	(*v)[idx] = i.(bool)
}

func (v *boolVector) SetConcreteAt(idx int, i interface{}) {
	v.Set(idx, i)
}

func (v *boolVector) Append(i interface{}) {
	(*v) = append((*v), i.(bool))
}

func (v *boolVector) At(i int) interface{} {
	return (*v)[i]
}

func (v *boolVector) PointerAt(i int) interface{} {
	return &(*v)[i]
}

func (v *boolVector) Len() int {
	return len((*v))
}

func (v *boolVector) CopyAt(i int) interface{} {
	var g bool
	g = (*v)[i]
	return g
}

func (v *boolVector) ConcreteAt(i int) (interface{}, bool) {
	return v.At(i), true
}

func (v *boolVector) Type() FieldType {
	return vectorFieldType(v)
}

func (v *boolVector) Extend(i int) {
	(*v) = append((*v), make([]bool, i)...)
}

//go:TimeTimeerate timeTimeny -in=$GOFILE -out=vector.TimeTime.go time.Time "TimeTime=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,string,bool,time.Time,time.Duration"

type timeTimeVector []time.Time

func newTimeTimeVector(n int) *timeTimeVector {
	v := timeTimeVector(make([]time.Time, n))
	return &v
}

func (v *timeTimeVector) Set(idx int, i interface{}) {
	(*v)[idx] = i.(time.Time)
}

func (v *timeTimeVector) SetConcreteAt(idx int, i interface{}) {
	v.Set(idx, i)
}

func (v *timeTimeVector) Append(i interface{}) {
	(*v) = append((*v), i.(time.Time))
}

func (v *timeTimeVector) At(i int) interface{} {
	return (*v)[i]
}

func (v *timeTimeVector) PointerAt(i int) interface{} {
	return &(*v)[i]
}

func (v *timeTimeVector) Len() int {
	return len((*v))
}

func (v *timeTimeVector) CopyAt(i int) interface{} {
	var g time.Time
	g = (*v)[i]
	return g
}

func (v *timeTimeVector) ConcreteAt(i int) (interface{}, bool) {
	return v.At(i), true
}

func (v *timeTimeVector) Type() FieldType {
	return vectorFieldType(v)
}

func (v *timeTimeVector) Extend(i int) {
	(*v) = append((*v), make([]time.Time, i)...)
}

//go:TimeDurationerate timeDurationny -in=$GOFILE -out=vector.TimeDuration.go time.Duration "TimeDuration=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,string,bool,time.Time,time.Duration"

type timeDurationVector []time.Duration

func newTimeDurationVector(n int) *timeDurationVector {
	v := timeDurationVector(make([]time.Duration, n))
	return &v
}

func (v *timeDurationVector) Set(idx int, i interface{}) {
	(*v)[idx] = i.(time.Duration)
}

func (v *timeDurationVector) SetConcreteAt(idx int, i interface{}) {
	v.Set(idx, i)
}

func (v *timeDurationVector) Append(i interface{}) {
	(*v) = append((*v), i.(time.Duration))
}

func (v *timeDurationVector) At(i int) interface{} {
	return (*v)[i]
}

func (v *timeDurationVector) PointerAt(i int) interface{} {
	return &(*v)[i]
}

func (v *timeDurationVector) Len() int {
	return len((*v))
}

func (v *timeDurationVector) CopyAt(i int) interface{} {
	var g time.Duration
	g = (*v)[i]
	return g
}

func (v *timeDurationVector) ConcreteAt(i int) (interface{}, bool) {
	return v.At(i), true
}

func (v *timeDurationVector) Type() FieldType {
	return vectorFieldType(v)
}

func (v *timeDurationVector) Extend(i int) {
	(*v) = append((*v), make([]time.Duration, i)...)
}
