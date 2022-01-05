package sdata_test

import (
	"strings"
	"testing"
	"time"

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

func TestMultiFrameSeriesValidiate_WithFrames(t *testing.T) {
	tests := []struct {
		name          string
		mfs           *sdata.MultiFrameSeries
		empty         bool
		errCount      int
		errorsContain []string
	}{
		{
			name:     "nil or empty set is valid and empty",
			empty:    true,
			errCount: 0,
		},
		{
			name: "frame with no fields is valid, and does not mean set is empty",
			mfs: &sdata.MultiFrameSeries{
				emptyFrameWithManySeriesMD(),
			},
			errCount: 0,
		},
		{
			name: "frame with only value field is not valid, missing time field",
			mfs: &sdata.MultiFrameSeries{
				addFields(emptyFrameWithManySeriesMD(),
					data.NewField("", nil, []float64{})),
			},
			errCount:      1,
			errorsContain: []string{"missing a Time field"},
		},
		{
			name: "frame with only a time field and no value is not valid",
			mfs: &sdata.MultiFrameSeries{
				addFields(emptyFrameWithManySeriesMD(),
					data.NewField("", nil, []float64{})),
			},
			errCount:      1,
			errorsContain: []string{"missing a Time field"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			empty, errors := tt.mfs.Validate()
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
			require.Equal(t, tt.empty, empty, "expected valid to be %v", tt.empty)
		})
	}
}
