package data_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/require"
)

// The slice data in the Field is a not exported, so methods on the Field are used to to manipulate its data.
type simpleFieldInfo struct {
	Name      string
	FieldType data.FieldType
}

func TestFieldTypeConversion(t *testing.T) {
	f := data.FieldTypeBool
	s := f.ItemTypeString()
	require.Equal(t, "bool", s)
	c, ok := data.FieldTypeFromItemTypeString(s)
	require.True(t, ok, "must parse ok")
	require.Equal(t, f, c)

	_, ok = data.FieldTypeFromItemTypeString("????")
	require.False(t, ok, "unknown type")

	c, ok = data.FieldTypeFromItemTypeString("float")
	require.True(t, ok, "must parse ok")
	require.Equal(t, data.FieldTypeFloat64, c)

	obj := &simpleFieldInfo{
		Name:      "hello",
		FieldType: data.FieldTypeFloat64,
	}
	body, err := json.Marshal(obj)
	require.NoError(t, err)

	objCopy := &simpleFieldInfo{}
	err = json.Unmarshal(body, &objCopy)
	require.NoError(t, err)

	require.Equal(t, obj.FieldType, objCopy.FieldType)
}

func TestFieldTypeFor(t *testing.T) {
	examples := []struct {
		input interface{}
		want  data.FieldType
	}{
		{input: int8(0), want: data.FieldTypeInt8},
		{input: int16(0), want: data.FieldTypeInt16},
		{input: int32(0), want: data.FieldTypeInt32},
		{input: int64(0), want: data.FieldTypeInt64},

		{input: uint8(0), want: data.FieldTypeUint8},
		{input: uint16(0), want: data.FieldTypeUint16},
		{input: uint32(0), want: data.FieldTypeUint32},
		{input: uint64(0), want: data.FieldTypeUint64},

		{input: float32(0.0), want: data.FieldTypeFloat32},
		{input: float64(0.0), want: data.FieldTypeFloat64},

		{input: false, want: data.FieldTypeBool},
		{input: "", want: data.FieldTypeString},
		{input: time.Now(), want: data.FieldTypeTime},
		{input: time.Second, want: data.FieldTypeDuration},

		{input: struct{}{}, want: data.FieldTypeUnknown},
	}

	for i, example := range examples {
		got := data.FieldTypeFor(example.input)
		require.Equal(t, example.want, got, "example %d failed: got %v, expected %v", i, got, example.want)
	}
}

func TestNullableType(t *testing.T) {
	examples := []struct {
		input data.FieldType
		want  data.FieldType
	}{
		{input: data.FieldTypeInt8, want: data.FieldTypeNullableInt8},
		{input: data.FieldTypeNullableInt8, want: data.FieldTypeNullableInt8},
		{input: data.FieldTypeInt16, want: data.FieldTypeNullableInt16},
		{input: data.FieldTypeNullableInt16, want: data.FieldTypeNullableInt16},
		{input: data.FieldTypeInt32, want: data.FieldTypeNullableInt32},
		{input: data.FieldTypeNullableInt32, want: data.FieldTypeNullableInt32},
		{input: data.FieldTypeInt64, want: data.FieldTypeNullableInt64},
		{input: data.FieldTypeNullableInt64, want: data.FieldTypeNullableInt64},

		{input: data.FieldTypeUint8, want: data.FieldTypeNullableUint8},
		{input: data.FieldTypeNullableUint8, want: data.FieldTypeNullableUint8},
		{input: data.FieldTypeUint16, want: data.FieldTypeNullableUint16},
		{input: data.FieldTypeNullableUint16, want: data.FieldTypeNullableUint16},
		{input: data.FieldTypeUint32, want: data.FieldTypeNullableUint32},
		{input: data.FieldTypeNullableUint32, want: data.FieldTypeNullableUint32},
		{input: data.FieldTypeUint64, want: data.FieldTypeNullableUint64},
		{input: data.FieldTypeNullableUint64, want: data.FieldTypeNullableUint64},

		{input: data.FieldTypeFloat32, want: data.FieldTypeNullableFloat32},
		{input: data.FieldTypeNullableFloat32, want: data.FieldTypeNullableFloat32},
		{input: data.FieldTypeFloat64, want: data.FieldTypeNullableFloat64},
		{input: data.FieldTypeNullableFloat64, want: data.FieldTypeNullableFloat64},

		{input: data.FieldTypeBool, want: data.FieldTypeNullableBool},
		{input: data.FieldTypeNullableBool, want: data.FieldTypeNullableBool},
		{input: data.FieldTypeString, want: data.FieldTypeNullableString},
		{input: data.FieldTypeNullableString, want: data.FieldTypeNullableString},
		{input: data.FieldTypeTime, want: data.FieldTypeNullableTime},
		{input: data.FieldTypeNullableTime, want: data.FieldTypeNullableTime},
		{input: data.FieldTypeDuration, want: data.FieldTypeNullableDuration},
		{input: data.FieldTypeNullableDuration, want: data.FieldTypeNullableDuration},
	}

	for i, example := range examples {
		got := example.input.NullableType()
		require.Equal(t, example.want, got, "example %d failed: got %v, expected %v", i, got, example.want)
	}
}

func TestNonNullableType(t *testing.T) {
	examples := []struct {
		input data.FieldType
		want  data.FieldType
	}{
		{input: data.FieldTypeInt8, want: data.FieldTypeInt8},
		{input: data.FieldTypeNullableInt8, want: data.FieldTypeInt8},
		{input: data.FieldTypeInt16, want: data.FieldTypeInt16},
		{input: data.FieldTypeNullableInt16, want: data.FieldTypeInt16},
		{input: data.FieldTypeInt32, want: data.FieldTypeInt32},
		{input: data.FieldTypeNullableInt32, want: data.FieldTypeInt32},
		{input: data.FieldTypeInt64, want: data.FieldTypeInt64},
		{input: data.FieldTypeNullableInt64, want: data.FieldTypeInt64},

		{input: data.FieldTypeUint8, want: data.FieldTypeUint8},
		{input: data.FieldTypeNullableUint8, want: data.FieldTypeUint8},
		{input: data.FieldTypeUint16, want: data.FieldTypeUint16},
		{input: data.FieldTypeNullableUint16, want: data.FieldTypeUint16},
		{input: data.FieldTypeUint32, want: data.FieldTypeUint32},
		{input: data.FieldTypeNullableUint32, want: data.FieldTypeUint32},
		{input: data.FieldTypeUint64, want: data.FieldTypeUint64},
		{input: data.FieldTypeNullableUint64, want: data.FieldTypeUint64},

		{input: data.FieldTypeFloat32, want: data.FieldTypeFloat32},
		{input: data.FieldTypeNullableFloat32, want: data.FieldTypeFloat32},
		{input: data.FieldTypeFloat64, want: data.FieldTypeFloat64},
		{input: data.FieldTypeNullableFloat64, want: data.FieldTypeFloat64},

		{input: data.FieldTypeBool, want: data.FieldTypeBool},
		{input: data.FieldTypeNullableBool, want: data.FieldTypeBool},

		{input: data.FieldTypeString, want: data.FieldTypeString},
		{input: data.FieldTypeNullableString, want: data.FieldTypeString},

		{input: data.FieldTypeTime, want: data.FieldTypeTime},
		{input: data.FieldTypeNullableTime, want: data.FieldTypeTime},
		{input: data.FieldTypeDuration, want: data.FieldTypeDuration},
		{input: data.FieldTypeNullableDuration, want: data.FieldTypeDuration},
	}

	for i, example := range examples {
		got := example.input.NonNullableType()
		require.Equal(t, example.want, got, "example %d failed: got %v, expected %v", i, got, example.want)
	}
}

func TestFieldTypeFromItemTypeString(t *testing.T) {
	examples := []struct {
		input string
		want  data.FieldType
	}{
		{input: "int8", want: data.FieldTypeInt8},
		{input: "*int8", want: data.FieldTypeNullableInt8},
		{input: "int16", want: data.FieldTypeInt16},
		{input: "*int16", want: data.FieldTypeNullableInt16},
		{input: "int32", want: data.FieldTypeInt32},
		{input: "*int32", want: data.FieldTypeNullableInt32},
		{input: "int64", want: data.FieldTypeInt64},
		{input: "*int64", want: data.FieldTypeNullableInt64},

		{input: "uint8", want: data.FieldTypeUint8},
		{input: "*uint8", want: data.FieldTypeNullableUint8},
		{input: "uint16", want: data.FieldTypeUint16},
		{input: "*uint16", want: data.FieldTypeNullableUint16},
		{input: "uint32", want: data.FieldTypeUint32},
		{input: "*uint32", want: data.FieldTypeNullableUint32},
		{input: "uint64", want: data.FieldTypeUint64},
		{input: "*uint64", want: data.FieldTypeNullableUint64},

		{input: "float32", want: data.FieldTypeFloat32},
		{input: "*float32", want: data.FieldTypeNullableFloat32},
		{input: "float64", want: data.FieldTypeFloat64},
		{input: "double", want: data.FieldTypeFloat64},
		{input: "float", want: data.FieldTypeFloat64},
		{input: "*float64", want: data.FieldTypeNullableFloat64},

		{input: "bool", want: data.FieldTypeBool},
		{input: "boolean", want: data.FieldTypeBool},
		{input: "*bool", want: data.FieldTypeNullableBool},

		{input: "time", want: data.FieldTypeTime},
		{input: "time.Time", want: data.FieldTypeTime},
		{input: "*time.Time", want: data.FieldTypeNullableTime},
		{input: "duration", want: data.FieldTypeDuration},
		{input: "time.Duration", want: data.FieldTypeDuration},
		{input: "*time.Duration", want: data.FieldTypeNullableDuration},
	}

	for i, example := range examples {
		got, converted := data.FieldTypeFromItemTypeString(example.input)
		require.Equal(t, example.want, got, "example %d failed: got %v, expected %v", i, got, example.want)
		require.True(t, converted, "example %d failed: got false, expected true", i)
	}

	// Everything else is considered a string
	want := data.FieldTypeNullableString
	got, converted := data.FieldTypeFromItemTypeString("hello")
	require.Equal(t, want, got)
	require.False(t, converted)
}

func TestFieldTypeItemTypeString(t *testing.T) {
	examples := []struct {
		input data.FieldType
		want  string
	}{
		{input: data.FieldTypeInt8, want: "int8"},
		{input: data.FieldTypeNullableInt8, want: "*int8"},
		{input: data.FieldTypeInt16, want: "int16"},
		{input: data.FieldTypeNullableInt16, want: "*int16"},
		{input: data.FieldTypeInt32, want: "int32"},
		{input: data.FieldTypeNullableInt32, want: "*int32"},
		{input: data.FieldTypeInt64, want: "int64"},
		{input: data.FieldTypeNullableInt64, want: "*int64"},
		{input: data.FieldTypeUint8, want: "uint8"},
		{input: data.FieldTypeNullableUint8, want: "*uint8"},

		{input: data.FieldTypeUint16, want: "uint16"},
		{input: data.FieldTypeNullableUint16, want: "*uint16"},
		{input: data.FieldTypeUint32, want: "uint32"},
		{input: data.FieldTypeNullableUint32, want: "*uint32"},
		{input: data.FieldTypeUint64, want: "uint64"},
		{input: data.FieldTypeNullableUint64, want: "*uint64"},

		{input: data.FieldTypeFloat32, want: "float32"},
		{input: data.FieldTypeNullableFloat32, want: "*float32"},
		{input: data.FieldTypeFloat64, want: "float64"},
		{input: data.FieldTypeNullableFloat64, want: "*float64"},

		{input: data.FieldTypeBool, want: "bool"},
		{input: data.FieldTypeNullableBool, want: "*bool"},
		{input: data.FieldTypeString, want: "string"},
		{input: data.FieldTypeNullableString, want: "*string"},

		{input: data.FieldTypeTime, want: "time.Time"},
		{input: data.FieldTypeNullableTime, want: "*time.Time"},
		{input: data.FieldTypeDuration, want: "time.Duration"},
		{input: data.FieldTypeNullableDuration, want: "*time.Duration"},
	}

	for i, example := range examples {
		got := example.input.ItemTypeString()
		require.Equal(t, example.want, got, "example %d failed: got %v, expected %v", i, got, example.want)
	}
}

func TestFieldTypeValidFieldType(t *testing.T) {
	validPrimitives := []interface{}{
		[]int8{},
		[]*int8{},
		[]int16{},
		[]*int16{},
		[]int32{},
		[]*int32{},
		[]int64{},
		[]*int64{},

		[]uint8{},
		[]*uint8{},
		[]uint16{},
		[]*uint16{},
		[]uint32{},
		[]*uint32{},
		[]uint64{},
		[]*uint64{},

		[]float32{},
		[]*float32{},
		[]float64{},
		[]*float64{},

		[]bool{},
		[]*bool{},

		[]string{},
		[]*string{},

		[]time.Time{},
		[]*time.Time{},
		[]time.Duration{},
		[]*time.Duration{},
	}

	for i, primitive := range validPrimitives {
		require.True(t, data.ValidFieldType(primitive), "example %d failed: got false, expected true", i)
	}
}
