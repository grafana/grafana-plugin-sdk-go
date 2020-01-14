// This file was automatically generated by genny.
// Any changes will be lost if this file is regenerated.
// see https://github.com/cheekybits/genny

package dataframe

import "time"

//Uint8o:Uint8enerate uint8enny -in=$GOFILE -out=vector.Uint8en.Uint8o uint8en "Uint8=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,strinUint8,bool,time.Time"

type uint8Vector struct {
	items []uint8
	pType VectorPType
}

func newUint8Vector(n int, pType VectorPType) *uint8Vector {
	return &uint8Vector{
		items: make([]uint8, n),
		pType: pType,
	}
}

func (v *uint8Vector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(uint8)
}

func (v *uint8Vector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(uint8))
}

func (v *uint8Vector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *uint8Vector) Len() int {
	return len((*v).items)
}

func (v *uint8Vector) PrimitiveType() VectorPType {
	return (*v).pType
}

//Uint16o:Uint16enerate uint16enny -in=$GOFILE -out=vector.Uint16en.Uint16o uint16en "Uint16=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,strinUint16,bool,time.Time"

type uint16Vector struct {
	items []uint16
	pType VectorPType
}

func newUint16Vector(n int, pType VectorPType) *uint16Vector {
	return &uint16Vector{
		items: make([]uint16, n),
		pType: pType,
	}
}

func (v *uint16Vector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(uint16)
}

func (v *uint16Vector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(uint16))
}

func (v *uint16Vector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *uint16Vector) Len() int {
	return len((*v).items)
}

func (v *uint16Vector) PrimitiveType() VectorPType {
	return (*v).pType
}

//Uint32o:Uint32enerate uint32enny -in=$GOFILE -out=vector.Uint32en.Uint32o uint32en "Uint32=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,strinUint32,bool,time.Time"

type uint32Vector struct {
	items []uint32
	pType VectorPType
}

func newUint32Vector(n int, pType VectorPType) *uint32Vector {
	return &uint32Vector{
		items: make([]uint32, n),
		pType: pType,
	}
}

func (v *uint32Vector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(uint32)
}

func (v *uint32Vector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(uint32))
}

func (v *uint32Vector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *uint32Vector) Len() int {
	return len((*v).items)
}

func (v *uint32Vector) PrimitiveType() VectorPType {
	return (*v).pType
}

//Uint64o:Uint64enerate uint64enny -in=$GOFILE -out=vector.Uint64en.Uint64o uint64en "Uint64=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,strinUint64,bool,time.Time"

type uint64Vector struct {
	items []uint64
	pType VectorPType
}

func newUint64Vector(n int, pType VectorPType) *uint64Vector {
	return &uint64Vector{
		items: make([]uint64, n),
		pType: pType,
	}
}

func (v *uint64Vector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(uint64)
}

func (v *uint64Vector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(uint64))
}

func (v *uint64Vector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *uint64Vector) Len() int {
	return len((*v).items)
}

func (v *uint64Vector) PrimitiveType() VectorPType {
	return (*v).pType
}

//Int8o:Int8enerate int8enny -in=$GOFILE -out=vector.Int8en.Int8o int8en "Int8=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,strinInt8,bool,time.Time"

type int8Vector struct {
	items []int8
	pType VectorPType
}

func newInt8Vector(n int, pType VectorPType) *int8Vector {
	return &int8Vector{
		items: make([]int8, n),
		pType: pType,
	}
}

func (v *int8Vector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(int8)
}

func (v *int8Vector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(int8))
}

func (v *int8Vector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *int8Vector) Len() int {
	return len((*v).items)
}

func (v *int8Vector) PrimitiveType() VectorPType {
	return (*v).pType
}

//Int16o:Int16enerate int16enny -in=$GOFILE -out=vector.Int16en.Int16o int16en "Int16=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,strinInt16,bool,time.Time"

type int16Vector struct {
	items []int16
	pType VectorPType
}

func newInt16Vector(n int, pType VectorPType) *int16Vector {
	return &int16Vector{
		items: make([]int16, n),
		pType: pType,
	}
}

func (v *int16Vector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(int16)
}

func (v *int16Vector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(int16))
}

func (v *int16Vector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *int16Vector) Len() int {
	return len((*v).items)
}

func (v *int16Vector) PrimitiveType() VectorPType {
	return (*v).pType
}

//Int32o:Int32enerate int32enny -in=$GOFILE -out=vector.Int32en.Int32o int32en "Int32=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,strinInt32,bool,time.Time"

type int32Vector struct {
	items []int32
	pType VectorPType
}

func newInt32Vector(n int, pType VectorPType) *int32Vector {
	return &int32Vector{
		items: make([]int32, n),
		pType: pType,
	}
}

func (v *int32Vector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(int32)
}

func (v *int32Vector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(int32))
}

func (v *int32Vector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *int32Vector) Len() int {
	return len((*v).items)
}

func (v *int32Vector) PrimitiveType() VectorPType {
	return (*v).pType
}

//Int64o:Int64enerate int64enny -in=$GOFILE -out=vector.Int64en.Int64o int64en "Int64=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,strinInt64,bool,time.Time"

type int64Vector struct {
	items []int64
	pType VectorPType
}

func newInt64Vector(n int, pType VectorPType) *int64Vector {
	return &int64Vector{
		items: make([]int64, n),
		pType: pType,
	}
}

func (v *int64Vector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(int64)
}

func (v *int64Vector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(int64))
}

func (v *int64Vector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *int64Vector) Len() int {
	return len((*v).items)
}

func (v *int64Vector) PrimitiveType() VectorPType {
	return (*v).pType
}

//Float32o:Float32enerate float32enny -in=$GOFILE -out=vector.Float32en.Float32o float32en "Float32=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,strinFloat32,bool,time.Time"

type float32Vector struct {
	items []float32
	pType VectorPType
}

func newFloat32Vector(n int, pType VectorPType) *float32Vector {
	return &float32Vector{
		items: make([]float32, n),
		pType: pType,
	}
}

func (v *float32Vector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(float32)
}

func (v *float32Vector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(float32))
}

func (v *float32Vector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *float32Vector) Len() int {
	return len((*v).items)
}

func (v *float32Vector) PrimitiveType() VectorPType {
	return (*v).pType
}

//Float64o:Float64enerate float64enny -in=$GOFILE -out=vector.Float64en.Float64o float64en "Float64=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,strinFloat64,bool,time.Time"

type float64Vector struct {
	items []float64
	pType VectorPType
}

func newFloat64Vector(n int, pType VectorPType) *float64Vector {
	return &float64Vector{
		items: make([]float64, n),
		pType: pType,
	}
}

func (v *float64Vector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(float64)
}

func (v *float64Vector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(float64))
}

func (v *float64Vector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *float64Vector) Len() int {
	return len((*v).items)
}

func (v *float64Vector) PrimitiveType() VectorPType {
	return (*v).pType
}

//Stringo:Stringenerate stringenny -in=$GOFILE -out=vector.Stringen.Stringo stringen "String=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,strinString,bool,time.Time"

type stringVector struct {
	items []string
	pType VectorPType
}

func newStringVector(n int, pType VectorPType) *stringVector {
	return &stringVector{
		items: make([]string, n),
		pType: pType,
	}
}

func (v *stringVector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(string)
}

func (v *stringVector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(string))
}

func (v *stringVector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *stringVector) Len() int {
	return len((*v).items)
}

func (v *stringVector) PrimitiveType() VectorPType {
	return (*v).pType
}

//Boolo:Boolenerate boolenny -in=$GOFILE -out=vector.Boolen.Boolo boolen "Bool=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,strinBool,bool,time.Time"

type boolVector struct {
	items []bool
	pType VectorPType
}

func newBoolVector(n int, pType VectorPType) *boolVector {
	return &boolVector{
		items: make([]bool, n),
		pType: pType,
	}
}

func (v *boolVector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(bool)
}

func (v *boolVector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(bool))
}

func (v *boolVector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *boolVector) Len() int {
	return len((*v).items)
}

func (v *boolVector) PrimitiveType() VectorPType {
	return (*v).pType
}

//TimeTimeo:TimeTimeenerate timeTimeenny -in=$GOFILE -out=vector.TimeTimeen.TimeTimeo timeTimeen "TimeTime=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,strinTimeTime,bool,time.Time"

type timeTimeVector struct {
	items []time.Time
	pType VectorPType
}

func newTimeTimeVector(n int, pType VectorPType) *timeTimeVector {
	return &timeTimeVector{
		items: make([]time.Time, n),
		pType: pType,
	}
}

func (v *timeTimeVector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(time.Time)
}

func (v *timeTimeVector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(time.Time))
}

func (v *timeTimeVector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *timeTimeVector) Len() int {
	return len((*v).items)
}

func (v *timeTimeVector) PrimitiveType() VectorPType {
	return (*v).pType
}
