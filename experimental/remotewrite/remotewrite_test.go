package remotewrite

import (
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/require"
)

func TestTsFromFrames(t *testing.T) {
	t1 := time.Now()
	t2 := time.Now().Add(time.Second)
	frame := data.NewFrame("test",
		data.NewField("time", nil, []time.Time{t1, t2}),
		data.NewField("value", nil, []float64{1.0, 2.0}),
	)
	ts := tsFromFrames(frame)
	require.Len(t, ts, 1)
	require.Len(t, ts[0].Samples, 2)
	require.Equal(t, toSampleTime(t1), ts[0].Samples[0].Timestamp)
	require.Equal(t, toSampleTime(t2), ts[0].Samples[1].Timestamp)
}

func TestSerialize(t *testing.T) {
	frame := data.NewFrame("test",
		data.NewField("time", nil, []time.Time{time.Now(), time.Now().Add(time.Second)}),
		data.NewField("value", nil, []float64{1.0, 2.0}),
	)
	_, err := Serialize(frame)
	require.NoError(t, err)
}
