package examples_test

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/sqlexprtest/grafanatest"
	"github.com/stretchr/testify/require"
)

// These pilots exercise actual plugin decoders and Grafana SQL through the
// direct HTTP path. Prometheus uses a deterministic upstream HTTP fixture;
// PostgreSQL uses a real database. Browser preparation is a separate test.
func TestLiveDatasourcePilots(t *testing.T) {
	if os.Getenv("SQLEXPRTEST_PILOTS") != "1" {
		t.Skip("set SQLEXPRTEST_PILOTS=1 with the compose fixtures to run backend pilots")
	}
	url, version := os.Getenv("SQLEXPRTEST_GRAFANA_URL"), os.Getenv("SQLEXPRTEST_GRAFANA_VERSION")
	require.NotEmpty(t, url)
	require.NotEmpty(t, version)
	client := &grafanatest.Client{URL: url, ExpectedVersion: version, Headers: make(http.Header)}
	if token := os.Getenv("SQLEXPRTEST_GRAFANA_TOKEN"); token != "" {
		client.Headers.Set("Authorization", "Bearer "+token)
	}
	t0 := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	postgres := json.RawMessage(`{"refId":"A","datasource":{"uid":"sqlexpr-postgres","type":"grafana-postgresql-datasource"},"rawSql":"SELECT host, bytes, optional FROM samples ORDER BY host","format":"table","editorMode":"code","rawQuery":true}`)
	prometheus := json.RawMessage(`{"refId":"A","datasource":{"uid":"sqlexpr-prometheus","type":"prometheus"},"expr":"fixture_bytes","instant":false,"range":true,"format":"time_series","intervalMs":1000,"maxDataPoints":10}`)
	cases := []struct {
		name  string
		query json.RawMessage
		sql   string
		want  *data.Frame
	}{
		{"postgres/exact integer and null", postgres, "SELECT host, bytes, optional FROM A ORDER BY host", data.NewFrame("",
			data.NewField("host", nil, []*string{new("a"), new("b")}),
			data.NewField("bytes", nil, []*int64{new(int64(9007199254740993)), new(int64(42))}),
			data.NewField("optional", nil, []*int64{nil, new(int64(7))}),
		)},
		{"postgres/null counts", postgres, "SELECT COUNT(*) AS rows_total, COUNT(optional) AS values_total FROM A", data.NewFrame("",
			data.NewField("rows_total", nil, []int64{2}),
			data.NewField("values_total", nil, []int64{1}),
		)},
		{"postgres/empty preserves schema", postgres, "SELECT host, bytes FROM A WHERE bytes < 0", data.NewFrame("",
			data.NewField("host", nil, []*string{}), data.NewField("bytes", nil, []*int64{}),
		)},
		{"prometheus/range labels values and time", prometheus, "SELECT host, __value__, time FROM A ORDER BY host, time", data.NewFrame("",
			data.NewField("host", nil, []*string{new("a"), new("a"), new("b"), new("b")}),
			data.NewField("__value__", nil, []*float64{new(float64(10)), new(float64(11)), new(float64(20)), new(float64(21))}),
			data.NewField("time", nil, []time.Time{t0, t0.Add(time.Second), t0, t0.Add(time.Second)}),
		)},
		{"prometheus/aggregation", prometheus, "SELECT host, SUM(__value__) AS total FROM A GROUP BY host ORDER BY host", data.NewFrame("",
			data.NewField("host", nil, []*string{new("a"), new("b")}),
			data.NewField("total", nil, []*float64{new(float64(21)), new(float64(41))}),
		)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := client.Query(t.Context(), grafanatest.Request{
				From: "1767323045000", To: "1767323046000", Queries: []json.RawMessage{tc.query, grafanatest.SQL("B", tc.sql)},
			})
			require.NoError(t, err)
			grafanatest.AssertResult(t, result, map[string]grafanatest.Expectation{"B": {Frames: data.Frames{tc.want}}})
			t.Logf("Grafana %s: %s", result.GrafanaVersion, tc.name)
		})
	}
}
