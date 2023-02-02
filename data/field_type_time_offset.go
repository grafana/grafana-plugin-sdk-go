package data

// this supports a time offset type

type timeOffsetVector []int64

func newTimeOffsetVector(n int) *timeOffsetVector {
	v := timeOffsetVector(make([]int64, n))
	return &v
}

func (v *timeOffsetVector) Set(idx int, i interface{}) {
	(*v)[idx] = i.(int64)
}

func (v *timeOffsetVector) SetConcrete(idx int, i interface{}) {
	v.Set(idx, i)
}

func (v *timeOffsetVector) Append(i interface{}) {
	*v = append(*v, i.(int64))
}

func (v *timeOffsetVector) At(i int) interface{} {
	return (*v)[i]
}

func (v *timeOffsetVector) PointerAt(i int) interface{} {
	return &(*v)[i]
}

func (v *timeOffsetVector) Len() int {
	return len(*v)
}

func (v *timeOffsetVector) CopyAt(i int) interface{} {
	return (*v)[i]
}

func (v *timeOffsetVector) ConcreteAt(i int) (interface{}, bool) {
	return v.At(i), true
}

func (v *timeOffsetVector) Type() FieldType {
	return vectorFieldType(v)
}

func (v *timeOffsetVector) Extend(i int) {
	*v = append(*v, make([]int64, i)...)
}

func (v *timeOffsetVector) Insert(i int, val interface{}) {
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

func (v *timeOffsetVector) Delete(i int) {
	*v = append((*v)[:i], (*v)[i+1:]...)
}

type nullableTimeOffsetVector []*int64

func newNullableTimeOffsetVector(n int) *nullableTimeOffsetVector {
	v := nullableTimeOffsetVector(make([]*int64, n))
	return &v
}

func (v *nullableTimeOffsetVector) Set(idx int, i interface{}) {
	if i == nil {
		(*v)[idx] = nil
		return
	}
	(*v)[idx] = i.(*int64)
}

func (v *nullableTimeOffsetVector) SetConcrete(idx int, i interface{}) {
	val := i.(int64)
	(*v)[idx] = &val
}

func (v *nullableTimeOffsetVector) Append(i interface{}) {
	if i == nil {
		*v = append(*v, nil)
		return
	}
	*v = append(*v, i.(*int64))
}

func (v *nullableTimeOffsetVector) At(i int) interface{} {
	return (*v)[i]
}

func (v *nullableTimeOffsetVector) CopyAt(i int) interface{} {
	if (*v)[i] == nil {
		var g *int64
		return g
	}
	g := *(*v)[i]
	return &g
}

func (v *nullableTimeOffsetVector) ConcreteAt(i int) (interface{}, bool) {
	var g int64
	val := (*v)[i]
	if val == nil {
		return g, false
	}
	g = *val
	return g, true
}

func (v *nullableTimeOffsetVector) PointerAt(i int) interface{} {
	return &(*v)[i]
}

func (v *nullableTimeOffsetVector) Len() int {
	return len(*v)
}

func (v *nullableTimeOffsetVector) Type() FieldType {
	return vectorFieldType(v)
}

func (v *nullableTimeOffsetVector) Extend(i int) {
	*v = append(*v, make([]*int64, i)...)
}

func (v *nullableTimeOffsetVector) Insert(i int, val interface{}) {
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

func (v *nullableTimeOffsetVector) Delete(i int) {
	*v = append((*v)[:i], (*v)[i+1:]...)
}
