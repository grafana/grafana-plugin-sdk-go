package sdata_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/sdata"
	"github.com/stretchr/testify/require"
)

func TestPlayGround(t *testing.T) {
	var mfs sdata.MultiFrameSeries
	mfs.AddMetric("os.cpu", data.Labels{"host": "a"}, []time.Time{time.Unix(1234567890, 0)}, []float64{3})
}

func emptyFrameWithManySeriesMD() *data.Frame {
	return data.NewFrame("").SetMeta(&data.FrameMeta{Type: data.FrameTypeTimeSeriesMany})
}

func addFields(frame *data.Frame, fields ...*data.Field) *data.Frame {
	frame.Fields = append(frame.Fields, fields...)
	return frame
}

func TestMultiFrameSeriesValidiate_WithFrames_InvalidCases(t *testing.T) {
	tests := []struct {
		name          string
		mfs           *sdata.MultiFrameSeries
		empty         bool
		errCount      int
		errorsContain []string
	}{
		{
			name: "frame must have type indicator",
			mfs: &sdata.MultiFrameSeries{
				data.NewFrame(""),
			},
			errCount:      1,
			errorsContain: []string{"missing type indicator"},
		},
		{
			name: "frame with only value field is not valid, missing time field",
			mfs: &sdata.MultiFrameSeries{
				addFields(emptyFrameWithManySeriesMD(),
					data.NewField("", nil, []float64{})),
			},
			errCount:      1,
			errorsContain: []string{"must have at least one time field"},
		},
		{
			name: "frame with only a time field and no value is not valid",
			mfs: &sdata.MultiFrameSeries{
				addFields(emptyFrameWithManySeriesMD(),
					data.NewField("", nil, []time.Time{})),
			},
			errCount:      1,
			errorsContain: []string{"must have at least one value field"},
		},
		{
			name: "fields must be of the same length",
			mfs: &sdata.MultiFrameSeries{
				addFields(emptyFrameWithManySeriesMD(),
					data.NewField("", nil, []float64{1, 2}),
					data.NewField("", nil, []time.Time{time.UnixMilli(1)})),
			},
			errCount:      1,
			errorsContain: []string{"mismatched field lengths"},
		},
		{
			name: "frame with unsorted time is not valid",
			mfs: &sdata.MultiFrameSeries{
				addFields(emptyFrameWithManySeriesMD(),
					data.NewField("", nil, []float64{1, 2}),
					data.NewField("", nil, []time.Time{time.UnixMilli(2), time.UnixMilli(1)})),
			},
			errCount:      1,
			errorsContain: []string{"unsorted time"},
		},
		{
			name: "duplicate metrics as identified by name + labes are invalid",
			mfs: &sdata.MultiFrameSeries{
				addFields(emptyFrameWithManySeriesMD(),
					data.NewField("os.cpu", data.Labels{"host": "a", "iface": "eth0"}, []float64{1, 2}),
					data.NewField("", nil, []time.Time{time.UnixMilli(1), time.UnixMilli(2)})),
				addFields(emptyFrameWithManySeriesMD(),
					data.NewField("os.cpu", data.Labels{"iface": "eth0", "host": "a"}, []float64{1, 2}),
					data.NewField("", nil, []time.Time{time.UnixMilli(1), time.UnixMilli(2)})),
			},
			errCount:      1,
			errorsContain: []string{"duplicate metrics found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			empty, ignoredFieldIndices, errors := tt.mfs.Validate()
			for _, errSubStr := range tt.errorsContain {
				foundErr := false
				for _, err := range errors {
					if strings.Contains(err.Error(), errSubStr) {
						foundErr = true
						break
					}
				}
				require.True(t, foundErr, "expected error substring %q not found, errors: %v", errSubStr, errors)
			}
			require.Equal(t, tt.errCount, len(errors), "expected %v validation errors, errors: %v", tt.errCount, errors)
			require.Nil(t, ignoredFieldIndices)
			require.Equal(t, tt.empty, empty, "expected valid to be %v", tt.empty)
		})
	}
}

func TestMultiFrameSeriesValidate_WithFrames_ValidCases(t *testing.T) {
	tests := []struct {
		name                string
		mfs                 *sdata.MultiFrameSeries
		empty               bool
		ignoredFieldIndices []sdata.FrameFieldIndex
	}{
		{
			name:  "nil or empty set is valid and empty",
			empty: true,
		},
		{
			name: "frame with no fields is valid, and does not mean set is empty",
			mfs: &sdata.MultiFrameSeries{
				emptyFrameWithManySeriesMD(),
			},
		},
		{
			name: "frame with unsorted time is not valid",
			mfs: &sdata.MultiFrameSeries{
				addFields(emptyFrameWithManySeriesMD(),
					data.NewField("", nil, []bool{true, false}),
					data.NewField("", nil, []time.Time{time.UnixMilli(1), time.UnixMilli(2)})),
			},
		},
		{
			name: "there can be extraneous fields (but they have no specific platform-wide meaning)",
			mfs: &sdata.MultiFrameSeries{
				addFields(emptyFrameWithManySeriesMD(),
					data.NewField("", nil, []int64{2, 3}),
					data.NewField("", nil, []float64{2, 3}),
					data.NewField("", nil, []time.Time{time.UnixMilli(1), time.UnixMilli(2)}),
					data.NewField("", nil, []time.Time{time.UnixMilli(5), time.UnixMilli(12)}),
					data.NewField("", nil, []string{"fair", "enough?"})),
			},
			ignoredFieldIndices: []sdata.FrameFieldIndex{
				{0, 1}, {0, 3}, {0, 4},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			empty, ignoredFieldIndices, errors := tt.mfs.Validate()
			require.Zero(t, errors)
			require.Equal(t, tt.empty, empty, "expected valid to be %v", tt.empty)
			require.Equal(t, tt.ignoredFieldIndices, ignoredFieldIndices)
		})
	}
}

func TestWideFrameAddMetric_ValidCases(t *testing.T) {
	t.Run("add two metrics", func(t *testing.T) {
		wf := sdata.WideFrameSeries{}

		err := wf.AddMetric("one", data.Labels{"host": "a"}, []time.Time{time.UnixMilli(1), time.UnixMilli(2)}, []float64{1, 2})
		require.NoError(t, err)

		err = wf.AddMetric("one", data.Labels{"host": "b"}, nil, []float64{3, 4})
		require.NoError(t, err)

		expectedFrame := data.NewFrame("",
			data.NewField("time", nil, []time.Time{time.UnixMilli(1), time.UnixMilli(2)}),
			data.NewField("one", data.Labels{"host": "a"}, []float64{1, 2}),
			data.NewField("one", data.Labels{"host": "b"}, []float64{3, 4}),
		).SetMeta(&data.FrameMeta{Type: data.FrameTypeTimeSeriesWide})

		if diff := cmp.Diff(expectedFrame, wf.Frame, data.FrameTestCompareOptions()...); diff != "" {
			require.FailNow(t, "mismatch (-want +got):\n%s\n", diff)
		}
	})
}

func TestWideFrameSeriesGetMetricRefs(t *testing.T) {
	t.Run("two metrics from wide to multi", func(t *testing.T) {
		wf := sdata.WideFrameSeries{}

		err := wf.AddMetric("one", data.Labels{"host": "a"}, []time.Time{time.UnixMilli(1), time.UnixMilli(2)}, []float64{1, 2})
		require.NoError(t, err)

		err = wf.AddMetric("one", data.Labels{"host": "b"}, nil, []float64{3, 4})
		require.NoError(t, err)
		refs, ignoredFields := wf.GetMetricRefs()

		expectedRefs := []sdata.TimeSeriesMetricRef{
			{
				ValueField: data.NewField("one", data.Labels{"host": "a"}, []float64{1, 2}),
				TimeField:  data.NewField("time", nil, []time.Time{time.UnixMilli(1), time.UnixMilli(2)}),
			},
			{
				ValueField: data.NewField("one", data.Labels{"host": "b"}, []float64{3, 4}),
				TimeField:  data.NewField("time", nil, []time.Time{time.UnixMilli(1), time.UnixMilli(2)}),
			},
		}

		require.Empty(t, ignoredFields) // TODO more specific []x{} vs nil

		if diff := cmp.Diff(expectedRefs, refs, data.FrameTestCompareOptions()...); diff != "" {
			require.FailNow(t, "mismatch (-want +got):\n%s\n", diff)
		}
	})
}

func TestLongSeriesGetMetricRefs(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		ls := sdata.LongSeries{
			Frame: data.NewFrame("",
				data.NewField("time", nil, []time.Time{time.UnixMilli(1), time.UnixMilli(1)}),
				data.NewField("host", nil, []string{"a", "b"}),
				data.NewField("iface", nil, []string{"eth0", "eth0"}),
				data.NewField("in_bytes", nil, []float64{1, 2}),
				data.NewField("out_bytes", nil, []int64{3, 4}),
			).SetMeta(&data.FrameMeta{Type: data.FrameTypeTimeSeriesLong}),
		}

		refs, ignoredFields := ls.GetMetricRefs()

		expectedRefs := []sdata.TimeSeriesMetricRef{
			{
				ValueField: data.NewField("in_bytes", data.Labels{"host": "a", "iface": "eth0"}, []float64{1}),
				TimeField:  data.NewField("time", nil, []time.Time{time.UnixMilli(1)}),
			},
			{
				ValueField: data.NewField("in_bytes", data.Labels{"host": "b", "iface": "eth0"}, []float64{2}),
				TimeField:  data.NewField("time", nil, []time.Time{time.UnixMilli(1)}),
			},
			{
				ValueField: data.NewField("out_bytes", data.Labels{"host": "a", "iface": "eth0"}, []int64{3}),
				TimeField:  data.NewField("time", nil, []time.Time{time.UnixMilli(1)}),
			},
			{
				ValueField: data.NewField("out_bytes", data.Labels{"host": "b", "iface": "eth0"}, []int64{4}),
				TimeField:  data.NewField("time", nil, []time.Time{time.UnixMilli(1)}),
			},
		}

		require.Empty(t, ignoredFields) // TODO more specific []x{} vs nil

		if diff := cmp.Diff(expectedRefs, refs, data.FrameTestCompareOptions()...); diff != "" {
			require.FailNow(t, "mismatch (-want +got):\n", diff)
		}
	})
}
