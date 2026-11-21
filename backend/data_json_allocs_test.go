package backend

import (
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func TestQueryDataResponseMarshalJSONAllocs(t *testing.T) {
	const rows = 1000
	ts := make([]time.Time, rows)
	vals := make([]float64, rows)
	for i := range ts {
		ts[i] = time.UnixMilli(int64(i) * 1000)
		vals[i] = float64(i) / 3
	}
	frame := data.NewFrame("allocs", data.NewField("time", nil, ts), data.NewField("value", nil, vals))
	rsp := QueryDataResponse{Responses: Responses{"A": DataResponse{Frames: data.Frames{frame}}}}

	got := testing.AllocsPerRun(20, func() {
		if _, err := rsp.MarshalJSON(); err != nil {
			t.Fatal(err)
		}
	})
	if got > 12 {
		t.Errorf("QueryDataResponse.MarshalJSON allocated %v times for a 2 field, %d row frame, want at most 12", got, rows)
	}
}
