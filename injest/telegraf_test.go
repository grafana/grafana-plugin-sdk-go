package injest

import (
	"testing"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/stretchr/testify/require"

	"github.com/influxdata/telegraf/metric"
	"github.com/influxdata/telegraf/plugins/parsers/influx"
)

func Metric(v telegraf.Metric, err error) telegraf.Metric {
	if err != nil {
		panic(err)
	}
	return v
}

var DefaultTime = func() time.Time {
	return time.Unix(42, 0)
}

func TestInjest(t *testing.T) {
	// RunServer()

	handler := influx.NewMetricHandler()
	parser := influx.NewParser(handler)
	parser.SetTimeFunc(DefaultTime)

	metrics, err := parser.Parse([]byte("cpu value=42 0"))
	require.NoError(t, err)

	require.Equal(t, 1, len(metrics))

	expected := Metric(
		metric.New(
			"cpu",
			map[string]string{},
			map[string]interface{}{
				"value": 42.0,
			},
			time.Unix(0, 0),
		),
	)
	i := 0
	require.Equal(t, expected.Name(), metrics[i].Name())
	require.Equal(t, expected.Tags(), metrics[i].Tags())
	require.Equal(t, expected.Fields(), metrics[i].Fields())
	require.Equal(t, expected.Time(), metrics[i].Time())
}
