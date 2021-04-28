package sqlutil

import (
	"errors"
	"reflect"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

var (
	ErrorUnrecognizedType = errors.New("unrecognized type")
)

// FrameConverter defines how to convert the scanned value into a value that can be put into a dataframe (OutputFieldType)
type FrameConverter struct {
	FieldType     data.FieldType
	ConverterFunc func(in interface{}) (interface{}, error)
}

// Converter is used to convert known types returned in sql.Row to a type usable in a dataframe.
type Converter struct {
	// Name is the name of the converter that is used to distinguish them when debugging or parsing log output
	Name string

	// InputScanType is the type that is used when (*sql.Rows).Scan(...) is called.
	// Some drivers require certain data types to be used when scanning data from sql rows, and this type should reflect that.
	InputScanType reflect.Type

	// InputTypeName is the case-sensitive name that must match the type that this converter matches
	InputTypeName string

	// FrameConverter defines how to convert the scanned value into a value that can be put into a dataframe
	FrameConverter FrameConverter
}

// The DefaultConverterFunc assumes that the scanned value, in, is already a type that can be put into a dataframe.
func DefaultConverterFunc(in interface{}) (interface{}, error) {
	return in, nil
}

func NewDefaultFrameConverter(t reflect.Type) (FrameConverter, error) {
	slice := reflect.MakeSlice(reflect.SliceOf(t), 0, 0).Interface()
	if !data.ValidFieldType(slice) {
		return FrameConverter{}, ErrorUnrecognizedType
	}

	v := reflect.New(t)

	var fieldType data.FieldType
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		fieldType = data.FieldTypeFor(v.Interface()).NullableType()
	} else {
		fieldType = data.FieldTypeFor(v.Interface())
	}

	return FrameConverter{
		FieldType:     fieldType,
		ConverterFunc: DefaultConverterFunc,
	}, nil
}
