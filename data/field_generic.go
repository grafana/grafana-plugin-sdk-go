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

// NewFieldGenericWithCapacity creates a new Field backed by a generic vector with length 0
// and the given capacity. Use this when the final size is known (e.g. a Prometheus matrix
// response with a known sample count) and values will be appended via AppendTyped.
// Pre-sizing avoids the repeated slice doubling that otherwise dominates allocation for
// large series.
func NewFieldGenericWithCapacity[T any](name string, labels Labels, capacity int) *Field {
	return &Field{
		Name:   name,
		vector: newGenericVectorWithCapacity[T](capacity),
		Labels: labels,
	}
}

// NewFieldGenericNullableWithCapacity is the nullable-vector counterpart to
// NewFieldGenericWithCapacity.
func NewFieldGenericNullableWithCapacity[T any](name string, labels Labels, capacity int) *Field {
	return &Field{
		Name:   name,
		vector: newNullableGenericVectorWithCapacity[T](capacity),
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
// Panics if the Field is not backed by genericVector[T] or nullableGenericVector[T],
// matching the error style of AtTyped/SetTyped/AppendTyped.
func ConcreteAtTyped[T any](f *Field, idx int) (T, bool) {
	switch vec := f.vector.(type) {
	case *nullableGenericVector[T]:
		return vec.ConcreteAtTyped(idx)
	case *genericVector[T]:
		return vec.AtTyped(idx), true
	default:
		panic("Field is not backed by genericVector[T] or nullableGenericVector[T]")
	}
}

// AppendTypedNullable appends a pointer value to a nullable Field with zero allocation.
// Panics if the Field is not backed by nullableGenericVector[T].
func AppendTypedNullable[T any](f *Field, val *T) {
	if gv, ok := f.vector.(*nullableGenericVector[T]); ok {
		gv.AppendTyped(val)
		return
	}
	panic("Field is not backed by nullableGenericVector[T]")
}

// TypedField is a typed view over a *Field backed by genericVector[T].
// It binds the type parameter once so subsequent calls avoid the per-call
// type assertion that AtTyped/SetTyped/AppendTyped perform.
type TypedField[T any] struct {
	f   *Field
	vec *genericVector[T]
}

// FieldAs returns a TypedField view of f if f is backed by genericVector[T].
// The second return value is false if the underlying vector has a different
// element type or is nullable.
func FieldAs[T any](f *Field) (*TypedField[T], bool) {
	gv, ok := f.vector.(*genericVector[T])
	if !ok {
		return nil, false
	}
	return &TypedField[T]{f: f, vec: gv}, true
}

func (t *TypedField[T]) Field() *Field          { return t.f }
func (t *TypedField[T]) Len() int               { return t.vec.Len() }
func (t *TypedField[T]) At(i int) T             { return t.vec.AtTyped(i) }
func (t *TypedField[T]) Set(i int, v T)         { t.vec.SetTyped(i, v) }
func (t *TypedField[T]) Append(v T)             { t.vec.AppendTyped(v) }
func (t *TypedField[T]) AppendMany(vs []T)      { t.vec.AppendManyTyped(vs) }
func (t *TypedField[T]) Slice() []T             { return t.vec.Slice() }

// NullableTypedField is a typed view over a *Field backed by nullableGenericVector[T].
type NullableTypedField[T any] struct {
	f   *Field
	vec *nullableGenericVector[T]
}

// NullableFieldAs returns a NullableTypedField view of f if f is backed by
// nullableGenericVector[T].
func NullableFieldAs[T any](f *Field) (*NullableTypedField[T], bool) {
	gv, ok := f.vector.(*nullableGenericVector[T])
	if !ok {
		return nil, false
	}
	return &NullableTypedField[T]{f: f, vec: gv}, true
}

func (t *NullableTypedField[T]) Field() *Field             { return t.f }
func (t *NullableTypedField[T]) Len() int                  { return t.vec.Len() }
func (t *NullableTypedField[T]) At(i int) *T               { return t.vec.AtTyped(i) }
func (t *NullableTypedField[T]) Set(i int, v *T)           { t.vec.SetTyped(i, v) }
func (t *NullableTypedField[T]) Append(v *T)               { t.vec.AppendTyped(v) }
func (t *NullableTypedField[T]) AppendMany(vs []*T)        { t.vec.AppendManyTyped(vs) }
func (t *NullableTypedField[T]) ConcreteAt(i int) (T, bool) { return t.vec.ConcreteAtTyped(i) }
func (t *NullableTypedField[T]) SetConcrete(i int, v T)    { t.vec.SetConcreteTyped(i, v) }
func (t *NullableTypedField[T]) Slice() []*T               { return t.vec.Slice() }

