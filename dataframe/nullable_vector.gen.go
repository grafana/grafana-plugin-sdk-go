// This file was automatically generated by genny.
// Any changes will be lost if this file is regenerated.
// see https://github.com/cheekybits/genny

package dataframe

import "time"

//Uint8o:Uint8enerate uint8enny -in=$GOFILE -out=nullable_vector.Uint8en.Uint8o uint8en "Uint8=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,strinUint8,bool,time.Time"

type nullableUint8Vector struct {
	items []*uint8
	pType VectorPType
}

func newNullableUint8Vector(n int, pType VectorPType) *nullableUint8Vector {
	return &nullableUint8Vector{items: make([]*uint8, n), pType: pType}
}

func (v *nullableUint8Vector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(*uint8)
}

func (v *nullableUint8Vector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(*uint8))
}

func (v *nullableUint8Vector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *nullableUint8Vector) Len() int {
	return len((*v).items)
}

func (v *nullableUint8Vector) PrimitiveType() VectorPType {
	return (*v).pType
}

//Uint16o:Uint16enerate uint16enny -in=$GOFILE -out=nullable_vector.Uint16en.Uint16o uint16en "Uint16=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,strinUint16,bool,time.Time"

type nullableUint16Vector struct {
	items []*uint16
	pType VectorPType
}

func newNullableUint16Vector(n int, pType VectorPType) *nullableUint16Vector {
	return &nullableUint16Vector{items: make([]*uint16, n), pType: pType}
}

func (v *nullableUint16Vector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(*uint16)
}

func (v *nullableUint16Vector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(*uint16))
}

func (v *nullableUint16Vector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *nullableUint16Vector) Len() int {
	return len((*v).items)
}

func (v *nullableUint16Vector) PrimitiveType() VectorPType {
	return (*v).pType
}

//Uint32o:Uint32enerate uint32enny -in=$GOFILE -out=nullable_vector.Uint32en.Uint32o uint32en "Uint32=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,strinUint32,bool,time.Time"

type nullableUint32Vector struct {
	items []*uint32
	pType VectorPType
}

func newNullableUint32Vector(n int, pType VectorPType) *nullableUint32Vector {
	return &nullableUint32Vector{items: make([]*uint32, n), pType: pType}
}

func (v *nullableUint32Vector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(*uint32)
}

func (v *nullableUint32Vector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(*uint32))
}

func (v *nullableUint32Vector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *nullableUint32Vector) Len() int {
	return len((*v).items)
}

func (v *nullableUint32Vector) PrimitiveType() VectorPType {
	return (*v).pType
}

//Uint64o:Uint64enerate uint64enny -in=$GOFILE -out=nullable_vector.Uint64en.Uint64o uint64en "Uint64=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,strinUint64,bool,time.Time"

type nullableUint64Vector struct {
	items []*uint64
	pType VectorPType
}

func newNullableUint64Vector(n int, pType VectorPType) *nullableUint64Vector {
	return &nullableUint64Vector{items: make([]*uint64, n), pType: pType}
}

func (v *nullableUint64Vector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(*uint64)
}

func (v *nullableUint64Vector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(*uint64))
}

func (v *nullableUint64Vector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *nullableUint64Vector) Len() int {
	return len((*v).items)
}

func (v *nullableUint64Vector) PrimitiveType() VectorPType {
	return (*v).pType
}

//Int8o:Int8enerate int8enny -in=$GOFILE -out=nullable_vector.Int8en.Int8o int8en "Int8=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,strinInt8,bool,time.Time"

type nullableInt8Vector struct {
	items []*int8
	pType VectorPType
}

func newNullableInt8Vector(n int, pType VectorPType) *nullableInt8Vector {
	return &nullableInt8Vector{items: make([]*int8, n), pType: pType}
}

func (v *nullableInt8Vector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(*int8)
}

func (v *nullableInt8Vector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(*int8))
}

func (v *nullableInt8Vector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *nullableInt8Vector) Len() int {
	return len((*v).items)
}

func (v *nullableInt8Vector) PrimitiveType() VectorPType {
	return (*v).pType
}

//Int16o:Int16enerate int16enny -in=$GOFILE -out=nullable_vector.Int16en.Int16o int16en "Int16=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,strinInt16,bool,time.Time"

type nullableInt16Vector struct {
	items []*int16
	pType VectorPType
}

func newNullableInt16Vector(n int, pType VectorPType) *nullableInt16Vector {
	return &nullableInt16Vector{items: make([]*int16, n), pType: pType}
}

func (v *nullableInt16Vector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(*int16)
}

func (v *nullableInt16Vector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(*int16))
}

func (v *nullableInt16Vector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *nullableInt16Vector) Len() int {
	return len((*v).items)
}

func (v *nullableInt16Vector) PrimitiveType() VectorPType {
	return (*v).pType
}

//Int32o:Int32enerate int32enny -in=$GOFILE -out=nullable_vector.Int32en.Int32o int32en "Int32=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,strinInt32,bool,time.Time"

type nullableInt32Vector struct {
	items []*int32
	pType VectorPType
}

func newNullableInt32Vector(n int, pType VectorPType) *nullableInt32Vector {
	return &nullableInt32Vector{items: make([]*int32, n), pType: pType}
}

func (v *nullableInt32Vector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(*int32)
}

func (v *nullableInt32Vector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(*int32))
}

func (v *nullableInt32Vector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *nullableInt32Vector) Len() int {
	return len((*v).items)
}

func (v *nullableInt32Vector) PrimitiveType() VectorPType {
	return (*v).pType
}

//Int64o:Int64enerate int64enny -in=$GOFILE -out=nullable_vector.Int64en.Int64o int64en "Int64=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,strinInt64,bool,time.Time"

type nullableInt64Vector struct {
	items []*int64
	pType VectorPType
}

func newNullableInt64Vector(n int, pType VectorPType) *nullableInt64Vector {
	return &nullableInt64Vector{items: make([]*int64, n), pType: pType}
}

func (v *nullableInt64Vector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(*int64)
}

func (v *nullableInt64Vector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(*int64))
}

func (v *nullableInt64Vector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *nullableInt64Vector) Len() int {
	return len((*v).items)
}

func (v *nullableInt64Vector) PrimitiveType() VectorPType {
	return (*v).pType
}

//Float32o:Float32enerate float32enny -in=$GOFILE -out=nullable_vector.Float32en.Float32o float32en "Float32=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,strinFloat32,bool,time.Time"

type nullableFloat32Vector struct {
	items []*float32
	pType VectorPType
}

func newNullableFloat32Vector(n int, pType VectorPType) *nullableFloat32Vector {
	return &nullableFloat32Vector{items: make([]*float32, n), pType: pType}
}

func (v *nullableFloat32Vector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(*float32)
}

func (v *nullableFloat32Vector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(*float32))
}

func (v *nullableFloat32Vector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *nullableFloat32Vector) Len() int {
	return len((*v).items)
}

func (v *nullableFloat32Vector) PrimitiveType() VectorPType {
	return (*v).pType
}

//Float64o:Float64enerate float64enny -in=$GOFILE -out=nullable_vector.Float64en.Float64o float64en "Float64=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,strinFloat64,bool,time.Time"

type nullableFloat64Vector struct {
	items []*float64
	pType VectorPType
}

func newNullableFloat64Vector(n int, pType VectorPType) *nullableFloat64Vector {
	return &nullableFloat64Vector{items: make([]*float64, n), pType: pType}
}

func (v *nullableFloat64Vector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(*float64)
}

func (v *nullableFloat64Vector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(*float64))
}

func (v *nullableFloat64Vector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *nullableFloat64Vector) Len() int {
	return len((*v).items)
}

func (v *nullableFloat64Vector) PrimitiveType() VectorPType {
	return (*v).pType
}

//Stringo:Stringenerate stringenny -in=$GOFILE -out=nullable_vector.Stringen.Stringo stringen "String=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,strinString,bool,time.Time"

type nullableStringVector struct {
	items []*string
	pType VectorPType
}

func newNullableStringVector(n int, pType VectorPType) *nullableStringVector {
	return &nullableStringVector{items: make([]*string, n), pType: pType}
}

func (v *nullableStringVector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(*string)
}

func (v *nullableStringVector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(*string))
}

func (v *nullableStringVector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *nullableStringVector) Len() int {
	return len((*v).items)
}

func (v *nullableStringVector) PrimitiveType() VectorPType {
	return (*v).pType
}

//Boolo:Boolenerate boolenny -in=$GOFILE -out=nullable_vector.Boolen.Boolo boolen "Bool=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,strinBool,bool,time.Time"

type nullableBoolVector struct {
	items []*bool
	pType VectorPType
}

func newNullableBoolVector(n int, pType VectorPType) *nullableBoolVector {
	return &nullableBoolVector{items: make([]*bool, n), pType: pType}
}

func (v *nullableBoolVector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(*bool)
}

func (v *nullableBoolVector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(*bool))
}

func (v *nullableBoolVector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *nullableBoolVector) Len() int {
	return len((*v).items)
}

func (v *nullableBoolVector) PrimitiveType() VectorPType {
	return (*v).pType
}

//TimeTimeo:TimeTimeenerate timeTimeenny -in=$GOFILE -out=nullable_vector.TimeTimeen.TimeTimeo timeTimeen "TimeTime=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,strinTimeTime,bool,time.Time"

type nullableTimeTimeVector struct {
	items []*time.Time
	pType VectorPType
}

func newNullableTimeTimeVector(n int, pType VectorPType) *nullableTimeTimeVector {
	return &nullableTimeTimeVector{items: make([]*time.Time, n), pType: pType}
}

func (v *nullableTimeTimeVector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(*time.Time)
}

func (v *nullableTimeTimeVector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(*time.Time))
}

func (v *nullableTimeTimeVector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *nullableTimeTimeVector) Len() int {
	return len((*v).items)
}

func (v *nullableTimeTimeVector) PrimitiveType() VectorPType {
	return (*v).pType
}
