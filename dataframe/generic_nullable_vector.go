package dataframe

//go:generate genny -in=$GOFILE -out=nullable_vector.gen.go gen "Generic=int64,uint64,float64,string,bool,time.Time"

type nullableGenericVector struct {
	items []*Generic
	pType VectorPType
}

func newNullableGenericVector(n int, pType VectorPType) *nullableGenericVector {
	return &nullableGenericVector{items: make([]*Generic, n), pType: pType}
}

func (v *nullableGenericVector) Set(idx int, i interface{}) {
	(*v).items[idx] = i.(*Generic)
}

func (v *nullableGenericVector) Append(i interface{}) {
	(*v).items = append((*v).items, i.(*Generic))
}

func (v *nullableGenericVector) At(i int) interface{} {
	return (*v).items[i]
}

func (v *nullableGenericVector) Len() int {
	return len((*v).items)
}

func (v *nullableGenericVector) PrimitiveType() VectorPType {
	return (*v).pType
}