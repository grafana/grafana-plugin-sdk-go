package converters

import (
	"fmt"
	"strconv"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// Int64NOOP Assume the value is already an int64.
var Int64NOOP = data.FieldConverter{
	OutputFieldType: data.FieldTypeInt64,
}

// BoolNOOP Assume the value is already a bool.
var BoolNOOP = data.FieldConverter{
	OutputFieldType: data.FieldTypeBool,
}

// Float64NOOP Assume the value is already an int64.
var Float64NOOP = data.FieldConverter{
	OutputFieldType: data.FieldTypeFloat64,
}

// StringNOOP Assume the value is already a string
var StringNOOP = data.FieldConverter{
	OutputFieldType: data.FieldTypeString,
}

// AnyToNullableString any value as a string
var AnyToNullableString = data.FieldConverter{
	OutputFieldType: data.FieldTypeNullableString,
	Converter: func(v interface{}) (interface{}, error) {
		if v == nil {
			return nil, nil
		}
		str := fmt.Sprintf("%+v", v) // the +v adds field names
		return &str, nil
	},
}

// AnyToString any value as a string
var AnyToString = data.FieldConverter{
	OutputFieldType: data.FieldTypeString,
	Converter: func(v interface{}) (interface{}, error) {
		if v == nil {
			return nil, nil
		}
		return fmt.Sprintf("%+v", v), nil // the +v adds field names
	},
}

// Float64ToNullableFloat64 optional float value
var Float64ToNullableFloat64 = data.FieldConverter{
	OutputFieldType: data.FieldTypeNullableFloat64,
	Converter: func(v interface{}) (interface{}, error) {
		if v == nil {
			return nil, nil
		}
		val, ok := v.(float64)
		if !ok {
			return nil, fmt.Errorf("expected float64 input but got type %T", v)
		}
		return &val, nil
	},
}

// Int64ToNullableInt64 optional int value
var Int64ToNullableInt64 = data.FieldConverter{
	OutputFieldType: data.FieldTypeNullableInt64,
	Converter: func(v interface{}) (interface{}, error) {
		if v == nil {
			return nil, nil
		}
		val, ok := v.(int64)
		if !ok {
			return nil, fmt.Errorf("expected int64 input but got type %T", v)
		}
		return &val, nil
	},
}

// UInt64ToNullableUInt64 optional int value
var UInt64ToNullableUInt64 = data.FieldConverter{
	OutputFieldType: data.FieldTypeNullableUint64,
	Converter: func(v interface{}) (interface{}, error) {
		if v == nil {
			return nil, nil
		}
		val, ok := v.(uint64)
		if !ok {
			return nil, fmt.Errorf("expected uint64 input but got type %T", v)
		}
		return &val, nil
	},
}

// BoolToNullableBool optional bool value
var BoolToNullableBool = data.FieldConverter{
	OutputFieldType: data.FieldTypeNullableBool,
	Converter: func(v interface{}) (interface{}, error) {
		if v == nil {
			return nil, nil
		}
		val, ok := v.(bool)
		if !ok {
			return nil, fmt.Errorf("expected bool input but got type %T", v)
		}
		return &val, nil
	},
}

// TimeToNullableTime optional time value
var TimeToNullableTime = data.FieldConverter{
	OutputFieldType: data.FieldTypeNullableTime,
	Converter: func(v interface{}) (interface{}, error) {
		if v == nil {
			return nil, nil
		}
		val, ok := v.(time.Time)
		if !ok {
			return nil, fmt.Errorf("expected time input but got type %T", v)
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

// StringToNullableFloat64 string to float
var StringToNullableFloat64 = data.FieldConverter{
	OutputFieldType: data.FieldTypeNullableFloat64,
	Converter: func(v interface{}) (interface{}, error) {
		if v == nil {
			return nil, nil
		}
		val, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("expected string input but got type %T", v)
		}
		fV, err := strconv.ParseFloat(val, 64)
		return &fV, err
	},
}

// Float64EpochSecondsToTime  numeric seconds to time
var Float64EpochSecondsToTime = data.FieldConverter{
	OutputFieldType: data.FieldTypeTime,
	Converter: func(v interface{}) (interface{}, error) {
		fV, ok := v.(float64)
		if !ok {
			return nil, fmt.Errorf("expected float64 input but got type %T", v)
		}
		return time.Unix(int64(fV), 0).UTC(), nil
	},
}

// Float64EpochMillisToTime convert numeric milliseconds to time
var Float64EpochMillisToTime = data.FieldConverter{
	OutputFieldType: data.FieldTypeTime,
	Converter: func(v interface{}) (interface{}, error) {
		fV, ok := v.(float64)
		if !ok {
			return nil, fmt.Errorf("expected float64 input but got type %T", v)
		}
		return time.Unix(0, int64(fV)*int64(time.Millisecond)).UTC(), nil
	},
}

// Boolean ...
var Boolean = data.FieldConverter{
	OutputFieldType: data.FieldTypeBool,
	Converter: func(v interface{}) (interface{}, error) {
		fV, ok := v.(bool)
		if !ok {
			return nil, fmt.Errorf("expected bool input but got type %T", v)
		}
		return fV, nil
	},
}
