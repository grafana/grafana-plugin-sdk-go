package data

// this supports the enum type
// it is diffent than the rest since it is backed by
// a uint16, but has special semantics and interacts with the metadata
// Unlike the other fields it can not be easily generated

// NOTE: no implementation for newEnumVectorWithValues -- since

type enumVector []uint16

func newEnumVector(n int) *enumVector {
	v := enumVector(make([]uint16, n))
	return &v
}

func (v *enumVector) Set(idx int, i interface{}) {
	(*v)[idx] = i.(uint16)
}

func (v *enumVector) SetConcrete(idx int, i interface{}) {
	v.Set(idx, i)
}

func (v *enumVector) Append(i interface{}) {
	*v = append(*v, i.(uint16))
}

func (v *enumVector) At(i int) interface{} {
	return (*v)[i]
}

func (v *enumVector) PointerAt(i int) interface{} {
	return &(*v)[i]
}

func (v *enumVector) Len() int {
	return len(*v)
}

func (v *enumVector) CopyAt(i int) interface{} {
	var g uint16
	g = (*v)[i]
	return g
}

func (v *enumVector) ConcreteAt(i int) (interface{}, bool) {
	return v.At(i), true
}

func (v *enumVector) Type() FieldType {
	return vectorFieldType(v)
}

func (v *enumVector) Extend(i int) {
	*v = append(*v, make([]uint16, i)...)
}

func (v *enumVector) Insert(i int, val interface{}) {
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

func (v *enumVector) Delete(i int) {
	*v = append((*v)[:i], (*v)[i+1:]...)
}

type nullableEnumVector []*uint16

func newNullableEnumVector(n int) *nullableEnumVector {
	v := nullableEnumVector(make([]*uint16, n))
	return &v
}

func (v *nullableEnumVector) Set(idx int, i interface{}) {
	if i == nil {
		(*v)[idx] = nil
		return
	}
	(*v)[idx] = i.(*uint16)
}

func (v *nullableEnumVector) SetConcrete(idx int, i interface{}) {
	val := i.(uint16)
	(*v)[idx] = &val
}

func (v *nullableEnumVector) Append(i interface{}) {
	if i == nil {
		*v = append(*v, nil)
		return
	}
	*v = append(*v, i.(*uint16))
}

func (v *nullableEnumVector) At(i int) interface{} {
	return (*v)[i]
}

func (v *nullableEnumVector) CopyAt(i int) interface{} {
	if (*v)[i] == nil {
		var g *uint16
		return g
	}
	var g uint16
	g = *(*v)[i]
	return &g
}

func (v *nullableEnumVector) ConcreteAt(i int) (interface{}, bool) {
	var g uint16
	val := (*v)[i]
	if val == nil {
		return g, false
	}
	g = *val
	return g, true
}

func (v *nullableEnumVector) PointerAt(i int) interface{} {
	return &(*v)[i]
}

func (v *nullableEnumVector) Len() int {
	return len(*v)
}

func (v *nullableEnumVector) Type() FieldType {
	return vectorFieldType(v)
}

func (v *nullableEnumVector) Extend(i int) {
	*v = append(*v, make([]*uint16, i)...)
}

func (v *nullableEnumVector) Insert(i int, val interface{}) {
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

func (v *nullableEnumVector) Delete(i int) {
	*v = append((*v)[:i], (*v)[i+1:]...)
}
