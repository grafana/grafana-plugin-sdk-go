package data_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyAtDoesNotMutatePointerVector(t *testing.T) {
	frameA := data.NewFrame("test", data.NewField("test", nil, []*float64{float64Ptr(1.0)}))
	rowLength, err := frameA.RowLen()
	require.NoError(t, err)
	frameB := data.NewFrame("test", data.NewField("test", nil, []*float64{nil}))
	for i := 0; i < rowLength; i++ {
		frameB.Set(0, i, frameA.Fields[0].CopyAt(i))
	}
	frameB.Set(0, 0, float64Ptr(2.0))
	require.Equal(t, frameA.At(0, 0), float64Ptr(1.0))
}

func TestCopyAtDoesNotMutateVector(t *testing.T) {
	frameA := data.NewFrame("test", data.NewField("test", nil, []float64{1.0}))
	rowLength, err := frameA.RowLen()
	require.NoError(t, err)
	frameB := data.NewFrame("test", data.NewField("test", nil, []float64{0.0}))
	for i := 0; i < rowLength; i++ {
		frameB.Set(0, i, frameA.Fields[0].CopyAt(i))
	}
	frameB.Set(0, 0, 2.0)
	require.Equal(t, frameA.At(0, 0), (1.0))
}

// The slice data in the Field is a not exported, so methods on the Field are used to to manipulate its data.
type simpleFieldInfo struct {
	Name      string
	FieldType data.FieldType
}

func TestFieldTypeConversion(t *testing.T) {
	f := data.FieldTypeBool
	s := f.ItemTypeString()
	assert.Equal(t, "bool", s)
	c, ok := data.FieldTypeFromItemTypeString(s)
	require.True(t, ok, "must parse ok")
	assert.Equal(t, f, c)

	_, ok = data.FieldTypeFromItemTypeString("????")
	require.False(t, ok, "unknown type")

	c, ok = data.FieldTypeFromItemTypeString("float")
	require.Equal(t, true, ok, "must parse ok")
	assert.Equal(t, data.FieldTypeFloat64, c)

	obj := &simpleFieldInfo{
		Name:      "hello",
		FieldType: data.FieldTypeFloat64,
	}
	body, err := json.Marshal(obj)
	require.NoError(t, err)
	fmt.Printf("JSON: %s\n", string(body))

	copy := &simpleFieldInfo{}
	err = json.Unmarshal(body, &copy)
	require.NoError(t, err)

	assert.Equal(t, obj.FieldType, copy.FieldType)
}
