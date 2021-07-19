package data_test

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/require"
)

func TestWriteCSV(t *testing.T) {
	frame := data.NewFrame("wide_to_long_test",
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
		}))

	buf := bytes.NewBufferString("")
	err := data.FrameToCSV(frame, buf, data.FrameToCSVOptions{
		ShowNames: true,
		ShowTypes: true,
	})
	require.NoError(t, err)

	fmt.Printf("%s", buf.String())
	require.Equal(t,
		`Time,Values Floats,Values Floats
time.Time,float64,float64
2020-01-02 03:04:00 +0000 UTC,1,2
2020-01-02 03:04:30 +0000 UTC,3,4
`, buf.String())
}
