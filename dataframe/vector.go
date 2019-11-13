package dataframe

import (
	"fmt"
	"time"
)

// Vector represents a collection of Elements.
type Vector interface {
	Set(idx int, i interface{})
	Append(i interface{})
	At(i int) interface{}
	Len() int
}

func newVector(t interface{}, n int) (v Vector) {
	switch t.(type) {
	case []string:
		v = newStringVector(n)
	case []*string:
		v = newNullableStringVector(n)
	case []float64:
		v = newFloatVector(n)
	case []*float64:
		v = newNullableFloatVector(n)
	case []bool:
		v = newBoolVector(n)
	case []*bool:
		v = newNullableBoolVector(n)
	case []time.Time:
		v = newTimeVector(n)
	case []*time.Time:
		v = newNullableTimeVector(n)
	default:
		panic(fmt.Sprintf("unsupported vector type of %T", t))
	}
	return
}
