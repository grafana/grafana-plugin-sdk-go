package data

// This is an array of frames -- always nillable

type dataFrameVector []*Frame

func newDataFrameVector(n int) *dataFrameVector {
	v := dataFrameVector(make([]*Frame, n))
	return &v
}

func (v *dataFrameVector) Set(idx int, i interface{}) {
	(*v)[idx] = i.(*Frame)
}

func (v *dataFrameVector) SetConcrete(idx int, i interface{}) {
	v.Set(idx, i)
}

func (v *dataFrameVector) Append(i interface{}) {
	*v = append(*v, i.(*Frame))
}

func (v *dataFrameVector) At(i int) interface{} {
	return (*v)[i]
}

func (v *dataFrameVector) PointerAt(i int) interface{} {
	return &(*v)[i]
}

func (v *dataFrameVector) Len() int {
	return len(*v)
}

func (v *dataFrameVector) CopyAt(i int) interface{} {
	return (*v)[i]
}

func (v *dataFrameVector) ConcreteAt(i int) (interface{}, bool) {
	return v.At(i), true
}

func (v *dataFrameVector) Type() FieldType {
	return vectorFieldType(v)
}

func (v *dataFrameVector) Extend(i int) {
	*v = append(*v, make([]*Frame, i)...)
}

func (v *dataFrameVector) Insert(i int, val interface{}) {
	switch {
	case i < v.Len():
		v.Extend(1)
		copy((*v)[i+1:], (*v)[i:])
		v.Set(i, val)
	case i == v.Len():
		v.Append(val)
	case i > v.Len():
		panic("Invalid index; vector length should be greater or equal to that index")
	}
}

func (v *dataFrameVector) Delete(i int) {
	*v = append((*v)[:i], (*v)[i+1:]...)
}
