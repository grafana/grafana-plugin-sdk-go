package tabular_test

import (
	"errors"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/experimental"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/sdata/tabular"
	"github.com/stretchr/testify/require"
)

func TestNewHistogramFrame(t *testing.T) {
	tests := []struct {
		name           string
		minValue       float64
		maxValue       float64
		bucketSize     float64
		wantBucketSize int
		wantErr        error
	}{
		{name: "should throw error when bucket size is invalid", minValue: 0, maxValue: 100, bucketSize: 0, wantErr: errors.New("invalid bucket size")},
		{name: "should throw error when min/max value is invalid", minValue: 10, maxValue: 10, bucketSize: 5, wantErr: errors.New("max value should be greater than min value")},
		{name: "should calculate the buckets length correctly", minValue: 0, maxValue: 100, bucketSize: 10, wantBucketSize: 10},
		{name: "should respect non zero min value", minValue: 12, maxValue: 98, bucketSize: 10, wantBucketSize: 9},
		{name: "should respect decimal value buckets", minValue: 7.5, maxValue: 10.6, bucketSize: 0.5, wantBucketSize: 7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tabular.NewHistogramFrame(tt.minValue, tt.maxValue, tt.bucketSize)
			if tt.wantErr != nil {
				require.Equal(t, tt.wantErr, err)
				return
			}
			require.Nil(t, err)
			require.NotNil(t, got)
			require.Equal(t, 2, len(got.Fields))
			require.Equal(t, "BucketMin", got.Fields[0].Name)
			require.Equal(t, "BucketMax", got.Fields[1].Name)
			require.Equal(t, tt.wantBucketSize, got.Rows())
			require.Equal(t, tt.minValue, got.Fields[0].At(0))
			require.Equal(t, tt.maxValue, got.Fields[1].At(got.Rows()-1))
		})
	}
}

func TestNewHistogramFrameWithValues(t *testing.T) {
	tests := []struct {
		name           string
		minValue       float64
		maxValue       float64
		bucketSize     float64
		metricName     string
		values         []float64
		wantBucketSize int
		wantErr        error
	}{
		{minValue: 0, maxValue: 100, bucketSize: 10, metricName: "foo", values: []float64{}, wantErr: errors.New("number of values not matching the buckets length 10")},
		{minValue: 0, maxValue: 100, bucketSize: 10, metricName: "foo", wantBucketSize: 10, values: []float64{1.2, 2.2, 3.2, 4.2, 5.2, 6.2, 7.2, 8.2, 9.2, 10.2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tabular.NewHistogramFrameWithValues(tt.minValue, tt.maxValue, tt.bucketSize, tt.metricName, tt.values)
			if tt.wantErr != nil {
				require.Equal(t, tt.wantErr, err)
				return
			}
			require.Nil(t, err)
			require.NotNil(t, got)
			require.Equal(t, 3, len(got.Fields))
			require.Equal(t, "BucketMin", got.Fields[0].Name)
			require.Equal(t, "BucketMax", got.Fields[1].Name)
			require.Equal(t, tt.metricName, got.Fields[2].Name)
			require.Equal(t, tt.wantBucketSize, got.Rows())
			require.Equal(t, tt.minValue, got.Fields[0].At(0))
			require.Equal(t, tt.maxValue, got.Fields[1].At(got.Rows()-1))
			require.Equal(t, tt.values[3], got.Fields[2].At(3))
		})
	}
}

func TestHistogram_AddValue(t *testing.T) {
	frame, err := tabular.NewHistogramFrame(0, 50, 10)
	require.Nil(t, err)
	err = frame.AddValue("foo", map[string]string{"env": "dev"}, []float64{1, 2, 3, 4, 5})
	require.Nil(t, err)
	err = frame.AddValue("foo", map[string]string{"env": "prod"}, []float64{5, 4, 3, 2, 1})
	require.Nil(t, err)
	require.Nil(t, experimental.CheckGoldenFrame("testdata/histogram-multiple.golden.txt", frame.Frame, false))
}
