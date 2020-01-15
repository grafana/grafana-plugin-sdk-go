package dataframe

import (
	"github.com/cheekybits/genny/generic"
)

//go:generate genny -in=$GOFILE -out=vector.gen.go gen "gen=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,string,bool,time.Time"

type gen generic.Type

type genVector struct {
	items []gen
	pType VectorPType
}

func newgenVector(n int, pType VectorPType) *genVector {
	return &genVector{
		items: make([]gen, n),
		pType: pType,
	}
}

func (v *genVector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(gen)
}

func (v *genVector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(gen))
}

func (v *genVector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *genVector) Len() int {
	return len((*v).items)
}

func (v *genVector) PrimitiveType() VectorPType {
	return (*v).pType
}
