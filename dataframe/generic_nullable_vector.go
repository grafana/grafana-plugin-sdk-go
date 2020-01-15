package dataframe

//go:generate genny -in=$GOFILE -out=nullable_vector.gen.go gen "gen=uint8,uint16,uint32,uint64,int8,int16,int32,int64,float32,float64,string,bool,time.Time"

type nullablegenVector struct {
	items []*gen
	pType VectorPType
}

func newNullablegenVector(n int, pType VectorPType) *nullablegenVector {
	return &nullablegenVector{items: make([]*gen, n), pType: pType}
}

func (v *nullablegenVector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(*gen)
}

func (v *nullablegenVector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(*gen))
}

func (v *nullablegenVector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *nullablegenVector) Len() int {
	return len((*v).items)
}

func (v *nullablegenVector) PrimitiveType() VectorPType {
	return (*v).pType
}
