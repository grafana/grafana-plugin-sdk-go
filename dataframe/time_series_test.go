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
			name: "two time values is a wide series",
			frame: dataframe.New("test", dataframe.NewField("timeValues", nil, []time.Time{time.Time{}}),
				dataframe.NewField("moreTimeValues", nil, []time.Time{time.Time{}})),
			tsType: dataframe.TimeSeriesTypeWide,
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
				dataframe.NewField(`Values Floats`, dataframe.Labels{"Animal Factor": "cat"}, []float64{
					1.0,
					3.0,
				}),
				dataframe.NewField(`Values Floats`, dataframe.Labels{"Animal Factor": "sloth"}, []float64{
					2.0,
					4.0,
				})),
			Err: require.NoError,
		},
		{
			name: "one value, two factors",
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
				}),
				dataframe.NewField("Location", nil, []string{
					"Florida",
					"Central & South America",
					"Florida",
					"Central & South America",
				})),

			wideFrame: dataframe.New("long_to_wide_test",
				dataframe.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
				}),
				dataframe.NewField(`Values Floats`,
					dataframe.Labels{"Animal Factor": "cat", "Location": "Florida"}, []float64{
						1.0,
						3.0,
					}),
				dataframe.NewField(`Values Floats`,
					dataframe.Labels{"Animal Factor": "sloth", "Location": "Central & South America"}, []float64{
						2.0,
						4.0,
					})),
			Err: require.NoError,
		},
		{
			name: "two values, one factor",
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
				dataframe.NewField("Values Int64", nil, []int64{
					1,
					2,
					3,
					4,
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
				dataframe.NewField(`Values Floats`, dataframe.Labels{"Animal Factor": "cat"}, []float64{
					1.0,
					3.0,
				}),
				dataframe.NewField(`Values Int64`, dataframe.Labels{"Animal Factor": "cat"}, []int64{
					1,
					3,
				}),
				dataframe.NewField(`Values Floats`, dataframe.Labels{"Animal Factor": "sloth"}, []float64{
					2.0,
					4.0,
				}),
				dataframe.NewField(`Values Int64`, dataframe.Labels{"Animal Factor": "sloth"}, []int64{
					2,
					4,
				})),
			Err: require.NoError,
		},
		{
			name: "two values, two factor",
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
				dataframe.NewField("Values Int64", nil, []int64{
					1,
					2,
					3,
					4,
				}),
				dataframe.NewField("Animal Factor", nil, []string{
					"cat",
					"sloth",
					"cat",
					"sloth",
				}),
				dataframe.NewField("Location", nil, []string{
					"Florida",
					"Central & South America",
					"Florida",
					"Central & South America",
				})),

			wideFrame: dataframe.New("long_to_wide_test",
				dataframe.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
				}),
				dataframe.NewField(`Values Floats`, dataframe.Labels{"Animal Factor": "cat", "Location": "Florida"}, []float64{
					1.0,
					3.0,
				}),
				dataframe.NewField(`Values Int64`, dataframe.Labels{"Animal Factor": "cat", "Location": "Florida"}, []int64{
					1,
					3,
				}),
				dataframe.NewField(`Values Floats`, dataframe.Labels{"Animal Factor": "sloth", "Location": "Central & South America"}, []float64{
					2.0,
					4.0,
				}),
				dataframe.NewField(`Values Int64`, dataframe.Labels{"Animal Factor": "sloth", "Location": "Central & South America"}, []int64{
					2,
					4,
				})),
			Err: require.NoError,
		},
		{
			name: "pointers: one value, one factor. Time becomes non-pointer since null time not supported",
			longFrame: dataframe.New("long_to_wide_test",
				dataframe.NewField("Time", nil, []*time.Time{
					timePtr(time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC)),
					timePtr(time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC)),
					timePtr(time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC)),
					timePtr(time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC)),
				}),
				dataframe.NewField("Values Floats", nil, []*float64{
					float64Ptr(1.0),
					float64Ptr(2.0),
					float64Ptr(3.0),
					float64Ptr(4.0),
				}),
				dataframe.NewField("Animal Factor", nil, []*string{
					stringPtr("cat"),
					stringPtr("sloth"),
					stringPtr("cat"),
					stringPtr("sloth"),
				})),

			wideFrame: dataframe.New("long_to_wide_test",
				dataframe.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
				}),
				dataframe.NewField(`Values Floats`, dataframe.Labels{"Animal Factor": "cat"}, []*float64{
					float64Ptr(1.0),
					float64Ptr(3.0),
				}),
				dataframe.NewField(`Values Floats`, dataframe.Labels{"Animal Factor": "sloth"}, []*float64{
					float64Ptr(2.0),
					float64Ptr(4.0),
				})),
			Err: require.NoError,
		},
		{
			name: "sparse: one value, two factor",
			longFrame: dataframe.New("long_to_wide_test",
				dataframe.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 5, 0, 0, time.UTC), // single time sample
					time.Date(2020, 1, 2, 3, 5, 30, 0, time.UTC),
				}),
				dataframe.NewField("Values Floats", nil, []float64{
					1.0,
					2.0,
					3.0,
					4.0,
					55.0,
					6.0,
				}),

				dataframe.NewField("Animal Factor", nil, []string{
					"cat",
					"sloth",
					"cat",
					"sloth",
					"pangolin", // single factor sample
					"sloth",
				}),
				dataframe.NewField("Location", nil, []string{
					"Florida",
					"Central & South America",
					"Florida",
					"Central & South America",
					"", // single factor sample
					"Central & South America",
				})),
			wideFrame: dataframe.New("long_to_wide_test",
				dataframe.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 5, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 5, 30, 0, time.UTC),
				}),
				dataframe.NewField(`Values Floats`, dataframe.Labels{"Animal Factor": "cat", "Location": "Florida"}, []float64{
					1.0,
					3.0,
					0.0,
					0.0,
				}),
				dataframe.NewField(`Values Floats`, dataframe.Labels{"Animal Factor": "sloth", "Location": "Central & South America"}, []float64{
					2.0,
					4.0,
					0.0,
					6.0,
				}),
				dataframe.NewField(`Values Floats`, dataframe.Labels{"Animal Factor": "pangolin", "Location": ""}, []float64{
					0.0,
					0.0,
					55.0,
					0.0,
				})),
			Err: require.NoError,
		},
		{
			name: "sparse & pointer: one value, two factor",
			longFrame: dataframe.New("long_to_wide_test",
				dataframe.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 5, 0, 0, time.UTC), // single time sample
					time.Date(2020, 1, 2, 3, 5, 30, 0, time.UTC),
				}),
				dataframe.NewField("Values Floats", nil, []*float64{
					float64Ptr(1.0),
					float64Ptr(2.0),
					float64Ptr(3.0),
					float64Ptr(4.0),
					float64Ptr(55.0),
					float64Ptr(6.0),
				}),

				dataframe.NewField("Animal Factor", nil, []string{
					"cat",
					"sloth",
					"cat",
					"sloth",
					"pangolin", // single factor sample
					"sloth",
				}),
				dataframe.NewField("Location", nil, []*string{
					stringPtr("Florida"),
					stringPtr("Central & South America"),
					stringPtr("Florida"),
					stringPtr("Central & South America"),
					nil, // single factor sample
					stringPtr("Central & South America"),
				})),
			wideFrame: dataframe.New("long_to_wide_test",
				dataframe.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 5, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 5, 30, 0, time.UTC),
				}),
				dataframe.NewField(`Values Floats`, dataframe.Labels{"Animal Factor": "cat", "Location": "Florida"}, []*float64{
					float64Ptr(1.0),
					float64Ptr(3.0),
					nil,
					nil,
				}),
				dataframe.NewField(`Values Floats`, dataframe.Labels{"Animal Factor": "sloth", "Location": "Central & South America"}, []*float64{
					float64Ptr(2.0),
					float64Ptr(4.0),
					nil,
					float64Ptr(6.0),
				}),
				dataframe.NewField(`Values Floats`, dataframe.Labels{"Animal Factor": "pangolin", "Location": ""}, []*float64{
					nil,
					nil,
					float64Ptr(55.0),
					nil,
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
		})
	}
}

func TestWideToLong(t *testing.T) {
	tests := []struct {
		name      string
		wideFrame *dataframe.Frame
		longFrame *dataframe.Frame
		Err       require.ErrorAssertionFunc
	}{
		{
			name: "one value, one factor",
			wideFrame: dataframe.New("wide_to_long_test",
				dataframe.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
				}),
				dataframe.NewField(`Values Floats`, dataframe.Labels{"Animal Factor": "cat"}, []float64{
					1.0,
					3.0,
				}),
				dataframe.NewField(`Values Floats`, dataframe.Labels{"Animal Factor": "sloth"}, []float64{
					2.0,
					4.0,
				})),

			longFrame: dataframe.New("wide_to_long_test",
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
			Err: require.NoError,
		},

		{
			name: "one value, two factors",
			wideFrame: dataframe.New("wide_to_long_test",
				dataframe.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
				}),
				dataframe.NewField(`Values Floats`,
					dataframe.Labels{"Animal Factor": "cat", "Location": "Florida"}, []float64{
						1.0,
						3.0,
					}),
				dataframe.NewField(`Values Floats`,
					dataframe.Labels{"Animal Factor": "sloth", "Location": "Central & South America"}, []float64{
						2.0,
						4.0,
					})),
			Err: require.NoError,

			longFrame: dataframe.New("wide_to_long_test",
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
				}),
				dataframe.NewField("Location", nil, []string{
					"Florida",
					"Central & South America",
					"Florida",
					"Central & South America",
				})),
		},
		{
			name: "two values, one factor",
			wideFrame: dataframe.New("wide_to_long_test",
				dataframe.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
				}),
				dataframe.NewField(`Values Floats`, dataframe.Labels{"Animal Factor": "cat"}, []float64{
					1.0,
					3.0,
				}),
				dataframe.NewField(`Values Int64`, dataframe.Labels{"Animal Factor": "cat"}, []int64{
					1,
					3,
				}),
				dataframe.NewField(`Values Floats`, dataframe.Labels{"Animal Factor": "sloth"}, []float64{
					2.0,
					4.0,
				}),
				dataframe.NewField(`Values Int64`, dataframe.Labels{"Animal Factor": "sloth"}, []int64{
					2,
					4,
				})),

			longFrame: dataframe.New("wide_to_long_test",
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
				dataframe.NewField("Values Int64", nil, []int64{
					1,
					2,
					3,
					4,
				}),
				dataframe.NewField("Animal Factor", nil, []string{
					"cat",
					"sloth",
					"cat",
					"sloth",
				})),
			Err: require.NoError,
		},
		{
			name: "two values, two factor",
			wideFrame: dataframe.New("wide_to_long_test",
				dataframe.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
				}),
				dataframe.NewField(`Values Floats`, dataframe.Labels{"Animal Factor": "cat", "Location": "Florida"}, []float64{
					1.0,
					3.0,
				}),
				dataframe.NewField(`Values Int64`, dataframe.Labels{"Animal Factor": "cat", "Location": "Florida"}, []int64{
					1,
					3,
				}),
				dataframe.NewField(`Values Floats`, dataframe.Labels{"Animal Factor": "sloth", "Location": "Central & South America"}, []float64{
					2.0,
					4.0,
				}),
				dataframe.NewField(`Values Int64`, dataframe.Labels{"Animal Factor": "sloth", "Location": "Central & South America"}, []int64{
					2,
					4,
				})),

			longFrame: dataframe.New("wide_to_long_test",
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
				dataframe.NewField("Values Int64", nil, []int64{
					1,
					2,
					3,
					4,
				}),
				dataframe.NewField("Animal Factor", nil, []string{
					"cat",
					"sloth",
					"cat",
					"sloth",
				}),
				dataframe.NewField("Location", nil, []string{
					"Florida",
					"Central & South America",
					"Florida",
					"Central & South America",
				})),
			Err: require.NoError,
		},
		{
			name: "pointers: one value, one factor. Time becomes non-pointer since null time not supported",
			wideFrame: dataframe.New("wide_to_long_test",
				dataframe.NewField("Time", nil, []*time.Time{
					timePtr(time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC)),
					timePtr(time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC)),
				}),
				dataframe.NewField(`Values Floats`, dataframe.Labels{"Animal Factor": "cat"}, []*float64{
					float64Ptr(1.0),
					float64Ptr(3.0),
				}),
				dataframe.NewField(`Values Floats`, dataframe.Labels{"Animal Factor": "sloth"}, []*float64{
					float64Ptr(2.0),
					float64Ptr(4.0),
				})),
			Err: require.NoError,

			longFrame: dataframe.New("wide_to_long_test",
				dataframe.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
				}),
				dataframe.NewField("Values Floats", nil, []*float64{
					float64Ptr(1.0),
					float64Ptr(2.0),
					float64Ptr(3.0),
					float64Ptr(4.0),
				}),
				dataframe.NewField("Animal Factor", nil, []string{
					"cat",
					"sloth",
					"cat",
					"sloth",
				})),
		},
		{
			name: "sparse: one value, two factor",
			wideFrame: dataframe.New("wide_to_long_test",
				dataframe.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 5, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 5, 30, 0, time.UTC),
				}),
				dataframe.NewField(`Values Floats`, dataframe.Labels{"Animal Factor": "cat", "Location": "Florida"}, []float64{
					1.0,
					3.0,
					0.0,
					0.0,
				}),
				dataframe.NewField(`Values Floats`, dataframe.Labels{"Animal Factor": "sloth", "Location": "Central & South America"}, []float64{
					2.0,
					4.0,
					0.0,
					6.0,
				}),
				dataframe.NewField(`Values Floats`, dataframe.Labels{"Animal Factor": "pangolin", "Location": ""}, []float64{
					0.0,
					0.0,
					55.0,
					0.0,
				})),

			longFrame: dataframe.New("wide_to_long_test",
				dataframe.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 5, 0, 0, time.UTC), // single time sample
					time.Date(2020, 1, 2, 3, 5, 0, 0, time.UTC), // single time sample
					time.Date(2020, 1, 2, 3, 5, 0, 0, time.UTC), // single time sample
					time.Date(2020, 1, 2, 3, 5, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 5, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 5, 30, 0, time.UTC),
				}),
				dataframe.NewField("Values Floats", nil, []float64{
					1.0,
					0.0,
					2.0,
					3.0,
					0.0,
					4.0,
					0.0,
					55.0,
					0.0,
					0.0,
					0.0,
					6.0,
				}),

				dataframe.NewField("Animal Factor", nil, []string{
					"cat",
					"pangolin",
					"sloth",
					"cat",
					"pangolin",
					"sloth",
					"cat",
					"pangolin",
					"sloth",
					"cat",
					"pangolin",
					"sloth",
				}),
				dataframe.NewField("Location", nil, []string{
					"Florida",
					"",
					"Central & South America",
					"Florida",
					"",
					"Central & South America",
					"Florida",
					"",
					"Central & South America",
					"Florida",
					"",
					"Central & South America",
				})),

			Err: require.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame, err := dataframe.WideToLong(tt.wideFrame)
			tt.Err(t, err)
			if diff := cmp.Diff(tt.longFrame, frame); diff != "" {
				t.Errorf("Result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
