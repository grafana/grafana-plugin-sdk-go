package grafanatest

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/require"
)

// TestQueryLive is opt-in. It tests TestData through an actual Grafana HTTP API;
// ordinary package tests never need a server or provider credentials.
func TestQueryLive(t *testing.T) {
	endpoint := os.Getenv("SQLEXPRTEST_GRAFANA_URL")
	if endpoint == "" {
		t.Skip("set SQLEXPRTEST_GRAFANA_URL and SQLEXPRTEST_GRAFANA_VERSION for live SQL contracts")
	}
	version := os.Getenv("SQLEXPRTEST_GRAFANA_VERSION")
	require.NotEmpty(t, version, "pin the expected Grafana version for a live test")
	uid := os.Getenv("SQLEXPRTEST_DATASOURCE_UID")
	if uid == "" {
		uid = "sqlexpr-testdata"
	}
	headers := make(http.Header)
	if token := os.Getenv("SQLEXPRTEST_GRAFANA_TOKEN"); token != "" {
		headers.Set("Authorization", "Bearer "+token)
	}
	client := Client{URL: endpoint, Headers: headers, ExpectedVersion: version, ExpectedCommit: os.Getenv("SQLEXPRTEST_GRAFANA_COMMIT")}
	source := func(ref, csv string) json.RawMessage {
		b, err := json.Marshal(map[string]any{"refId": ref, "datasource": map[string]string{"uid": uid, "type": "grafana-testdata-datasource"}, "scenarioId": "csv_content", "csvContent": csv, "intervalMs": 1000, "maxDataPoints": 100})
		require.NoError(t, err)
		return b
	}
	a := source("A", "host,n\na,9007199254740993\nb,null\nc,2")
	r := source("R", "host,zone\na,west\nc,east")
	jsonFrame := data.NewFrame("", data.NewField("doc", nil, []*json.RawMessage{new(json.RawMessage(`null`)), nil}))
	arrow, err := jsonFrame.MarshalArrow()
	require.NoError(t, err)
	j, err := json.Marshal(map[string]any{"refId": "J", "datasource": map[string]string{"uid": uid, "type": "grafana-testdata-datasource"}, "scenarioId": "arrow", "stringInput": base64.StdEncoding.EncodeToString(arrow)})
	require.NoError(t, err)
	for _, tc := range []struct {
		name, sql string
		extra     json.RawMessage
		want      Expectation
	}{
		{name: "projection preserves integer and null", sql: "SELECT host, n FROM A ORDER BY host", want: Expectation{Frames: data.Frames{data.NewFrame("", data.NewField("host", nil, []*string{new("a"), new("b"), new("c")}), data.NewField("n", nil, []*int64{new(int64(9007199254740993)), nil, new(int64(2))}))}}},
		{name: "count null semantics", sql: "SELECT COUNT(*) AS row_count, COUNT(n) AS value_count FROM A", want: Expectation{Frames: data.Frames{data.NewFrame("", data.NewField("row_count", nil, []int64{3}), data.NewField("value_count", nil, []int64{2}))}}},
		{name: "empty result preserves schema", sql: "SELECT host, n FROM A WHERE n < 0", want: Expectation{Frames: data.Frames{data.NewFrame("", data.NewField("host", nil, []*string{}), data.NewField("n", nil, []*int64{}))}}},
		{name: "join two RefIDs", sql: "SELECT A.host, A.n, R.zone FROM A JOIN R ON A.host = R.host ORDER BY A.host", extra: r, want: Expectation{Frames: data.Frames{data.NewFrame("", data.NewField("host", nil, []*string{new("a"), new("c")}), data.NewField("n", nil, []*int64{new(int64(9007199254740993)), new(int64(2))}), data.NewField("zone", nil, []*string{new("west"), new("east")}))}}},
		{name: "JSON null differs from SQL null before serialization", sql: "SELECT COUNT(*) AS row_count, COUNT(doc) AS json_count FROM J", extra: j, want: Expectation{Frames: data.Frames{data.NewFrame("", data.NewField("row_count", nil, []int64{2}), data.NewField("json_count", nil, []int64{1}))}}},
		{name: "expected query error", sql: "SELECT absent FROM A", want: Expectation{ErrorContains: "absent"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			queries := []json.RawMessage{a}
			if tc.extra != nil {
				queries = append(queries, tc.extra)
			}
			queries = append(queries, SQL("B", tc.sql))
			result, err := client.Query(t.Context(), Request{From: "1760000000000", To: "1760000060000", Queries: queries})
			require.NoError(t, err)
			AssertResult(t, result, map[string]Expectation{"B": tc.want})
			t.Logf("observed Grafana %s commit %s through %s", result.GrafanaVersion, result.GrafanaCommit, result.ExecutionPath)
		})
	}
}
