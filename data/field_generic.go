package data

// NewFieldGeneric creates a new Field using generic vectors for better performance.
// This eliminates interface{} boxing overhead for typed operations.
func NewFieldGeneric[T any](name string, labels Labels, values []T) *Field {
	vec := newGenericVectorWithValues(values)
	return &Field{
		Name:   name,
		vector: vec,
		Labels: labels,
	}
}

// NewFieldGenericNullable creates a new Field using nullable generic vectors.
func NewFieldGenericNullable[T any](name string, labels Labels, values []*T) *Field {
	vec := newNullableGenericVectorWithValues(values)
	return &Field{
		Name:   name,
		vector: vec,
		Labels: labels,
	}
}

// AtTyped returns the value at index idx with zero allocation.
// This method panics if the Field's underlying vector is not a genericVector[T].
// For optimal performance, use this with fields created via NewFieldGeneric.
func AtTyped[T any](f *Field, idx int) T {
	if gv, ok := f.vector.(*genericVector[T]); ok {
		return gv.AtTyped(idx)
	}
	panic("Field is not backed by genericVector[T]")
}

// SetTyped sets the value at index idx with zero allocation.
// This method panics if the Field's underlying vector is not a genericVector[T].
func SetTyped[T any](f *Field, idx int, val T) {
	if gv, ok := f.vector.(*genericVector[T]); ok {
		gv.SetTyped(idx, val)
		return
	}
	panic("Field is not backed by genericVector[T]")
}

// AppendTyped appends a value with zero allocation.
// This method panics if the Field's underlying vector is not a genericVector[T].
func AppendTyped[T any](f *Field, val T) {
	if gv, ok := f.vector.(*genericVector[T]); ok {
		gv.AppendTyped(val)
		return
	}
	panic("Field is not backed by genericVector[T]")
}

// AtTypedNullable returns the pointer value at index idx with zero allocation.
// This method panics if the Field's underlying vector is not a nullableGenericVector[T].
func AtTypedNullable[T any](f *Field, idx int) *T {
	if gv, ok := f.vector.(*nullableGenericVector[T]); ok {
		return gv.AtTyped(idx)
	}
	panic("Field is not backed by nullableGenericVector[T]")
}

// SetTypedNullable sets the pointer value at index idx with zero allocation.
// This method panics if the Field's underlying vector is not a nullableGenericVector[T].
func SetTypedNullable[T any](f *Field, idx int, val *T) {
	if gv, ok := f.vector.(*nullableGenericVector[T]); ok {
		gv.SetTyped(idx, val)
		return
	}
	panic("Field is not backed by nullableGenericVector[T]")
}

// ConcreteAtTyped returns the dereferenced value for nullable fields with minimal allocation.
// The second return value indicates if the value was non-nil.
func ConcreteAtTyped[T any](f *Field, idx int) (T, bool) {
	switch vec := f.vector.(type) {
	case *nullableGenericVector[T]:
		return vec.ConcreteAtTyped(idx)
	case *genericVector[T]:
		return vec.AtTyped(idx), true
	default:
		var zero T
		return zero, false
	}
}

// IsgenericVector returns true if the field is backed by a genericVector.
func (f *Field) IsgenericVector() bool {
	switch f.vector.(type) {
	case *genericVector[int8], *genericVector[int16], *genericVector[int32], *genericVector[int64]:
		return true
	case *genericVector[uint8], *genericVector[uint16], *genericVector[uint32], *genericVector[uint64]:
		return true
	case *genericVector[float32], *genericVector[float64]:
		return true
	case *genericVector[string], *genericVector[bool]:
		return true
	case *nullableGenericVector[int8], *nullableGenericVector[int16], *nullableGenericVector[int32], *nullableGenericVector[int64]:
		return true
	case *nullableGenericVector[uint8], *nullableGenericVector[uint16], *nullableGenericVector[uint32], *nullableGenericVector[uint64]:
		return true
	case *nullableGenericVector[float32], *nullableGenericVector[float64]:
		return true
	case *nullableGenericVector[string], *nullableGenericVector[bool]:
		return true
	case *genericVector[EnumItemIndex], *nullableGenericVector[EnumItemIndex]:
		return true
	}
	return false
}
