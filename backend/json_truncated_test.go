package backend_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/data/utils/jsoniter"
)

func TestQueryDataResponseJSONCutAtEveryByte(t *testing.T) {
	ts := []time.Time{time.Unix(1, 0), time.Unix(2, 0)}
	qdr := backend.NewQueryDataResponse()
	qdr.Responses["A"] = backend.DataResponse{Frames: data.Frames{
		data.NewFrame("a", data.NewField("time", nil, ts), data.NewField("value", data.Labels{"host": "a"}, []float64{1, 2})),
		data.NewFrame("b", data.NewField("time", nil, ts), data.NewField("value", data.Labels{"host": "b"}, []*float64{nil, nil})),
	}}
	qdr.Responses["B"] = backend.DataResponse{Error: errors.New("boom"), Status: backend.StatusBadRequest}
	b, err := qdr.MarshalJSON()
	require.NoError(t, err)

	for i := range len(b) {
		t.Run("UnmarshalJSON", func(t *testing.T) {
			out := &backend.QueryDataResponse{}
			var err error
			require.NotPanics(t, func() { err = out.UnmarshalJSON(b[:i]) }, "cut at byte %d", i)
			require.Error(t, err, "cut at byte %d", i)
		})
		t.Run("ReadVal", func(t *testing.T) {
			iter, err := jsoniter.ParseBytes(jsoniter.ConfigCompatibleWithStandardLibrary, b[:i])
			require.NoError(t, err)
			out := &backend.QueryDataResponse{}
			require.NotPanics(t, func() { err = iter.ReadVal(out) }, "cut at byte %d", i)
			require.Error(t, err, "cut at byte %d", i)
		})
	}
}
