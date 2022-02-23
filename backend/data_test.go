package backend

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/require"
)

func TestQueryDataResponse(t *testing.T) {
	dr := testDataResponse(t)
	qdr := NewQueryDataResponse()
	qdr.Responses["A"] = dr

	require.Nil(t, qdr.ResponseProxy())

	b, err := json.Marshal(qdr)
	require.NoError(t, err)

	str := string(b)
	require.Equal(t, `{"results":{"A":{"frames":[{"schema":{"name":"simple","fields":[{"name":"time","type":"time","typeInfo":{"frame":"time.Time"}},{"name":"valid","type":"boolean","typeInfo":{"frame":"bool"}}]},"data":{"values":[[1577934240000,1577934300000],[true,false]]}},{"schema":{"name":"other","fields":[{"name":"value","type":"number","typeInfo":{"frame":"float64"}}]},"data":{"values":[[1]]}}]}}}`, str)

	var qdrNew QueryDataResponse
	err = json.Unmarshal(b, &qdrNew)
	require.NoError(t, err)

	require.NotNil(t, qdrNew.Responses)
	require.NotNil(t, qdrNew.ResponseProxy())
	jsonProxy, ok := qdrNew.proxy.(*jsonResponseProxy)
	require.True(t, ok)
	require.NotNil(t, jsonProxy)
	require.Len(t, jsonProxy.raw.data, len(b))
	responses, err := qdrNew.ResponseProxy().Responses()
	require.NoError(t, err)
	require.NotNil(t, responses)
}

func testDataResponse(t *testing.T) DataResponse {
	t.Helper()

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
