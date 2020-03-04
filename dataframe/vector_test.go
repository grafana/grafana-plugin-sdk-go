package dataframe_test

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/dataframe"
	"github.com/stretchr/testify/require"
)

func TestCopyAtDoesNotMutatePointerVector(t *testing.T) {
	frameA := dataframe.New("test", dataframe.NewField("test", nil, []*float64{float64Ptr(1.0)}))
	rowLength, err := frameA.RowLen()
	require.NoError(t, err)
	frameB := dataframe.New("test", dataframe.NewField("test", nil, []*float64{nil}))
	for i := 0; i < rowLength; i++ {
		frameB.Set(0, i, frameA.Fields[0].Vector.CopyAt(i))
	}
	frameB.Set(0, 0, float64Ptr(2.0))
	require.Equal(t, frameA.At(0, 0), float64Ptr(1.0))
}

func TestCopyAtDoesNotMutateVector(t *testing.T) {
	frameA := dataframe.New("test", dataframe.NewField("test", nil, []float64{1.0}))
	rowLength, err := frameA.RowLen()
	require.NoError(t, err)
	frameB := dataframe.New("test", dataframe.NewField("test", nil, []float64{0.0}))
	for i := 0; i < rowLength; i++ {
		frameB.Set(0, i, frameA.Fields[0].Vector.CopyAt(i))
	}
	frameB.Set(0, 0, 2.0)
	require.Equal(t, frameA.At(0, 0), (1.0))
}
