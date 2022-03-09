package sdata_test

import (
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/sdata"
	"github.com/stretchr/testify/require"
)

func TestMultiFrameSeriesValidate_ValidCases(t *testing.T) {
	tests := []struct {
		name                string
		mfs                 func() *sdata.MultiFrameSeries
		ignoredFieldIndices []sdata.FrameFieldIndex
	}{
		{
			name: "frame with no fields is valid (empty response)",
			mfs: func() *sdata.MultiFrameSeries {
				s := sdata.NewMultiFrameSeries()
				return s
			},
		},
		{
			name: "there can be extraneous fields (but they have no specific platform-wide meaning)",
			mfs: func() *sdata.MultiFrameSeries {
				s := sdata.NewMultiFrameSeries()
				s.AddMetric("one", nil, []time.Time{{}, time.Now().Add(time.Second)}, []float64{0, 1})
				(*s)[0].Fields = append((*s)[0].Fields, data.NewField("a", nil, []float64{2, 3}))
				(*s)[0].Fields = append((*s)[0].Fields, data.NewField("a", nil, []string{"4", "cats"}))
				return s
			},
			ignoredFieldIndices: []sdata.FrameFieldIndex{{0, 2}, {0, 3}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ignoredFieldIndices, err := tt.mfs().Validate()
			require.Nil(t, err)
			require.Equal(t, tt.ignoredFieldIndices, ignoredFieldIndices)
		})
	}
}

func TestMultiFrameSeriesValidiate_WithFrames_InvalidCases(t *testing.T) {
	tests := []struct {
		name        string
		mfs         *sdata.MultiFrameSeries
		errContains string
	}{
		{
			name: "frame must have type indicator",
			mfs: &sdata.MultiFrameSeries{
				data.NewFrame(""),
			},
			errContains: "missing type indicator",
		},
		{
			name: "frame with only value field is not valid, missing time field",
			mfs: &sdata.MultiFrameSeries{
				addFields(emptyFrameWithTypeMD(data.FrameTypeTimeSeriesMany),
					data.NewField("", nil, []float64{})),
			},
			errContains: "must have at least one time field",
		},
		{
			name: "frame with only a time field and no value is not valid",
			mfs: &sdata.MultiFrameSeries{
				addFields(emptyFrameWithTypeMD(data.FrameTypeTimeSeriesMany),
					data.NewField("", nil, []time.Time{})),
			},
			errContains: "must have at least one value field",
		},
		{
			name: "fields must be of the same length",
			mfs: &sdata.MultiFrameSeries{
				addFields(emptyFrameWithTypeMD(data.FrameTypeTimeSeriesMany),
					data.NewField("", nil, []float64{1, 2}),
					data.NewField("", nil, []time.Time{time.UnixMilli(1)})),
			},
			errContains: "mismatched field lengths",
		},
		{
			name: "frame with unsorted time is not valid",
			mfs: &sdata.MultiFrameSeries{
				addFields(emptyFrameWithTypeMD(data.FrameTypeTimeSeriesMany),
					data.NewField("", nil, []float64{1, 2}),
					data.NewField("", nil, []time.Time{time.UnixMilli(2), time.UnixMilli(1)})),
			},
			errContains: "unsorted time",
		},
		{
			name: "duplicate metrics as identified by name + labes are invalid",
			mfs: &sdata.MultiFrameSeries{
				addFields(emptyFrameWithTypeMD(data.FrameTypeTimeSeriesMany),
					data.NewField("os.cpu", data.Labels{"host": "a", "iface": "eth0"}, []float64{1, 2}),
					data.NewField("", nil, []time.Time{time.UnixMilli(1), time.UnixMilli(2)})),
				addFields(emptyFrameWithTypeMD(data.FrameTypeTimeSeriesMany),
					data.NewField("os.cpu", data.Labels{"iface": "eth0", "host": "a"}, []float64{1, 2}),
					data.NewField("", nil, []time.Time{time.UnixMilli(1), time.UnixMilli(2)})),
			},
			errContains: "duplicate metrics found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ignoredFieldIndices, err := tt.mfs.Validate()
			require.True(t, strings.Contains(err.Error(), tt.errContains))
			require.Nil(t, ignoredFieldIndices)
		})
	}
}

func emptyFrameWithTypeMD(t data.FrameType) *data.Frame {
	return data.NewFrame("").SetMeta(&data.FrameMeta{Type: t})
}
