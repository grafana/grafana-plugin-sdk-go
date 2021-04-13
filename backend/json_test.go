package backend_test

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/require"
)

func testDataResponse() backend.DataResponse {
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
	return backend.DataResponse{
		Frames: frames,
	}
}

// TestResponseEncoder makes sure that the JSON produced from arrow and dataframes match
func TestResponseEncoder(t *testing.T) {
	dr := testDataResponse()

	b, err := json.Marshal(dr)
	require.NoError(t, err)

	str := string(b)
	require.Equal(t, `{"frames":[{"schema":{"name":"simple","fields":[{"name":"time","type":"time","typeInfo":{"frame":"time.Time"}},{"name":"valid","type":"boolean","typeInfo":{"frame":"boolean"}}]},"data":{"values":[[1577934240000,1577934300000],[true,false]]}},{"schema":{"name":"other","fields":[{"name":"value","type":"number","typeInfo":{"frame":"float64"}}]},"data":{"values":[[1]]}}]}`, str)

	b2, err := json.Marshal(&dr)
	require.NoError(t, err)
	require.Equal(t, str, string(b2), "same result from pointer or object")

	// Now the same thing in query data
	qdr := backend.NewQueryDataResponse()
	qdr.Responses["A"] = dr

	b, err = json.Marshal(qdr)
	require.NoError(t, err)

	str = string(b)
	require.Equal(t, `{"results":{"A":{"frames":[{"schema":{"name":"simple","fields":[{"name":"time","type":"time","typeInfo":{"frame":"time.Time"}},{"name":"valid","type":"boolean","typeInfo":{"frame":"boolean"}}]},"data":{"values":[[1577934240000,1577934300000],[true,false]]}},{"schema":{"name":"other","fields":[{"name":"value","type":"number","typeInfo":{"frame":"float64"}}]},"data":{"values":[[1]]}}]}}}`, str)

	// Read the parsed result and make sure it is the same
	copy := &backend.QueryDataResponse{}
	err = json.Unmarshal(b, copy)
	require.NoError(t, err)
	require.Equal(t, len(qdr.Responses), len(copy.Responses))

	// Check the final result
	for k, val := range qdr.Responses {
		other := copy.Responses[k]
		require.Equal(t, len(val.Frames), len(other.Frames))

		for idx := range val.Frames {
			a := val.Frames[idx]
			b := other.Frames[idx]

			if diff := cmp.Diff(a, b, data.FrameTestCompareOptions()...); diff != "" {
				t.Errorf("Result mismatch (-want +got):\n%s", diff)
			}
		}
	}
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
	qdr := backend.NewQueryDataResponse()
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
