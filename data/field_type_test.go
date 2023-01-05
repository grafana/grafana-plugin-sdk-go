package data_test

import (
	"encoding/json"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/require"
)

// The slice data in the Field is a not exported, so methods on the Field are used to to manipulate its data.
type simpleFieldInfo struct {
	Name      string
	FieldType data.FieldType
}

func TestFieldTypeConversion(t *testing.T) {
	type scenario struct {
		ftype data.FieldType
		value string
	}

	info := []scenario{
		{ftype: data.FieldTypeBool, value: "bool"},
		{ftype: data.FieldTypeEnum, value: "enum"},
		{ftype: data.FieldTypeNullableEnum, value: "*enum"},
		{ftype: data.FieldTypeJSON, value: "json.RawMessage"},
	}
	for idx, check := range info {
		s := check.ftype.ItemTypeString()
		require.Equal(t, check.value, s, "index: %d", idx)
		c, ok := data.FieldTypeFromItemTypeString(s)
		require.True(t, ok, "must parse ok")
		require.Equal(t, check.ftype, c)
	}

	_, ok := data.FieldTypeFromItemTypeString("????")
	require.False(t, ok, "unknown type")

	c, ok := data.FieldTypeFromItemTypeString("float")
	require.True(t, ok, "must parse ok")
	require.Equal(t, data.FieldTypeFloat64, c)

	obj := &simpleFieldInfo{
		Name:      "hello",
		FieldType: data.FieldTypeFloat64,
	}
	body, err := json.Marshal(obj)
	require.NoError(t, err)

	objCopy := &simpleFieldInfo{}
	err = json.Unmarshal(body, objCopy)
	require.NoError(t, err)

	require.Equal(t, obj.FieldType, objCopy.FieldType)
}
