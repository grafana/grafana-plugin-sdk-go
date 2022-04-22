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
	err = json.Unmarshal(body, objCopy)
	require.NoError(t, err)

	require.Equal(t, obj.FieldType, objCopy.FieldType)
}
