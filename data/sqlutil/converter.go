package sqlutil

import (
	"database/sql"
	"fmt"
	"reflect"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// FrameConverter defines how to convert the scanned value into a value that can be put into a dataframe (OutputFieldType)
type FrameConverter struct {
	// FieldType is the type that is created for the dataframe field.
	// The returned value from `ConverterFunc` should match this type, otherwise the data package will panic.
	FieldType data.FieldType
	// ConverterFunc defines how to convert the scanned `InputScanType` to the supplied `FieldType`.
	// `in` is always supplied as a pointer, as it is scanned as a pointer, even if `InputScanType` is not a pointer.
	// For example, if `InputScanType` is `string`, then `in` is `*string`
	ConverterFunc func(in interface{}) (interface{}, error)
}

// StringConverter can be used to store types not supported by
// a Frame into a *string. When scanning, if a SQL's row's InputScanType's Kind
// and InputScanKind match that returned by the sql response, then the
// conversion func will be run on the row.
// Note, a Converter should be favored over a StringConverter as not all SQL rows can be scanned into a string.
// This type is only here for backwards compatibility.
type StringConverter struct {
	// Name is an optional property that can be used to identify a converter
	Name          string
	InputScanKind reflect.Kind // reflect.Type might better or worse option?
	InputTypeName string

	// Conversion func may be nil to do no additional operations on the string conversion.
	ConversionFunc func(in *string) (*string, error)

	// If the Replacer is not nil, the replacement will be performed.
	Replacer *StringFieldReplacer
}

// Note: StringConverter is perhaps better understood as []byte. However, currently
// the Vector type ([][]byte) is not supported. https://github.com/grafana/grafana-plugin-sdk-go/issues/57

// StringFieldReplacer is used to replace a *string Field in a Frame. The type
// returned by the ReplaceFunc must match the type of elements of VectorType.
// Both properties must be non-nil.
// Note, a Converter should be favored over a StringConverter as not all SQL rows can be scanned into a string.
// This type is only here for backwards compatibility.
type StringFieldReplacer struct {
	OutputFieldType data.FieldType
	ReplaceFunc     func(in *string) (interface{}, error)
}

// ToConverter turns this StringConverter into a Converter, using the ScanType of string
func (s StringConverter) ToConverter() Converter {
	return Converter{
		Name:           s.Name,
		InputScanType:  reflect.TypeOf(""),
		InputTypeName:  s.InputTypeName,
		FrameConverter: StringFrameConverter(s),
	}
}

// StringFrameConverter creates a FrameConverter from a StringConverter
func StringFrameConverter(s StringConverter) FrameConverter {
	f := data.FieldTypeNullableString

	if s.Replacer != nil {
		f = s.Replacer.OutputFieldType
	}

	return FrameConverter{
		FieldType: f,
		ConverterFunc: func(in interface{}) (interface{}, error) {
			v := in.(*string)
			if s.ConversionFunc != nil {
				converted, err := s.ConversionFunc(v)
				if err != nil {
					return nil, err
				}
				v = converted
			}

			if s.Replacer.ReplaceFunc != nil {
				return s.Replacer.ReplaceFunc(v)
			}

			return v, nil
		},
	}
}

// ToConverters creates a slice of Converters from a slice of StringConverters
func ToConverters(s ...StringConverter) []Converter {
	n := make([]Converter, len(s))
	for i, v := range s {
		n[i] = v.ToConverter()
	}

	return n
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

// DefaultConverterFunc assumes that the scanned value, in, is already a type that can be put into a dataframe.
func DefaultConverterFunc(in interface{}) (interface{}, error) {
	return in, nil
}

// NewDefaultConverter creates a Converter that assumes that the value is scannable into a String, and placed into the dataframe as a nullable string.
func NewDefaultConverter(name string, nullable bool, t reflect.Type) Converter {
	slice := reflect.MakeSlice(reflect.SliceOf(t), 0, 0).Interface()
	if !data.ValidFieldType(slice) {
		return Converter{
			Name:          fmt.Sprintf("[%s] String converter", t),
			InputScanType: reflect.TypeOf(""),
			InputTypeName: name,
			FrameConverter: FrameConverter{
				FieldType: data.FieldTypeNullableString,
				ConverterFunc: func(in interface{}) (interface{}, error) {
					v := in.(*string)
					return v, nil
				},
			},
		}
	}

	v := reflect.New(t)

	var fieldType data.FieldType
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		fieldType = data.FieldTypeFor(v.Interface()).NullableType()
	} else {
		fieldType = data.FieldTypeFor(v.Interface())
	}

	if nullable {
		if converter, ok := NullConverters[t.String()]; ok {
			return converter
		}
	}

	return Converter{
		Name:          fmt.Sprintf("Default converter for %s", name),
		InputScanType: t,
		InputTypeName: name,
		FrameConverter: FrameConverter{
			FieldType:     fieldType,
			ConverterFunc: DefaultConverterFunc,
		},
	}
}

var (
	// NullStringConverter creates a *string using the scan type of `sql.NullString`
	NullStringConverter = Converter{
		Name:          "nullable string converter",
		InputScanType: reflect.TypeOf(sql.NullString{}),
		InputTypeName: "STRING",
		FrameConverter: FrameConverter{
			FieldType: data.FieldTypeNullableString,
			ConverterFunc: func(n interface{}) (interface{}, error) {
				v := n.(*sql.NullFloat64)

				if !v.Valid {
					return (*float64)(nil), nil
				}

				f := v.Float64
				return &f, nil
			},
		},
	}

	// NullDecimalConverter creates a *float64 using the scan type of `sql.NullFloat64`
	NullDecimalConverter = Converter{
		Name:          "NULLABLE decimal converter",
		InputScanType: reflect.TypeOf(sql.NullFloat64{}),
		InputTypeName: "DOUBLE",
		FrameConverter: FrameConverter{
			FieldType: data.FieldTypeNullableFloat64,
			ConverterFunc: func(n interface{}) (interface{}, error) {
				v := n.(*sql.NullFloat64)

				if !v.Valid {
					return (*float64)(nil), nil
				}

				f := v.Float64
				return &f, nil
			},
		},
	}
)

// NullConverters is a map of data type names (from reflect.TypeOf(...).String()) to converters
// Converters supplied here are used as defaults for fields that do not have a supplied Converter
var NullConverters = map[string]Converter{
	"float64": NullDecimalConverter,
	"string":  NullStringConverter,
}
