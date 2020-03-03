package dataframe_test

import (
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/dataframe"
	"github.com/stretchr/testify/require"
)

func TestTimeSeriesType(t *testing.T) {
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
			tsType := tt.frame.TimeSeriesType()
			require.Equal(t, tt.tsType.String(), tsType.String())
		})
	}
}
