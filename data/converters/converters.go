// Package converters provides data.FieldConverters commonly used by plugins.
package converters

import (
	"fmt"
	"strconv"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func toConversionError(expected string, v interface{}) error {
	return fmt.Errorf(`expected %s input but got type %T for value "%v"`, expected, v, v)
}

// Int64NOOP is a data.FieldConverter that performs no conversion.
// It should be used when the input type is an int64 and the Field's type
// is also an int64. The conversion will panic if the the input type does
// not match the Field type.
var Int64NOOP = data.FieldConverter{
	OutputFieldType: data.FieldTypeInt64,
}

// BoolNOOP is a data.FieldConverter that performs no conversion.
// It should be used when the input type is an bool and the Field's type
// is also an bool. The conversion will panic if the the input type does
// not match the Field type.
var BoolNOOP = data.FieldConverter{
	OutputFieldType: data.FieldTypeBool,
}

// Float64NOOP is a data.FieldConverter that performs no conversion.
// It should be used when the input type is an float64 and the Field's type
// is also an float64. The conversion will panic if the the input type does
// not match the Field type.
var Float64NOOP = data.FieldConverter{
	OutputFieldType: data.FieldTypeFloat64,
}

// StringNOOP is a data.FieldConverter that performs no conversion.
// It should be used when the input type is an int64 and the Field's type
// is also an string. The conversion will panic if the the input type does
// not match the Field type.
var StringNOOP = data.FieldConverter{
	OutputFieldType: data.FieldTypeString,
}

// AnyToNullableString converts any non-nil value into a *string.
// If the the input value is nil the output value is *string typed nil.
var AnyToNullableString = data.FieldConverter{
	OutputFieldType: data.FieldTypeNullableString,
	Converter: func(v interface{}) (interface{}, error) {
		var str *string
		if v != nil {
			s := fmt.Sprintf("%v", v)
			str = &s
		}
		return str, nil
	},
}

// AnyToString converts any value into a string.
var AnyToString = data.FieldConverter{
	OutputFieldType: data.FieldTypeString,
	Converter: func(v interface{}) (interface{}, error) {
		return fmt.Sprintf("%v", v), nil
	},
}

// Float64ToNullableFloat64 returns an error if the input is not a float64.
var Float64ToNullableFloat64 = data.FieldConverter{
	OutputFieldType: data.FieldTypeNullableFloat64,
	Converter: func(v interface{}) (interface{}, error) {
		var ptr *float64
		if v == nil {
			return ptr, nil
		}
		val, ok := v.(float64)
		if !ok {
			return ptr, toConversionError("float64", v)
		}
		ptr = &val
		return ptr, nil
	},
}

// Int64ToNullableInt64 returns an error if the input is not an int64.
var Int64ToNullableInt64 = data.FieldConverter{
	OutputFieldType: data.FieldTypeNullableInt64,
	Converter: func(v interface{}) (interface{}, error) {
		var ptr *int64
		if v == nil {
			return ptr, nil
		}
		val, ok := v.(int64)
		if !ok {
			return ptr, toConversionError("int64", v)
		}
		ptr = &val
		return ptr, nil
	},
}

// Uint64ToNullableUInt64 returns an error if the input is not a uint64.
var Uint64ToNullableUInt64 = data.FieldConverter{
	OutputFieldType: data.FieldTypeNullableUint64,
	Converter: func(v interface{}) (interface{}, error) {
		var ptr *uint64
		if v == nil {
			return ptr, nil
		}
		val, ok := v.(uint64)
		if !ok {
			return ptr, toConversionError("uint64", v)
		}
		ptr = &val
		return ptr, nil
	},
}

// BoolToNullableBool returns an error if the input is not a bool.
var BoolToNullableBool = data.FieldConverter{
	OutputFieldType: data.FieldTypeNullableBool,
	Converter: func(v interface{}) (interface{}, error) {
		var ptr *bool
		if v == nil {
			return ptr, nil
		}
		val, ok := v.(bool)
		if !ok {
			return ptr, toConversionError("bool", v)
		}
		ptr = &val
		return ptr, nil
	},
}

// RFC3339StringToNullableTime convert a string with RFC3339 to a *time.Time object.
func RFC3339StringToNullableTime(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}

	rv, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, err
	}

	u := rv.UTC()
	return &u, nil
}

// StringToNullableFloat64 parses a float64 value from a string.
var StringToNullableFloat64 = data.FieldConverter{
	OutputFieldType: data.FieldTypeNullableFloat64,
	Converter: func(v interface{}) (interface{}, error) {
		var ptr *float64
		if v == nil {
			return ptr, nil
		}
		val, ok := v.(string)
		if !ok {
			return ptr, toConversionError("string", v)
		}
		fV, err := strconv.ParseFloat(val, 64)
		ptr = &fV
		return ptr, err
	},
}

// Float64EpochSecondsToTime converts a numeric seconds to time.Time.
var Float64EpochSecondsToTime = data.FieldConverter{
	OutputFieldType: data.FieldTypeTime,
	Converter: func(v interface{}) (interface{}, error) {
		fV, ok := v.(float64)
		if !ok {
			return nil, toConversionError("float64", v)
		}
		return time.Unix(int64(fV), 0).UTC(), nil
	},
}

// Float64EpochMillisToTime convert numeric milliseconds to time.Time
var Float64EpochMillisToTime = data.FieldConverter{
	OutputFieldType: data.FieldTypeTime,
	Converter: func(v interface{}) (interface{}, error) {
		fV, ok := v.(float64)
		if !ok {
			return nil, toConversionError("float64", v)
		}
		return time.Unix(0, int64(fV)*int64(time.Millisecond)).UTC(), nil
	},
}

// Boolean returns an error if the input is not a bool.
var Boolean = data.FieldConverter{
	OutputFieldType: data.FieldTypeBool,
	Converter: func(v interface{}) (interface{}, error) {
		fV, ok := v.(bool)
		if !ok {
			return nil, toConversionError("bool", v)
		}
		return fV, nil
	},
}
