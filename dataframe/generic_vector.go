package dataframe

import (
	"github.com/cheekybits/genny/generic"
)

//go:generate genny -in=$GOFILE -out=vector.gen.go gen "Generic=int64,uint64,float64,string,bool,time.Time"

type Generic generic.Type

type GenericVector struct {
	items []Generic
	pType VectorPType
}

func newGenericVector(n int, pType VectorPType) *GenericVector {
	return &GenericVector{
		items: make([]Generic, n),
		pType: pType,
	}
}

func (v *GenericVector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(Generic)
}

func (v *GenericVector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(Generic))
}

func (v *GenericVector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *GenericVector) Len() int {
	return len((*v).items)
}

func (v *GenericVector) PrimitiveType() VectorPType {
	return (*v).pType
}