package data

import (
	"errors"
	"reflect"
)

// Errors returned by the Marshal function
var (
	ErrorNotSlice = errors.New("the data provided is not a slice")
)

// MarshalField is a field that is requested and turned into a field into a data.Frame
type MarshalField struct {
	Name  string
	Alias string
}

// Marshal turns `v` into a list of data.Frames.
// The list of fields can contain periods to refer to sub-structs and map keys.
// If `v` is not a slice, then ErrorNotSlice is returned.
func Marshal(name string, fields []MarshalField, v interface{}) (*Frame, error) {
	t := reflect.TypeOf(v)

	if t.Kind() != reflect.Slice {
		return nil, ErrorNotSlice
	}
	return nil, nil
}
