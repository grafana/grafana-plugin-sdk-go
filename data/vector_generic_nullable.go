package data

// nullableGenericVector is a nullable vector implementation using Go generics.
// It stores pointers to T, allowing nil values.
type nullableGenericVector[T any] struct {
	data []*T
}

// newNullableGenericVector creates a new nullable generic vector with the specified size.
func newNullableGenericVector[T any](n int) *nullableGenericVector[T] {
	return &nullableGenericVector[T]{
		data: make([]*T, n),
	}
}

// newNullableGenericVectorWithCapacity creates a new nullable generic vector with length 0 but pre-allocated capacity.
// This is useful for avoiding reallocations when the final size is known in advance.
func newNullableGenericVectorWithCapacity[T any](capacity int) *nullableGenericVector[T] {
	return &nullableGenericVector[T]{
		data: make([]*T, 0, capacity),
	}
}

// newNullableGenericVectorWithValues creates a new nullable generic vector from an existing slice.
func newNullableGenericVectorWithValues[T any](values []*T) *nullableGenericVector[T] {
	data := make([]*T, len(values))
	copy(data, values)
	return &nullableGenericVector[T]{data: data}
}

// AtTyped returns the pointer at index i without boxing.
// Returns nil if the value is null.
func (v *nullableGenericVector[T]) AtTyped(i int) *T {
	if i < 0 || i >= v.Len() {
		panic("Invalid index; vector length should be greater or equal to that index")
	}
	return v.data[i]
}

// SetTyped sets the pointer at index i without boxing.
func (v *nullableGenericVector[T]) SetTyped(i int, val *T) {
	if i < 0 || i >= v.Len() {
		panic("Invalid index; vector length should be greater or equal to that index")
	}
	v.data[i] = val
}

// AppendTyped adds a pointer to the end without boxing.
func (v *nullableGenericVector[T]) AppendTyped(val *T) {
	v.data = append(v.data, val)
}

// AppendManyTyped appends multiple pointer values at once from a slice.
// This is more efficient than calling AppendTyped repeatedly.
func (v *nullableGenericVector[T]) AppendManyTyped(vals []*T) {
	v.data = append(v.data, vals...)
}

// AppendManyWithNulls appends values from a slice, creating pointers for non-null values.
// The isNull function should return true if the value at index i is null.
// This is optimized for batch operations from Arrow arrays.
func (v *nullableGenericVector[T]) AppendManyWithNulls(vals []T, isNull func(int) bool) {
	startIdx := len(v.data)
	// Pre-allocate space
	v.data = append(v.data, make([]*T, len(vals))...)

	// Fill in the values
	for i, val := range vals {
		if !isNull(i) {
			// Create a new variable to get a stable pointer
			valCopy := val
			v.data[startIdx+i] = &valCopy
		}
		// else: already nil from make()
	}
}

// ConcreteAtTyped returns the dereferenced value if not nil.
// The second return value indicates if the value was non-nil.
func (v *nullableGenericVector[T]) ConcreteAtTyped(i int) (T, bool) {
	if i < 0 || i >= v.Len() {
		panic("Invalid index; vector length should be greater or equal to that index")
	}
	if v.data[i] == nil {
		var zero T
		return zero, false
	}
	return *v.data[i], true
}

// SetConcreteTyped sets the value by creating a pointer to val.
func (v *nullableGenericVector[T]) SetConcreteTyped(i int, val T) {
	if i < 0 || i >= v.Len() {
		panic("Invalid index; vector length should be greater or equal to that index")
	}
	v.data[i] = &val
}

// Len returns the length of the vector.
func (v *nullableGenericVector[T]) Len() int {
	return len(v.data)
}

// Extend extends the vector by n elements with nil values.
func (v *nullableGenericVector[T]) Extend(n int) {
	v.data = append(v.data, make([]*T, n)...)
}

// InsertTyped inserts a value at index i.
func (v *nullableGenericVector[T]) InsertTyped(i int, val *T) {
	switch {
	case i < v.Len():
		v.Extend(1)
		copy(v.data[i+1:], v.data[i:])
		v.SetTyped(i, val)
	case i == v.Len():
		v.AppendTyped(val)
	default:
		panic("Invalid index; vector length should be greater or equal to that index")
	}
}

// DeleteTyped removes the element at index i.
func (v *nullableGenericVector[T]) DeleteTyped(i int) {
	if i < 0 || i >= v.Len() {
		panic("Invalid index; vector length should be greater or equal to that index")
	}
	v.data = append(v.data[:i], v.data[i+1:]...)
}

// CopyAtTyped returns a copy of the pointer value at index i.
// If the value is nil, returns nil. Otherwise returns a new pointer to a copy.
func (v *nullableGenericVector[T]) CopyAtTyped(i int) *T {
	if i < 0 || i >= v.Len() {
		panic("Invalid index; vector length should be greater or equal to that index")
	}
	if v.data[i] == nil {
		return nil
	}
	val := *v.data[i]
	return &val
}

// Slice returns the underlying slice (read-only access recommended).
func (v *nullableGenericVector[T]) Slice() []*T {
	return v.data
}

// --- Backward compatibility interface{} methods ---

// Set sets the value at index idx.
func (v *nullableGenericVector[T]) Set(idx int, i interface{}) {
	if i == nil {
		v.data[idx] = nil
		return
	}
	v.data[idx] = i.(*T)
}

// SetConcrete sets the value by converting from concrete type.
func (v *nullableGenericVector[T]) SetConcrete(idx int, i interface{}) {
	val := i.(T)
	v.data[idx] = &val
}

// Append adds a value to the end.
func (v *nullableGenericVector[T]) Append(i interface{}) {
	if i == nil {
		v.data = append(v.data, nil)
		return
	}
	v.data = append(v.data, i.(*T))
}

// NilAt returns true if the value at index i is nil.
func (v *nullableGenericVector[T]) NilAt(i int) bool {
	return v.data[i] == nil
}

// At returns the pointer at index i as interface{}.
func (v *nullableGenericVector[T]) At(i int) interface{} {
	return v.data[i]
}

// CopyAt returns a copy of the value as interface{}.
func (v *nullableGenericVector[T]) CopyAt(i int) interface{} {
	if v.data[i] == nil {
		var g *T
		return g
	}
	val := *v.data[i]
	return &val
}

// ConcreteAt returns the dereferenced value as interface{}.
func (v *nullableGenericVector[T]) ConcreteAt(i int) (interface{}, bool) {
	var zero T
	val := v.data[i]
	if val == nil {
		return zero, false
	}
	return *val, true
}

// PointerAt returns a pointer to the pointer at index i.
func (v *nullableGenericVector[T]) PointerAt(i int) interface{} {
	return &v.data[i]
}

// Type returns the FieldType for this vector.
func (v *nullableGenericVector[T]) Type() FieldType {
	return vectorFieldType(v)
}

// Insert inserts a value at index i.
func (v *nullableGenericVector[T]) Insert(i int, val interface{}) {
	if val == nil {
		v.InsertTyped(i, nil)
		return
	}
	v.InsertTyped(i, val.(*T))
}

// Delete removes the element at index i.
func (v *nullableGenericVector[T]) Delete(i int) {
	v.DeleteTyped(i)
}
