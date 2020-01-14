package dataframe

//go:generate genny -in=$GOFILE -out=nullable_vector.gen.go gen "g=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,string,bool,time.Time"

type nullablegVector struct {
	items []*g
	pType VectorPType
}

func newNullablegVector(n int, pType VectorPType) *nullablegVector {
	return &nullablegVector{items: make([]*g, n), pType: pType}
}

func (v *nullablegVector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(*g)
}

func (v *nullablegVector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(*g))
}

func (v *nullablegVector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *nullablegVector) Len() int {
	return len((*v).items)
}

func (v *nullablegVector) PrimitiveType() VectorPType {
	return (*v).pType
}
