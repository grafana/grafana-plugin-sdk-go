package converters

import (
	"fmt"
	"strconv"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func toConversionError(expected string, v interface{}) error {
	return fmt.Errorf("expected %s input but got type %T for value \"%v\"", expected, v, v)
}

// Int64NOOP The input and output values are both int64.
var Int64NOOP = data.FieldConverter{
	OutputFieldType: data.FieldTypeInt64,
}

// BoolNOOP The input and output values are both bool.
var BoolNOOP = data.FieldConverter{
	OutputFieldType: data.FieldTypeBool,
}

// Float64NOOP The input and output values are both int64.
var Float64NOOP = data.FieldConverter{
	OutputFieldType: data.FieldTypeFloat64,
}

// StringNOOP The input and output values are both string.
var StringNOOP = data.FieldConverter{
	OutputFieldType: data.FieldTypeString,
}

// AnyToNullableString Converts any non-null value into a string.
var AnyToNullableString = data.FieldConverter{
	OutputFieldType: data.FieldTypeNullableString,
	Converter: func(v interface{}) (interface{}, error) {
		if v == nil {
			return nil, nil
		}
		str := fmt.Sprintf("%v", v)
		return &str, nil
	},
}

// AnyToString Converts any value into a string.
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
		if v == nil {
			return nil, nil
		}
		val, ok := v.(float64)
		if !ok {
			return nil, toConversionError("float64", v)
		}
		return &val, nil
	},
}

// Int64ToNullableInt64 returns an error if the input is not an int64.
var Int64ToNullableInt64 = data.FieldConverter{
	OutputFieldType: data.FieldTypeNullableInt64,
	Converter: func(v interface{}) (interface{}, error) {
		if v == nil {
			return nil, nil
		}
		val, ok := v.(int64)
		if !ok {
			return nil, toConversionError("int64", v)
		}
		return &val, nil
	},
}

// Uint64ToNullableUInt64 returns an error if the input is not a uint64.
var Uint64ToNullableUInt64 = data.FieldConverter{
	OutputFieldType: data.FieldTypeNullableUint64,
	Converter: func(v interface{}) (interface{}, error) {
		if v == nil {
			return nil, nil
		}
		val, ok := v.(uint64)
		if !ok {
			return nil, toConversionError("uint64", v)
		}
		return &val, nil
	},
}

// BoolToNullableBool returns an error if the input is not a bool.
var BoolToNullableBool = data.FieldConverter{
	OutputFieldType: data.FieldTypeNullableBool,
	Converter: func(v interface{}) (interface{}, error) {
		if v == nil {
			return nil, nil
		}
		val, ok := v.(bool)
		if !ok {
			return nil, toConversionError("bool", v)
		}
		return &val, nil
	},
}

// TimeToNullableTime returns an error if the input is not a time.Time.
var TimeToNullableTime = data.FieldConverter{
	OutputFieldType: data.FieldTypeNullableTime,
	Converter: func(v interface{}) (interface{}, error) {
		if v == nil {
			return nil, nil
		}
		val, ok := v.(time.Time)
		if !ok {
			return nil, toConversionError("time.Time", v)
		}
		return &val, nil
	},
}

// RFC3339StringToNullableTime convert a string with RFC3339 to a time object.
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
		if v == nil {
			return nil, nil
		}
		val, ok := v.(string)
		if !ok {
			return nil, toConversionError("string", v)
		}
		fV, err := strconv.ParseFloat(val, 64)
		return &fV, err
	},
}

// Float64EpochSecondsToTime converts a numberic seconds to time.Time.
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
