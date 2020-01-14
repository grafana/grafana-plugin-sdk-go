package dataframe

import (
	"github.com/cheekybits/genny/generic"
)

//go:generate genny -in=$GOFILE -out=vector.gen.go gen "g=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,string,bool,time.Time"

type g generic.Type

type gVector struct {
	items []g
	pType VectorPType
}

func newgVector(n int, pType VectorPType) *gVector {
	return &gVector{
		items: make([]g, n),
		pType: pType,
	}
}

func (v *gVector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(g)
}

func (v *gVector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(g))
}

func (v *gVector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *gVector) Len() int {
	return len((*v).items)
}

func (v *gVector) PrimitiveType() VectorPType {
	return (*v).pType
}
