package data

var _ vector = (*genericVector[int8])(nil)

// genericVector is a type-safe vector implementation using Go generics.
// It eliminates interface{} boxing overhead for better performance.
type genericVector[T any] struct {
	data []T
}

// newGenericVector creates a new generic vector with the specified size.
func newGenericVector[T any](n int) *genericVector[T] {
	return &genericVector[T]{
		data: make([]T, n),
	}
}

// newGenericVectorWithCapacity creates a new generic vector with length 0 but pre-allocated capacity.
// This is useful for avoiding reallocations when the final size is known in advance.
func newGenericVectorWithCapacity[T any](capacity int) *genericVector[T] {
	return &genericVector[T]{
		data: make([]T, 0, capacity),
	}
}

// newGenericVectorWithValues creates a new generic vector from an existing slice.
// It copies the data to prevent external modifications.
func newGenericVectorWithValues[T any](values []T) *genericVector[T] {
	data := make([]T, len(values))
	copy(data, values)
	return &genericVector[T]{data: data}
}

// AtTyped returns the value at index i without boxing.
// This is the zero-allocation accessor method.
func (v *genericVector[T]) AtTyped(i int) T {
	if i < 0 || i >= v.Len() {
		panic("Invalid index; vector length should be greater or equal to that index")
	}
	return v.data[i]
}

// SetTyped sets the value at index i without boxing.
// This is the zero-allocation setter method.
func (v *genericVector[T]) SetTyped(i int, val T) {
	if i < 0 || i >= v.Len() {
		panic("Invalid index; vector length should be greater or equal to that index")
	}
	v.data[i] = val
}

// AppendTyped adds a value to the end without boxing.
func (v *genericVector[T]) AppendTyped(val T) {
	v.data = append(v.data, val)
}

// AppendManyTyped appends multiple values at once from a slice.
// This is more efficient than calling AppendTyped repeatedly as it
// reduces slice growth operations and bounds checking overhead.
func (v *genericVector[T]) AppendManyTyped(vals []T) {
	v.data = append(v.data, vals...)
}

// Len returns the length of the vector.
func (v *genericVector[T]) Len() int {
	return len(v.data)
}

// Extend extends the vector by n elements with zero values.
func (v *genericVector[T]) Extend(n int) {
	v.data = append(v.data, make([]T, n)...)
}

// InsertTyped inserts a value at index i.
func (v *genericVector[T]) InsertTyped(i int, val T) {
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
func (v *genericVector[T]) DeleteTyped(i int) {
	if i < 0 || i >= v.Len() {
		panic("Invalid index; vector length should be greater or equal to that index")
	}
	v.data = append(v.data[:i], v.data[i+1:]...)
}

// CopyAtTyped returns a copy of the value at index i.
// For value types, this is the same as AtTyped.
func (v *genericVector[T]) CopyAtTyped(i int) T {
	if i < 0 || i >= v.Len() {
		panic("Invalid index; vector length should be greater or equal to that index")
	}
	return v.data[i]
}

// Slice returns the underlying slice (read-only access recommended).
func (v *genericVector[T]) Slice() []T {
	return v.data
}

// Set sets the value at index idx to i (requires type assertion).
func (v *genericVector[T]) Set(idx int, i interface{}) {
	v.data[idx] = i.(T)
}

// SetConcrete sets the value at index idx (same as Set for non-nullable).
func (v *genericVector[T]) SetConcrete(idx int, i interface{}) {
	v.Set(idx, i)
}

// Append adds a value to the end (requires type assertion).
func (v *genericVector[T]) Append(i interface{}) {
	v.data = append(v.data, i.(T))
}

// At returns the value at index i as interface{}.
// Note: This boxes the value and causes allocation for value types.
func (v *genericVector[T]) At(i int) interface{} {
	return v.data[i]
}

// NilAt returns false for non-nullable vectors.
func (v *genericVector[T]) NilAt(_ int) bool {
	return false
}

// PointerAt returns a pointer to the element at index i.
func (v *genericVector[T]) PointerAt(i int) interface{} {
	return &v.data[i]
}

// CopyAt returns a copy of the value as interface{}.
func (v *genericVector[T]) CopyAt(i int) interface{} {
	val := v.data[i]
	return val
}

// ConcreteAt returns the value at index i as interface{}.
func (v *genericVector[T]) ConcreteAt(i int) (interface{}, bool) {
	return v.data[i], true
}

// Type returns the FieldType for this vector.
func (v *genericVector[T]) Type() FieldType {
	return vectorFieldType(v)
}

// Insert inserts a value at index i (requires type assertion).
func (v *genericVector[T]) Insert(i int, val interface{}) {
	v.InsertTyped(i, val.(T))
}

// Delete removes the element at index i.
func (v *genericVector[T]) Delete(i int) {
	v.DeleteTyped(i)
}
