package backend

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/require"
)

func testDataResponse() DataResponse {
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
	return DataResponse{
		Frames: frames,
	}
}

// TestResponseEncoder makes sure that the JSON produced from arrow and dataframes match
func TestResponseEncoder(t *testing.T) {
	dr := testDataResponse()

	b, err := json.Marshal(dr)
	require.NoError(t, err)

	str := string(b)
	require.Equal(t, `{"frames":[{"schema":{"name":"simple","fields":[{"name":"time","type":"time","typeInfo":{"frame":"time.Time"}},{"name":"valid","type":"bool","typeInfo":{"frame":"bool"}}]},"data":{"values":[[1577934240000,1577934300000],[true,false]]}},{"schema":{"name":"other","fields":[{"name":"value","type":"number","typeInfo":{"frame":"float64"}}]},"data":{"values":[[1]]}}]}`, str)

	b2, err := json.Marshal(&dr)
	require.NoError(t, err)
	require.Equal(t, str, string(b2), "same result from pointer or object")

	// Now the same thing in query data
	qdr := NewQueryDataResponse()
	qdr.Responses["A"] = dr

	b, err = json.Marshal(qdr)
	require.NoError(t, err)

	str = string(b)
	require.Equal(t, `{"results":[{"refId":"A","frames":[{"schema":{"name":"simple","fields":[{"name":"time","type":"time","typeInfo":{"frame":"time.Time"}},{"name":"valid","type":"bool","typeInfo":{"frame":"bool"}}]},"data":{"values":[[1577934240000,1577934300000],[true,false]]}},{"schema":{"name":"other","fields":[{"name":"value","type":"number","typeInfo":{"frame":"float64"}}]},"data":{"values":[[1]]}}]}]}`, str)
}

func TestDataResponseMarshalJSONConcurrent(t *testing.T) {
	dr := testDataResponse()
	initialJSON, err := json.Marshal(dr)
	require.NoError(t, err)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				jsonData, err := json.Marshal(dr)
				require.NoError(t, err)
				require.JSONEq(t, string(initialJSON), string(jsonData))
			}
		}()
	}
	wg.Wait()
}

func TestQueryDataResponseMarshalJSONConcurrent(t *testing.T) {
	qdr := NewQueryDataResponse()
	qdr.Responses["A"] = testDataResponse()
	initialJSON, err := json.Marshal(qdr)
	require.NoError(t, err)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				jsonData, err := json.Marshal(qdr)
				require.NoError(t, err)
				require.JSONEq(t, string(initialJSON), string(jsonData))
			}
		}()
	}
	wg.Wait()
}
