package sqlexprtest_test

import (
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/sqlexprtest"
)

// A time series frame with a status field. SQL Expressions keep the numeric
// values and drop the status field.
func ExampleCheck() {
	t0 := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	frame := data.NewFrame("",
		data.NewField("time", nil, []time.Time{t0}),
		data.NewField("status", nil, []string{"degraded"}),
		data.NewField("latency", data.Labels{"host": "a"}, []float64{120}),
	)
	frame.RefID = "A"
	frame.Meta = &data.FrameMeta{Type: data.FrameTypeTimeSeriesWide}

	r := sqlexprtest.Check(data.Frames{frame})
	fmt.Println(r)
	// Output:
	// query "A": mode full-long, input type timeseries-wide, rows 1
	//   columns: time TIMESTAMP, __value__ DOUBLE, __metric_name__ TEXT, host TEXT
	//   warning [dropped-field, grafana] frame 0 field "status": Grafana keeps only numeric value fields when it converts timeseries-wide for SQL, so field 1 (string) is dropped without an error. Return a table or long frame to keep it.
}

func ExampleCheckResponse() {
	response := &backend.QueryDataResponse{Responses: backend.Responses{}}
	reports := sqlexprtest.CheckResponse(response, []string{"A"})
	r := reports["A"]
	fmt.Println(r.RefID, r.Mode, r.Rows, r.Has(sqlexprtest.CodeMissingRefID))
	// Output: A table 0 true
}
