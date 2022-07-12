package timeseries_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/sdata"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/sdata/timeseries"
	"github.com/stretchr/testify/require"
)

func TestMultiFrameSeriesValidate_ValidCases(t *testing.T) {
	tests := []struct {
		name                string
		mfs                 func() *timeseries.MultiFrame
		ignoredFieldIndices []sdata.FrameFieldIndex
	}{
		{
			name: "frame with no fields is valid (empty response)",
			mfs: func() *timeseries.MultiFrame {
				s := timeseries.NewMultiFrame()
				return s
			},
		},
		{
			name: "basic example",
			mfs: func() *timeseries.MultiFrame {
				s := timeseries.NewMultiFrame()

				err := s.AddSeries("one", nil, []time.Time{{}, time.Now().Add(time.Second)}, []float64{0, 1})
				require.NoError(t, err)

				err = s.AddSeries("two", nil, []time.Time{{}, time.Now().Add(time.Second * 2)}, []float64{0, 1})
				require.NoError(t, err)
				return s
			},
		},
		{
			name: "there can be extraneous fields (but they have no specific platform-wide meaning)",
			mfs: func() *timeseries.MultiFrame {
				s := timeseries.NewMultiFrame()

				err := s.AddSeries("one", nil, []time.Time{{}, time.Now().Add(time.Second)}, []float64{0, 1})
				require.NoError(t, err)

				(*s)[0].Fields = append((*s)[0].Fields, data.NewField("a", nil, []float64{2, 3}))
				(*s)[0].Fields = append((*s)[0].Fields, data.NewField("a", nil, []string{"4", "cats"}))
				return s
			},
			ignoredFieldIndices: []sdata.FrameFieldIndex{
				{FrameIdx: 0, FieldIdx: 2, Reason: "additional numeric value field"},
				{FrameIdx: 0, FieldIdx: 3, Reason: "unsupported field type []string"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ignoredFieldIndices, err := tt.mfs().Validate(true)
			require.Nil(t, err)
			require.Equal(t, tt.ignoredFieldIndices, ignoredFieldIndices)
		})
	}
}

func TestMultiFrameSeriesValidate_WithFrames_InvalidCases(t *testing.T) {
	tests := []struct {
		name        string
		mfs         *timeseries.MultiFrame
		errContains string
		dataOnly    bool
	}{
		{
			name: "frame must have type indicator",
			mfs: &timeseries.MultiFrame{
				data.NewFrame(""),
			},
			errContains: "missing a type indicator",
		},
		{
			name: "frame with only value field is not valid, missing time field",
			mfs: &timeseries.MultiFrame{
				addFields(emptyFrameWithTypeMD(data.FrameTypeTimeSeriesMany),
					data.NewField("", nil, []float64{})),
			},
			errContains: "missing a []time.Time field",
		},
		{
			name: "frame with only a time field and no value is not valid",
			mfs: &timeseries.MultiFrame{
				addFields(emptyFrameWithTypeMD(data.FrameTypeTimeSeriesMany),
					data.NewField("", nil, []time.Time{})),
			},
			errContains: "must have at least one value field",
		},
		{
			name: "fields must be of the same length",
			mfs: &timeseries.MultiFrame{
				addFields(emptyFrameWithTypeMD(data.FrameTypeTimeSeriesMany),
					data.NewField("", nil, []float64{1, 2}),
					data.NewField("", nil, []time.Time{time.UnixMilli(1)})),
			},
			errContains: "mismatched field lengths",
		},
		{
			name: "frame with unsorted time is not valid",
			mfs: &timeseries.MultiFrame{
				addFields(emptyFrameWithTypeMD(data.FrameTypeTimeSeriesMany),
					data.NewField("", nil, []float64{1, 2}),
					data.NewField("", nil, []time.Time{time.UnixMilli(2), time.UnixMilli(1)})),
			},
			errContains: "unsorted time",
			dataOnly:    true,
		},
		{
			name: "duplicate metrics as identified by name + labels are invalid",
			mfs: &timeseries.MultiFrame{
				addFields(emptyFrameWithTypeMD(data.FrameTypeTimeSeriesMany),
					data.NewField("os.cpu", data.Labels{"host": "a", "iface": "eth0"}, []float64{1, 2}),
					data.NewField("", nil, []time.Time{time.UnixMilli(1), time.UnixMilli(2)})),
				addFields(emptyFrameWithTypeMD(data.FrameTypeTimeSeriesMany),
					data.NewField("os.cpu", data.Labels{"iface": "eth0", "host": "a"}, []float64{1, 2}),
					data.NewField("", nil, []time.Time{time.UnixMilli(1), time.UnixMilli(2)})),
			},
			errContains: "duplicate metrics found",
			dataOnly:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ignoredFieldIndices, err := tt.mfs.Validate(true)
			require.True(t, strings.Contains(err.Error(), tt.errContains), fmt.Sprintf("error '%v' does not contain '%v'", err.Error(), tt.errContains))
			require.Nil(t, ignoredFieldIndices)

			// If the test does not have dataOnly, make sure it is the same with Validate(false)
			if !tt.dataOnly {
				ignoredFieldIndices, err := tt.mfs.Validate(false)
				require.True(t, strings.Contains(err.Error(), tt.errContains), fmt.Sprintf("error '%v' does not contain '%v'", err.Error(), tt.errContains))
				require.Nil(t, ignoredFieldIndices)
			}

			// Also check that GetMetricRefs returns matching errors when not checking data
			refs, ignoredFieldIndices, err := tt.mfs.GetMetricRefs()
			if !tt.dataOnly {
				require.Error(t, err)
				require.True(t, strings.Contains(err.Error(), tt.errContains), fmt.Sprintf("error '%v' does not contain '%v'", err.Error(), tt.errContains))
				require.Nil(t, ignoredFieldIndices)
				require.Nil(t, refs)
			}
		})
	}
}

func emptyFrameWithTypeMD(t data.FrameType) *data.Frame {
	return data.NewFrame("").SetMeta(&data.FrameMeta{Type: t})
}

var _ = emptyFrameWithTypeMD(data.FrameTypeUnknown) // linter

func TestMultiFrameSeriesGetMetricRefs_Empty_Invalid_Edge_Cases(t *testing.T) {
	t.Run("empty response reads as zero length metric refs and nil ignoredFields", func(t *testing.T) {
		s := timeseries.NewMultiFrame()

		refs, ignoredFieldIndices, err := s.GetMetricRefs()
		require.Nil(t, err)

		require.Nil(t, ignoredFieldIndices)
		require.NotNil(t, refs)
		require.Len(t, refs, 0)
	})

	t.Run("empty response frame with an additional frames cause additional frames to be ignored", func(t *testing.T) {
		s := timeseries.NewMultiFrame()

		// (s.AddMetric) would alter the first frame which would be the "right thing" to do.
		*s = append(*s, emptyFrameWithTypeMD(data.FrameTypeTimeSeriesMany))
		(*s)[1].Fields = append((*s)[1].Fields,
			data.NewField("time", nil, []time.Time{}),
			data.NewField("cpu", nil, []float64{}),
		)

		refs, ignoredFieldIndices, err := s.GetMetricRefs()
		require.NoError(t, err)
		require.Len(t, refs, 0)
		require.Equal(t, []sdata.FrameFieldIndex{
			{FrameIdx: 1, FieldIdx: 0, Reason: "extra frame on empty response"},
			{FrameIdx: 1, FieldIdx: 1, Reason: "extra frame on empty response"},
		}, ignoredFieldIndices)
	})

	t.Run("uninitialized frames returns nil refs and nil ignored", func(t *testing.T) {
		s := timeseries.MultiFrame{}

		refs, ignoredFieldIndices, err := s.GetMetricRefs()
		require.Error(t, err)

		require.Nil(t, ignoredFieldIndices)
		require.Nil(t, refs)
	})

	t.Run("a nil frame (a nil entry in slice of frames (very odd)), is not a valid in a response", func(t *testing.T) {
		s := timeseries.NewMultiFrame()
		*s = append(*s, nil)

		refs, ignoredFieldIndices, err := s.GetMetricRefs()
		require.Nil(t, refs)
		require.Nil(t, ignoredFieldIndices)
		require.Error(t, err)
	})

	t.Run("no type metadata means error if first", func(t *testing.T) {
		s := timeseries.MultiFrame{
			data.NewFrame("",
				data.NewField("", nil, []time.Time{}),
				data.NewField("foo", nil, []float64{}),
			)}

		refs, ignoredFieldIndices, err := s.GetMetricRefs()

		require.Nil(t, refs)
		require.Nil(t, ignoredFieldIndices)
		require.Error(t, err)
	})
}
