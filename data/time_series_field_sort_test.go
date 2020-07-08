package data

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestSortWideFrameFields(t *testing.T) {
	aTime := time.Date(2020, 1, 1, 12, 30, 0, 0, time.UTC)
	tests := []struct {
		name        string
		frameToSort *Frame
		afterSort   *Frame
	}{
		{
			name: "wide frame with names only",
			frameToSort: NewFrame("",
				NewField("time", nil, []time.Time{aTime}),
				NewField("bValue", nil, []float64{5}),
				NewField("aValue", nil, []float64{1}),
			),
			afterSort: NewFrame("",
				NewField("time", nil, []time.Time{aTime}),
				NewField("aValue", nil, []float64{1}),
				NewField("bValue", nil, []float64{5}),
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := sortWideFrameFields(tt.frameToSort)
			require.NoError(t, err)
			if diff := cmp.Diff(tt.frameToSort, tt.afterSort, FrameTestCompareOptions()...); diff != "" {
				t.Errorf("Result mismatch (-want +got):\n%s", diff)
			}
		})
	}

}
