package data_test

import (
	"database/sql"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/require"
)

func TestTimeSeriesSchema(t *testing.T) {
	tests := []struct {
		name   string
		frame  *data.Frame
		tsType data.TimeSeriesType
	}{
		{
			name:   "empty frame is not a time series",
			frame:  &data.Frame{},
			tsType: data.TimeSeriesTypeNot,
		},
		{
			name:   "time field only is not a time series",
			frame:  data.NewFrame("test", data.NewField("timeValues", nil, []time.Time{time.Time{}})),
			tsType: data.TimeSeriesTypeNot,
		},
		{
			name: "two time values is a wide series",
			frame: data.NewFrame("test", data.NewField("timeValues", nil, []time.Time{time.Time{}}),
				data.NewField("moreTimeValues", nil, []time.Time{time.Time{}})),
			tsType: data.TimeSeriesTypeWide,
		},
		{
			name:   "simple wide time series",
			frame:  data.NewFrame("test", data.NewField("timeValues", nil, []time.Time{time.Time{}}), data.NewField("floatValues", nil, []float64{1.0})),
			tsType: data.TimeSeriesTypeWide,
		},
		{
			name: "simple long time series with one facet",
			frame: data.NewFrame("test", data.NewField("timeValues", nil, []time.Time{time.Time{}}),
				data.NewField("floatValues", nil, []float64{1.0}),
				data.NewField("user", nil, []string{"Lord Slothius"})),
			tsType: data.TimeSeriesTypeLong,
		},
		{
			name: "multi-value wide time series",
			frame: data.NewFrame("test", data.NewField("floatValues", nil, []float64{1.0}),
				data.NewField("timeValues", nil, []time.Time{time.Time{}}),
				data.NewField("int64 Values", nil, []int64{1})),
			tsType: data.TimeSeriesTypeWide,
		},
		{
			name: "multi-value and multi-facet long series",
			frame: data.NewFrame("test", data.NewField("floatValues", nil, []float64{1.0}),
				data.NewField("timeValues", nil, []time.Time{time.Time{}}),
				data.NewField("int64 Values", nil, []int64{1}),
				data.NewField("user", nil, []string{"Lord Slothius"}),
				data.NewField("location", nil, []string{"Slothingham"})),
			tsType: data.TimeSeriesTypeLong,
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
		longFrame *data.Frame
		wideFrame *data.Frame
		Err       require.ErrorAssertionFunc
	}{
		{
			name: "one value, one factor",
			longFrame: data.NewFrame("long_to_wide_test",
				data.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
				}),
				data.NewField("Values Floats", nil, []float64{
					1.0,
					2.0,
					3.0,
					4.0,
				}),
				data.NewField("Animal Factor", nil, []string{
					"cat",
					"sloth",
					"cat",
					"sloth",
				})),

			wideFrame: data.NewFrame("long_to_wide_test",
				data.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
				}),
				data.NewField(`Values Floats`, data.Labels{"Animal Factor": "cat"}, []float64{
					1.0,
					3.0,
				}),
				data.NewField(`Values Floats`, data.Labels{"Animal Factor": "sloth"}, []float64{
					2.0,
					4.0,
				})),
			Err: require.NoError,
		},
		{
			name: "one value, two factors",
			longFrame: data.NewFrame("long_to_wide_test",
				data.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
				}),
				data.NewField("Values Floats", nil, []float64{
					1.0,
					2.0,
					3.0,
					4.0,
				}),
				data.NewField("Animal Factor", nil, []string{
					"cat",
					"sloth",
					"cat",
					"sloth",
				}),
				data.NewField("Location", nil, []string{
					"Florida",
					"Central & South America",
					"Florida",
					"Central & South America",
				})),

			wideFrame: data.NewFrame("long_to_wide_test",
				data.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
				}),
				data.NewField(`Values Floats`,
					data.Labels{"Animal Factor": "cat", "Location": "Florida"}, []float64{
						1.0,
						3.0,
					}),
				data.NewField(`Values Floats`,
					data.Labels{"Animal Factor": "sloth", "Location": "Central & South America"}, []float64{
						2.0,
						4.0,
					})),
			Err: require.NoError,
		},
		{
			name: "two values, one factor",
			longFrame: data.NewFrame("long_to_wide_test",
				data.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
				}),
				data.NewField("Values Floats", nil, []float64{
					1.0,
					2.0,
					3.0,
					4.0,
				}),
				data.NewField("Values Int64", nil, []int64{
					1,
					2,
					3,
					4,
				}),
				data.NewField("Animal Factor", nil, []string{
					"cat",
					"sloth",
					"cat",
					"sloth",
				})),

			wideFrame: data.NewFrame("long_to_wide_test",
				data.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
				}),
				data.NewField(`Values Floats`, data.Labels{"Animal Factor": "cat"}, []float64{
					1.0,
					3.0,
				}),
				data.NewField(`Values Int64`, data.Labels{"Animal Factor": "cat"}, []int64{
					1,
					3,
				}),
				data.NewField(`Values Floats`, data.Labels{"Animal Factor": "sloth"}, []float64{
					2.0,
					4.0,
				}),
				data.NewField(`Values Int64`, data.Labels{"Animal Factor": "sloth"}, []int64{
					2,
					4,
				})),
			Err: require.NoError,
		},
		{
			name: "two values, two factor",
			longFrame: data.NewFrame("long_to_wide_test",
				data.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
				}),
				data.NewField("Values Floats", nil, []float64{
					1.0,
					2.0,
					3.0,
					4.0,
				}),
				data.NewField("Values Int64", nil, []int64{
					1,
					2,
					3,
					4,
				}),
				data.NewField("Animal Factor", nil, []string{
					"cat",
					"sloth",
					"cat",
					"sloth",
				}),
				data.NewField("Location", nil, []string{
					"Florida",
					"Central & South America",
					"Florida",
					"Central & South America",
				})),

			wideFrame: data.NewFrame("long_to_wide_test",
				data.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
				}),
				data.NewField(`Values Floats`, data.Labels{"Animal Factor": "cat", "Location": "Florida"}, []float64{
					1.0,
					3.0,
				}),
				data.NewField(`Values Int64`, data.Labels{"Animal Factor": "cat", "Location": "Florida"}, []int64{
					1,
					3,
				}),
				data.NewField(`Values Floats`, data.Labels{"Animal Factor": "sloth", "Location": "Central & South America"}, []float64{
					2.0,
					4.0,
				}),
				data.NewField(`Values Int64`, data.Labels{"Animal Factor": "sloth", "Location": "Central & South America"}, []int64{
					2,
					4,
				})),
			Err: require.NoError,
		},
		{
			name: "pointers: one value, one factor. Time becomes non-pointer since null time not supported",
			longFrame: data.NewFrame("long_to_wide_test",
				data.NewField("Time", nil, []*time.Time{
					timePtr(time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC)),
					timePtr(time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC)),
					timePtr(time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC)),
					timePtr(time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC)),
				}),
				data.NewField("Values Floats", nil, []*float64{
					float64Ptr(1.0),
					float64Ptr(2.0),
					float64Ptr(3.0),
					float64Ptr(4.0),
				}),
				data.NewField("Animal Factor", nil, []*string{
					stringPtr("cat"),
					stringPtr("sloth"),
					stringPtr("cat"),
					stringPtr("sloth"),
				})),

			wideFrame: data.NewFrame("long_to_wide_test",
				data.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
				}),
				data.NewField(`Values Floats`, data.Labels{"Animal Factor": "cat"}, []*float64{
					float64Ptr(1.0),
					float64Ptr(3.0),
				}),
				data.NewField(`Values Floats`, data.Labels{"Animal Factor": "sloth"}, []*float64{
					float64Ptr(2.0),
					float64Ptr(4.0),
				})),
			Err: require.NoError,
		},
		{
			name: "sparse: one value, two factor",
			longFrame: data.NewFrame("long_to_wide_test",
				data.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 5, 0, 0, time.UTC), // single time sample
					time.Date(2020, 1, 2, 3, 5, 30, 0, time.UTC),
				}),
				data.NewField("Values Floats", nil, []float64{
					1.0,
					2.0,
					3.0,
					4.0,
					55.0,
					6.0,
				}),

				data.NewField("Animal Factor", nil, []string{
					"cat",
					"sloth",
					"cat",
					"sloth",
					"pangolin", // single factor sample
					"sloth",
				}),
				data.NewField("Location", nil, []string{
					"Florida",
					"Central & South America",
					"Florida",
					"Central & South America",
					"", // single factor sample
					"Central & South America",
				})),
			wideFrame: data.NewFrame("long_to_wide_test",
				data.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 5, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 5, 30, 0, time.UTC),
				}),
				data.NewField(`Values Floats`, data.Labels{"Animal Factor": "cat", "Location": "Florida"}, []float64{
					1.0,
					3.0,
					0.0,
					0.0,
				}),
				data.NewField(`Values Floats`, data.Labels{"Animal Factor": "sloth", "Location": "Central & South America"}, []float64{
					2.0,
					4.0,
					0.0,
					6.0,
				}),
				data.NewField(`Values Floats`, data.Labels{"Animal Factor": "pangolin", "Location": ""}, []float64{
					0.0,
					0.0,
					55.0,
					0.0,
				})),
			Err: require.NoError,
		},
		{
			name: "sparse & pointer: one value, two factor",
			longFrame: data.NewFrame("long_to_wide_test",
				data.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 5, 0, 0, time.UTC), // single time sample
					time.Date(2020, 1, 2, 3, 5, 30, 0, time.UTC),
				}),
				data.NewField("Values Floats", nil, []*float64{
					float64Ptr(1.0),
					float64Ptr(2.0),
					float64Ptr(3.0),
					float64Ptr(4.0),
					float64Ptr(55.0),
					float64Ptr(6.0),
				}),

				data.NewField("Animal Factor", nil, []string{
					"cat",
					"sloth",
					"cat",
					"sloth",
					"pangolin", // single factor sample
					"sloth",
				}),
				data.NewField("Location", nil, []*string{
					stringPtr("Florida"),
					stringPtr("Central & South America"),
					stringPtr("Florida"),
					stringPtr("Central & South America"),
					nil, // single factor sample
					stringPtr("Central & South America"),
				})),
			wideFrame: data.NewFrame("long_to_wide_test",
				data.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 5, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 5, 30, 0, time.UTC),
				}),
				data.NewField(`Values Floats`, data.Labels{"Animal Factor": "cat", "Location": "Florida"}, []*float64{
					float64Ptr(1.0),
					float64Ptr(3.0),
					nil,
					nil,
				}),
				data.NewField(`Values Floats`, data.Labels{"Animal Factor": "sloth", "Location": "Central & South America"}, []*float64{
					float64Ptr(2.0),
					float64Ptr(4.0),
					nil,
					float64Ptr(6.0),
				}),
				data.NewField(`Values Floats`, data.Labels{"Animal Factor": "pangolin", "Location": ""}, []*float64{
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
			frame, err := data.LongToWide(tt.longFrame)
			tt.Err(t, err)
			if diff := cmp.Diff(tt.wideFrame, frame, data.FrameTestCompareOptions()...); diff != "" {
				t.Errorf("Result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestWideToLong(t *testing.T) {
	tests := []struct {
		name      string
		wideFrame *data.Frame
		longFrame *data.Frame
		Err       require.ErrorAssertionFunc
	}{
		{
			name: "one value, one factor",
			wideFrame: data.NewFrame("wide_to_long_test",
				data.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
				}),
				data.NewField(`Values Floats`, data.Labels{"Animal Factor": "cat"}, []float64{
					1.0,
					3.0,
				}),
				data.NewField(`Values Floats`, data.Labels{"Animal Factor": "sloth"}, []float64{
					2.0,
					4.0,
				})),

			longFrame: data.NewFrame("wide_to_long_test",
				data.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
				}),
				data.NewField("Values Floats", nil, []float64{
					1.0,
					2.0,
					3.0,
					4.0,
				}),
				data.NewField("Animal Factor", nil, []string{
					"cat",
					"sloth",
					"cat",
					"sloth",
				})),
			Err: require.NoError,
		},

		{
			name: "one value, two factors",
			wideFrame: data.NewFrame("wide_to_long_test",
				data.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
				}),
				data.NewField(`Values Floats`,
					data.Labels{"Animal Factor": "cat", "Location": "Florida"}, []float64{
						1.0,
						3.0,
					}),
				data.NewField(`Values Floats`,
					data.Labels{"Animal Factor": "sloth", "Location": "Central & South America"}, []float64{
						2.0,
						4.0,
					})),
			Err: require.NoError,

			longFrame: data.NewFrame("wide_to_long_test",
				data.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
				}),
				data.NewField("Values Floats", nil, []float64{
					1.0,
					2.0,
					3.0,
					4.0,
				}),
				data.NewField("Animal Factor", nil, []string{
					"cat",
					"sloth",
					"cat",
					"sloth",
				}),
				data.NewField("Location", nil, []string{
					"Florida",
					"Central & South America",
					"Florida",
					"Central & South America",
				})),
		},
		{
			name: "two values, one factor",
			wideFrame: data.NewFrame("wide_to_long_test",
				data.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
				}),
				data.NewField(`Values Floats`, data.Labels{"Animal Factor": "cat"}, []float64{
					1.0,
					3.0,
				}),
				data.NewField(`Values Int64`, data.Labels{"Animal Factor": "cat"}, []int64{
					1,
					3,
				}),
				data.NewField(`Values Floats`, data.Labels{"Animal Factor": "sloth"}, []float64{
					2.0,
					4.0,
				}),
				data.NewField(`Values Int64`, data.Labels{"Animal Factor": "sloth"}, []int64{
					2,
					4,
				})),

			longFrame: data.NewFrame("wide_to_long_test",
				data.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
				}),
				data.NewField("Values Floats", nil, []float64{
					1.0,
					2.0,
					3.0,
					4.0,
				}),
				data.NewField("Values Int64", nil, []int64{
					1,
					2,
					3,
					4,
				}),
				data.NewField("Animal Factor", nil, []string{
					"cat",
					"sloth",
					"cat",
					"sloth",
				})),
			Err: require.NoError,
		},
		{
			name: "two values, two factor",
			wideFrame: data.NewFrame("wide_to_long_test",
				data.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
				}),
				data.NewField(`Values Floats`, data.Labels{"Animal Factor": "cat", "Location": "Florida"}, []float64{
					1.0,
					3.0,
				}),
				data.NewField(`Values Int64`, data.Labels{"Animal Factor": "cat", "Location": "Florida"}, []int64{
					1,
					3,
				}),
				data.NewField(`Values Floats`, data.Labels{"Animal Factor": "sloth", "Location": "Central & South America"}, []float64{
					2.0,
					4.0,
				}),
				data.NewField(`Values Int64`, data.Labels{"Animal Factor": "sloth", "Location": "Central & South America"}, []int64{
					2,
					4,
				})),

			longFrame: data.NewFrame("wide_to_long_test",
				data.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
				}),
				data.NewField("Values Floats", nil, []float64{
					1.0,
					2.0,
					3.0,
					4.0,
				}),
				data.NewField("Values Int64", nil, []int64{
					1,
					2,
					3,
					4,
				}),
				data.NewField("Animal Factor", nil, []string{
					"cat",
					"sloth",
					"cat",
					"sloth",
				}),
				data.NewField("Location", nil, []string{
					"Florida",
					"Central & South America",
					"Florida",
					"Central & South America",
				})),
			Err: require.NoError,
		},
		{
			name: "pointers: one value, one factor. Time becomes non-pointer since null time not supported",
			wideFrame: data.NewFrame("wide_to_long_test",
				data.NewField("Time", nil, []*time.Time{
					timePtr(time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC)),
					timePtr(time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC)),
				}),
				data.NewField(`Values Floats`, data.Labels{"Animal Factor": "cat"}, []*float64{
					float64Ptr(1.0),
					float64Ptr(3.0),
				}),
				data.NewField(`Values Floats`, data.Labels{"Animal Factor": "sloth"}, []*float64{
					float64Ptr(2.0),
					float64Ptr(4.0),
				})),
			Err: require.NoError,

			longFrame: data.NewFrame("wide_to_long_test",
				data.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
				}),
				data.NewField("Values Floats", nil, []*float64{
					float64Ptr(1.0),
					float64Ptr(2.0),
					float64Ptr(3.0),
					float64Ptr(4.0),
				}),
				data.NewField("Animal Factor", nil, []string{
					"cat",
					"sloth",
					"cat",
					"sloth",
				})),
		},
		{
			name: "sparse: one value, two factor",
			wideFrame: data.NewFrame("wide_to_long_test",
				data.NewField("Time", nil, []time.Time{
					time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 5, 0, 0, time.UTC),
					time.Date(2020, 1, 2, 3, 5, 30, 0, time.UTC),
				}),
				data.NewField(`Values Floats`, data.Labels{"Animal Factor": "cat", "Location": "Florida"}, []float64{
					1.0,
					3.0,
					0.0,
					0.0,
				}),
				data.NewField(`Values Floats`, data.Labels{"Animal Factor": "sloth", "Location": "Central & South America"}, []float64{
					2.0,
					4.0,
					0.0,
					6.0,
				}),
				data.NewField(`Values Floats`, data.Labels{"Animal Factor": "pangolin", "Location": ""}, []float64{
					0.0,
					0.0,
					55.0,
					0.0,
				})),

			longFrame: data.NewFrame("wide_to_long_test",
				data.NewField("Time", nil, []time.Time{
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
				data.NewField("Values Floats", nil, []float64{
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

				data.NewField("Animal Factor", nil, []string{
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
				data.NewField("Location", nil, []string{
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
			frame, err := data.WideToLong(tt.wideFrame)
			tt.Err(t, err)
			if diff := cmp.Diff(tt.longFrame, frame, data.FrameTestCompareOptions()...); diff != "" {
				t.Errorf("Result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFloatAt(t *testing.T) {
	mixedFrame := data.NewFrame("",
		data.NewField("", nil, []*int64{nil, int64Ptr(-5), int64Ptr(5)}),
		data.NewField("", nil, []*string{nil, stringPtr("-5"), stringPtr("5")}),
		data.NewField("", nil, []*bool{nil, boolPtr(true), boolPtr(false)}),
		data.NewField("", nil, []*time.Time{
			nil,
			timePtr(time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC)),
			timePtr(time.Date(2020, 1, 2, 3, 4, 30, 0, time.UTC)),
		}),
		data.NewField("", nil, []uint64{0, 12, math.MaxUint64}),
	)

	expectedFloatFrame := data.NewFrame("",
		data.NewField("", nil, []float64{math.NaN(), -5, 5}),
		data.NewField("", nil, []float64{math.NaN(), -5, 5}),
		data.NewField("", nil, []float64{0, 1, 0}),
		data.NewField("", nil, []float64{math.NaN(), 1577934240000, 1577934270000}),
		data.NewField("", nil, []float64{0, 12, 1.8446744073709552e+19}), // Note: loss of precision.
	)

	floatFrame := data.NewFrame("")
	floatFrame.Fields = make([]*data.Field, len(mixedFrame.Fields))
	for fieldIdx, field := range mixedFrame.Fields {
		floatFrame.Fields[fieldIdx] = data.NewFieldFromFieldType(data.FieldTypeFloat64, field.Len())
		for i := 0; i < field.Len(); i++ {
			fv, err := field.FloatAt(i)
			require.NoError(t, err)
			floatFrame.Set(fieldIdx, i, fv)
		}
	}

	if diff := cmp.Diff(expectedFloatFrame, floatFrame, data.FrameTestCompareOptions()...); diff != "" {
		t.Errorf("Result mismatch (-want +got):\n%s", diff)
	}
}

func TestToGrafanaSeries(t *testing.T) {
	tests := []struct {
		name        string
		frame       *data.Frame
		seriesSlice timeSeriesSlice
		Err         require.ErrorAssertionFunc
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seriesSlice, err := toGTimeSeries(tt.frame)
			tt.Err(t, err)
			require.Equal(t, tt.seriesSlice, seriesSlice)
		})
	}
}

type nullFloat struct {
	sql.NullFloat64
}

type timePoint [2]nullFloat // value,time because it was opposite day. Time is epoch in milliseconds.
type timeSeriesPoints []timePoint
type timeSeriesSlice []*timeSeries

type timeSeries struct {
	Name   string            `json:"name"`
	Points timeSeriesPoints  `json:"points"`
	Tags   map[string]string `json:"tags,omitempty"`
}

func toGTimeSeries(frame *data.Frame) (timeSeriesSlice, error) {
	tsSchema := frame.TimeSeriesSchema()
	if tsSchema.Type == data.TimeSeriesTypeNot {
		return nil, fmt.Errorf("input frame is not recognized as a time series")
	}
	// If Long, make wide
	if tsSchema.Type == data.TimeSeriesTypeWide {
		var err error
		frame, err = data.LongToWide(frame)
		if err != nil {
			return nil, err // TODO: Kyle needs to figure out error wrapping and get with times.
		}
		tsSchema = frame.TimeSeriesSchema()
	}
	//seriesCount := len(tsSchema.ValueIndices)
	//seriesSlice := make(timeSeriesSlice, 0, len(seriesCount))
	//for _,
	return nil, nil

}
