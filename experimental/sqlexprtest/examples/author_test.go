package examples_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/sqlexprtest"
	"github.com/stretchr/testify/require"
)

// This reference handler decodes an upstream fixture through QueryData. Replace
// it with your plugin's handler and mock the service below its decoder.
type fixtureDatasource struct{ payload []byte }

type queryModel struct {
	Mode       string `json:"mode"`
	Expression string `json:"expression"`
	Empty      bool   `json:"empty"`
}

func (ds fixtureDatasource) QueryData(_ context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	response := backend.NewQueryDataResponse()
	for _, q := range req.Queries {
		var model queryModel
		if err := json.Unmarshal(q.JSON, &model); err != nil {
			response.Responses[q.RefID] = backend.DataResponse{Error: err}
			continue
		}
		if model.Mode == "prepared" && model.Expression == "" {
			response.Responses[q.RefID] = backend.DataResponse{Error: fmt.Errorf("backend expression is required")}
			continue
		}
		var samples []struct {
			Host  string `json:"host"`
			Bytes int64  `json:"bytes"`
		}
		if err := json.Unmarshal(ds.payload, &samples); err != nil {
			response.Responses[q.RefID] = backend.DataResponse{Error: err}
			continue
		}
		var frame *data.Frame
		if model.Mode == "metrics" {
			times := []time.Time{q.TimeRange.From}
			if model.Empty {
				times = nil
			}
			frame = data.NewFrame("", data.NewField("time", nil, times)).SetMeta(&data.FrameMeta{Type: data.FrameTypeTimeSeriesWide})
			for _, sample := range samples {
				values := []int64{sample.Bytes}
				if model.Empty {
					values = nil
				}
				frame.Fields = append(frame.Fields, data.NewField("bytes", data.Labels{"host": sample.Host}, values))
			}
		} else {
			hosts, values := []string{}, []int64{}
			if !model.Empty {
				for _, sample := range samples {
					hosts, values = append(hosts, sample.Host), append(values, sample.Bytes)
				}
			}
			frame = data.NewFrame("", data.NewField("host", nil, hosts), data.NewField("bytes", nil, values)).SetMeta(&data.FrameMeta{Type: data.FrameTypeTable})
		}
		response.Responses[q.RefID] = backend.DataResponse{Frames: data.Frames{frame}}
	}
	return response, nil
}

func fixtureRequest(t *testing.T, model queryModel) *backend.QueryDataRequest {
	t.Helper()
	b, err := json.Marshal(model)
	require.NoError(t, err)
	from := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	return &backend.QueryDataRequest{Queries: []backend.DataQuery{{RefID: "A", JSON: b, TimeRange: backend.TimeRange{From: from, To: from.Add(time.Minute)}}}}
}

func tableContract(rows int) sqlexprtest.Contract {
	return sqlexprtest.Contract{Mode: sqlexprtest.ModeTable, Rows: &rows, Columns: []sqlexprtest.Column{
		{Name: "host", Type: data.FieldTypeString, SQLType: "TEXT"},
		{Name: "bytes", Type: data.FieldTypeInt64, SQLType: "BIGINT"},
	}}
}

func TestTableDatasourceSuite(t *testing.T) {
	ds := fixtureDatasource{payload: []byte(`[{"host":"a","bytes":9007199254740993},{"host":"b","bytes":42}]`)}
	suite := sqlexprtest.Suite{Name: "table datasource", Modes: []sqlexprtest.QueryMode{{Name: "table"}}}
	for _, empty := range []bool{false, true} {
		rows := 2
		if empty {
			rows = 0
		}
		suite.Scenarios = append(suite.Scenarios, sqlexprtest.Scenario{Name: fmt.Sprintf("empty=%t", empty), Mode: "table",
			Input: sqlexprtest.QueryDataInput(ds, fixtureRequest(t, queryModel{Mode: "table", Empty: empty})),
			Want:  map[string]sqlexprtest.Contract{"A": tableContract(rows)},
		})
	}
	r := sqlexprtest.AssertSuite(t, t.Context(), suite, sqlexprtest.MinGrafana("13.2.4"))
	require.NotNil(t, r)
	require.True(t, r.Passed())
	encoded, err := json.MarshalIndent(r, "", "  ")
	require.NoError(t, err)
	t.Log(string(encoded))
}

func TestMetricsDatasourceSuite(t *testing.T) {
	ds := fixtureDatasource{payload: []byte(`[{"host":"a","bytes":10},{"host":"b","bytes":20}]`)}
	suite := sqlexprtest.Suite{Name: "labeled metrics datasource", Modes: []sqlexprtest.QueryMode{{Name: "metrics"}}}
	for _, empty := range []bool{false, true} {
		rows := 2
		if empty {
			rows = 0
		}
		suite.Scenarios = append(suite.Scenarios, sqlexprtest.Scenario{Name: fmt.Sprintf("empty=%t", empty), Mode: "metrics",
			Input: sqlexprtest.QueryDataInput(ds, fixtureRequest(t, queryModel{Mode: "metrics", Empty: empty})),
			Want: map[string]sqlexprtest.Contract{"A": {Mode: sqlexprtest.ModeFullLong, Rows: &rows, MaxCells: 100, Columns: []sqlexprtest.Column{
				{Name: "time", Type: data.FieldTypeTime, SQLType: "TIMESTAMP"},
				{Name: "__value__", Type: data.FieldTypeNullableFloat64, SQLType: "DOUBLE"},
				{Name: "__metric_name__", Type: data.FieldTypeString, SQLType: "TEXT"},
				{Name: "host", Type: data.FieldTypeNullableString, SQLType: "TEXT"},
			}}},
		})
	}
	r := sqlexprtest.AssertSuite(t, t.Context(), suite)
	require.NotNil(t, r)
	require.True(t, r.Passed())
}

func TestPreparedQuerySuite(t *testing.T) {
	ds := fixtureDatasource{payload: []byte(`[{"host":"a","bytes":10}]`)}
	suite := sqlexprtest.Suite{Name: "query preparation", Modes: []sqlexprtest.QueryMode{
		{Name: "prepared"}, {Name: "browser-only", UnsupportedReason: "This reference mode has no backend query execution."},
	}}
	for _, withHeader := range []bool{false, true} {
		req := fixtureRequest(t, queryModel{Mode: "prepared", Expression: "bytes"})
		if withHeader {
			req.SetHTTPHeader(sqlexprtest.HeaderFromExpression, "true")
		}
		suite.Scenarios = append(suite.Scenarios, sqlexprtest.Scenario{Name: fmt.Sprintf("prepared with header=%t", withHeader), Mode: "prepared",
			Input: sqlexprtest.QueryDataInput(ds, req), Want: map[string]sqlexprtest.Contract{"A": tableContract(1)},
		})
	}
	suite.Scenarios = append(suite.Scenarios, sqlexprtest.Scenario{Name: "unprepared model is rejected", Mode: "prepared", ExpectedFailure: true,
		Input:      sqlexprtest.QueryDataInput(ds, fixtureRequest(t, queryModel{Mode: "prepared"})),
		Want:       map[string]sqlexprtest.Contract{"A": {Mode: sqlexprtest.ModeNone}},
		Allowances: []sqlexprtest.Allowance{{Code: sqlexprtest.CodeQueryError, RefID: "A", Reason: "Negative fixture verifies the backend rejects an unprepared query."}},
	})
	r := sqlexprtest.AssertSuite(t, t.Context(), suite)
	require.NotNil(t, r)
	require.False(t, r.HasFailures())
	require.False(t, r.Passed(), "an explicitly unsupported mode is not tested support")
}
