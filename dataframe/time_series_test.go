package dataframe_test

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/grafana/grafana-plugin-sdk-go/dataframe"
	"github.com/stretchr/testify/require"
)

func TestTimeSeriesSchema(t *testing.T) {
	tests := []struct {
		name   string
		frame  *dataframe.Frame
		tsType dataframe.TimeSeriesType
	}{
		{
			name:   "empty frame is not a time series",
			frame:  &dataframe.Frame{},
			tsType: dataframe.TimeSeriesTypeNot,
		},
		{
			name:   "time field only is not a time series",
			frame:  dataframe.New("test", dataframe.NewField("timeValues", nil, []time.Time{time.Time{}})),
			tsType: dataframe.TimeSeriesTypeNot,
		},
		{
			name: "more than one time field is not a time series",
			frame: dataframe.New("test", dataframe.NewField("timeValues", nil, []time.Time{time.Time{}}),
				dataframe.NewField("moreTimeValues", nil, []time.Time{time.Time{}})),
			tsType: dataframe.TimeSeriesTypeNot,
		},
		{
			name:   "simple wide time series",
			frame:  dataframe.New("test", dataframe.NewField("timeValues", nil, []time.Time{time.Time{}}), dataframe.NewField("floatValues", nil, []float64{1.0})),
			tsType: dataframe.TimeSeriesTypeWide,
		},
		{
			name: "simple long time series with one facet",
			frame: dataframe.New("test", dataframe.NewField("timeValues", nil, []time.Time{time.Time{}}),
				dataframe.NewField("floatValues", nil, []float64{1.0}),
				dataframe.NewField("user", nil, []string{"Lord Slothius"})),
			tsType: dataframe.TimeSeriesTypeLong,
		},
		{
			name: "multi-value wide time series",
			frame: dataframe.New("test", dataframe.NewField("floatValues", nil, []float64{1.0}),
				dataframe.NewField("timeValues", nil, []time.Time{time.Time{}}),
				dataframe.NewField("int64 Values", nil, []int64{1})),
			tsType: dataframe.TimeSeriesTypeWide,
		},
		{
			name: "multi-value and multi-facet long series",
			frame: dataframe.New("test", dataframe.NewField("floatValues", nil, []float64{1.0}),
				dataframe.NewField("timeValues", nil, []time.Time{time.Time{}}),
				dataframe.NewField("int64 Values", nil, []int64{1}),
				dataframe.NewField("user", nil, []string{"Lord Slothius"}),
				dataframe.NewField("location", nil, []string{"Slothingham"})),
			tsType: dataframe.TimeSeriesTypeLong,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tsSchema := tt.frame.TimeSeriesSchema()
			require.Equal(t, tt.tsType.String(), tsSchema.Type.String())
		})
	}
}

func TestLongToWide(t *testing.T) {
	tests := []struct {
		name      string
		longFrame *dataframe.Frame
		wideFrame *dataframe.Frame
		Err       require.ErrorAssertionFunc
	}{
		{
			name: "one value, one factor",
			longFrame: dataframe.New("long_to_wide_test",
				dataframe.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
				}),
				dataframe.NewField("Values Floats", nil, []float64{
					1.0,
					2.0,
					3.0,
					4.0,
				}),
				dataframe.NewField("Animal Factor", nil, []string{
					"cat",
					"sloth",
					"cat",
					"sloth",
				})),

			wideFrame: dataframe.New("long_to_wide_test",
				dataframe.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
				}),
				dataframe.NewField(`Values Floats[["Animal Factor","cat"]]`, dataframe.Labels{"Animal Factor": "cat"}, []float64{
					1.0,
					3.0,
				}),
				dataframe.NewField(`Values Floats[["Animal Factor","sloth"]]`, dataframe.Labels{"Animal Factor": "sloth"}, []float64{
					2.0,
					4.0,
				})),
			Err: require.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame, err := dataframe.LongToWide(tt.longFrame)
			tt.Err(t, err)
			if diff := cmp.Diff(tt.wideFrame, frame); diff != "" {
				t.Errorf("Result mismatch (-want +got):\n%s", diff)
			}
			//spew.Dump(frame)
		})
	}
}
