package backend_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	jsoniter "github.com/json-iterator/go"
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
		Status: http.StatusAccepted,
	}
}

// TestResponseEncoder makes sure that the JSON produced from arrow and dataframes match
func TestResponseEncoder(t *testing.T) {
	// invalid status will not be marshalled
	dr := testDataResponse()
	dr.Status = 999

	b, err := json.Marshal(dr)
	require.NoError(t, err)

	str := string(b)
	require.Equal(t, `{"frames":[{"schema":{"name":"simple","fields":[{"name":"time","type":"time","typeInfo":{"frame":"time.Time"}},{"name":"valid","type":"boolean","typeInfo":{"frame":"bool"}}]},"data":{"values":[[1577934240000,1577934300000],[true,false]]}},{"schema":{"name":"other","fields":[{"name":"value","type":"number","typeInfo":{"frame":"float64"}}]},"data":{"values":[[1]]}}]}`, str)

	// valid status will be marshalled
	dr = testDataResponse()

	b, err = json.Marshal(dr)
	require.NoError(t, err)

	str = string(b)
	require.Equal(t, `{"status":202,"frames":[{"schema":{"name":"simple","fields":[{"name":"time","type":"time","typeInfo":{"frame":"time.Time"}},{"name":"valid","type":"boolean","typeInfo":{"frame":"bool"}}]},"data":{"values":[[1577934240000,1577934300000],[true,false]]}},{"schema":{"name":"other","fields":[{"name":"value","type":"number","typeInfo":{"frame":"float64"}}]},"data":{"values":[[1]]}}]}`, str)

	b2, err := json.Marshal(&dr)
	require.NoError(t, err)
	require.Equal(t, str, string(b2), "same result from pointer or object")

	// Now the same thing in query data
	qdr := backend.NewQueryDataResponse()
	qdr.Responses["A"] = dr

	b, err = json.Marshal(qdr)
	require.NoError(t, err)

	str = string(b)
	require.Equal(t, `{"results":{"A":{"status":202,"frames":[{"schema":{"name":"simple","fields":[{"name":"time","type":"time","typeInfo":{"frame":"time.Time"}},{"name":"valid","type":"boolean","typeInfo":{"frame":"bool"}}]},"data":{"values":[[1577934240000,1577934300000],[true,false]]}},{"schema":{"name":"other","fields":[{"name":"value","type":"number","typeInfo":{"frame":"float64"}}]},"data":{"values":[[1]]}}]}}}`, str)

	// Read the parsed result and make sure it is the same
	respCopy := &backend.QueryDataResponse{}
	err = json.Unmarshal(b, respCopy)
	require.NoError(t, err)
	require.Equal(t, len(qdr.Responses), len(respCopy.Responses))
	require.Equal(t, backend.Status(202), respCopy.Responses["A"].Status)

	// Check the final result
	for k, val := range qdr.Responses {
		other := respCopy.Responses[k]
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
	for range 2 {
		wg.Add(1)
		go func(dr backend.DataResponse) {
			defer wg.Done()
			for range 100 {
				jsonData, err := json.Marshal(dr)
				require.NoError(t, err)
				require.JSONEq(t, string(initialJSON), string(jsonData))
			}
		}(dr)
	}
	wg.Wait()
}

func TestQueryDataResponseMarshalJSONConcurrent(t *testing.T) {
	qdr := backend.NewQueryDataResponse()
	qdr.Responses["A"] = testDataResponse()
	initialJSON, err := json.Marshal(qdr)
	require.NoError(t, err)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func(qdr *backend.QueryDataResponse) {
			defer wg.Done()
			for range 100 {
				jsonData, err := json.Marshal(qdr)
				require.NoError(t, err)
				require.JSONEq(t, string(initialJSON), string(jsonData))
			}
		}(qdr)
	}
	wg.Wait()
}

func TestQueryDataResponseOrdering(t *testing.T) {
	qdr := backend.NewQueryDataResponse()
	qdr.Responses["C"] = testDataResponse()
	qdr.Responses["A"] = testDataResponse()
	qdr.Responses["B"] = testDataResponse()
	b, err := json.Marshal(qdr)
	require.NoError(t, err)

	expectedDataResponse := `{"status":202,"frames":[{"schema":{"name":"simple","fields":[{"name":"time","type":"time","typeInfo":{"frame":"time.Time"}},{"name":"valid","type":"boolean","typeInfo":{"frame":"bool"}}]},"data":{"values":[[1577934240000,1577934300000],[true,false]]}},{"schema":{"name":"other","fields":[{"name":"value","type":"number","typeInfo":{"frame":"float64"}}]},"data":{"values":[[1]]}}]}`
	expected := fmt.Sprintf(`{"results":{"A":%s,"B":%s,"C":%s}}`, expectedDataResponse, expectedDataResponse, expectedDataResponse)

	require.JSONEq(t, expected, string(b))
}

func testQueryDataResponseRoundtrip(t *testing.T, qdr *backend.QueryDataResponse) {
	t.Helper()
	// to json
	b, err := json.Marshal(qdr)
	require.NoError(t, err)

	// from json
	qdr2 := &backend.QueryDataResponse{}
	err = json.Unmarshal(b, qdr2)
	require.NoError(t, err)

	// to json again, to compare
	b2, err := json.Marshal(qdr2)
	require.NoError(t, err)
	require.Equal(t, b2, b)
}

func TestQueryDataWithoutRefID(t *testing.T) {
	qdr := backend.NewQueryDataResponse()
	qdr.Responses[""] = testDataResponse()

	testQueryDataResponseRoundtrip(t, qdr)
}

func TestQueryDataWithtRefID(t *testing.T) {
	qdr := backend.NewQueryDataResponse()
	qdr.Responses["A"] = testDataResponse()

	testQueryDataResponseRoundtrip(t, qdr)
}

func TestQueryDataWithoutResponses(t *testing.T) {
	qdr := backend.NewQueryDataResponse()
	testQueryDataResponseRoundtrip(t, qdr)
}

// BenchmarkQueryDataResponseEncode encodes responses the way grafana-server writes /api/ds/query responses.
func BenchmarkQueryDataResponseEncode(b *testing.B) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	const rows = 1_000_000
	times := make([]time.Time, rows)
	values := make([]float64, rows)
	for i := range rows {
		times[i] = start.Add(time.Duration(i) * time.Second)
		values[i] = 20 + float64(i%977)*0.0137
	}
	series := make(data.Frames, 0, 20_000)
	for i := range 20_000 {
		labels := data.Labels{
			"__name__": "node_cpu_seconds_total", "cpu": fmt.Sprint(i % 64), "instance": fmt.Sprintf("10.0.%d.%d:9100", i/250, i%250),
			"job": "node", "mode": "idle", "pod": fmt.Sprintf("node-exporter-%05d", i),
		}
		series = append(series, data.NewFrame("", data.NewField("Time", nil, times[:1]), data.NewField("Value", labels, values[:1])))
	}

	benchmarks := []struct {
		name   string
		frames data.Frames
	}{
		{"time_and_float64_1M_rows", data.Frames{data.NewFrame("", data.NewField("time", nil, times), data.NewField("value", nil, values))}},
		{"single_row_series_20k", series},
	}
	for _, bm := range benchmarks {
		qdr := backend.NewQueryDataResponse()
		qdr.Responses["A"] = backend.DataResponse{Frames: bm.frames}
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if err := jsoniter.ConfigCompatibleWithStandardLibrary.NewEncoder(io.Discard).Encode(qdr); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
