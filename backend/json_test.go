package backend

import (
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/require"
)

// TestResponseEncoder makes sure that the JSON produced from arrow and dataframes match
func TestResponseEncoder(t *testing.T) {
	frames := data.Frames{
		data.NewFrame("simple",
			data.NewField("time", nil, []time.Time{
				time.Date(2020, 1, 2, 3, 4, 0, 0, time.UTC),
				time.Date(2020, 1, 2, 3, 5, 0, 0, time.UTC),
			}),
			data.NewField("valid", nil, []bool{true, false}),
		),
		data.NewFrame("other",
			data.NewField("value", nil, []float64{1.0}),
		),
	}

	dr := DataResponse{
		Frames: frames,
	}

	b, err := jsoniter.Marshal(dr)
	require.NoError(t, err)

	str := string(b)
	require.Equal(t, `{"frames":[{"schema":{"name":"simple","fields":[{"name":"time","type":"time"},{"name":"valid","type":"bool"}]},"data":{"values":[[1577934240000,1577934300000],[true,false]]}}{"schema":{"name":"other","fields":[{"name":"value","type":"number"}]},"data":{"values":[[1]]}}]}`, str)

	// Now the same thing in query data
	qdr := NewQueryDataResponse()
	qdr.Responses["A"] = dr

	b, err = jsoniter.Marshal(qdr)
	require.NoError(t, err)

	str = string(b)
	require.Equal(t, `{"responses":{"A":{"frames":[{"schema":{"name":"simple","fields":[{"name":"time","type":"time"},{"name":"valid","type":"bool"}]},"data":{"values":[[1577934240000,1577934300000],[true,false]]}}{"schema":{"name":"other","fields":[{"name":"value","type":"number"}]},"data":{"values":[[1]]}}]}}}`, str)
}
