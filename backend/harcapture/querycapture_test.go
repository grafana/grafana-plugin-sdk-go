package harcapture

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend/querycapture"
)

// harDoc is the parsed shape of the emitted document, enough to assert on a query entry.
type harDoc struct {
	Log struct {
		Entries []struct {
			StartedDateTime string  `json:"startedDateTime"`
			Time            float64 `json:"time"`
			Request         struct {
				Method   string `json:"method"`
				URL      string `json:"url"`
				BodySize int64  `json:"bodySize"`
				PostData *struct {
					MimeType string `json:"mimeType"`
					Text     string `json:"text"`
					Params   []struct {
						Name  string `json:"name"`
						Value string `json:"value"`
					} `json:"params"`
				} `json:"postData"`
			} `json:"request"`
			Response struct {
				Status   int   `json:"status"`
				BodySize int64 `json:"bodySize"`
				Content  struct {
					MimeType string `json:"mimeType"`
					Text     string `json:"text"`
					Size     int64  `json:"size"`
				} `json:"content"`
			} `json:"response"`
			Comment string `json:"comment"`
			Query   *struct {
				Kind               string `json:"kind"`
				DatasourceUID      string `json:"datasourceUid"`
				DatasourceType     string `json:"datasourceType"`
				DatasourceName     string `json:"datasourceName"`
				RefID              string `json:"refId"`
				StatementTruncated bool   `json:"statementTruncated"`
				ArgsTruncated      bool   `json:"argsTruncated"`
				FrameCount         int    `json:"frameCount"`
				RowCount           int    `json:"rowCount"`
				Error              string `json:"error"`
			} `json:"_query"`
		} `json:"entries"`
	} `json:"log"`
}

func strPtr(s string) *string { return &s }

func toDoc(t *testing.T, b *Buffer) harDoc {
	t.Helper()
	raw, err := b.ToHARString()
	require.NoError(t, err)
	var doc harDoc
	require.NoError(t, json.Unmarshal([]byte(raw), &doc))
	return doc
}

func TestAddQueryInteraction_successEntry(t *testing.T) {
	started := time.Date(2026, 7, 28, 10, 30, 0, 0, time.UTC)
	b := NewBuffer()
	b.AddQueryInteraction(querycapture.Interaction{
		Kind:           querycapture.KindSQLQuery,
		StartedAt:      started,
		Duration:       25 * time.Millisecond,
		DatasourceUID:  "P1234",
		DatasourceType: "grafana-clickhouse-datasource",
		DatasourceName: "ClickHouse diag",
		RefID:          "A",
		Statement:      "SELECT host, value FROM metrics WHERE host = ?",
		Args:           []string{"host-a"},
		FrameCount:     1,
		RowCount:       10,
	})

	doc := toDoc(t, b)
	require.Len(t, doc.Log.Entries, 1)
	e := doc.Log.Entries[0]

	// The statement is the request body, verbatim: it is the evidence, so nothing may be appended to
	// it. The datasource identity is addressed by the URL, not smuggled into the SQL.
	assert.Equal(t, "QUERY", e.Request.Method)
	assert.Equal(t, "sql://grafana-clickhouse-datasource/P1234?refId=A", e.Request.URL)
	require.NotNil(t, e.Request.PostData)
	assert.Equal(t, "application/sql", e.Request.PostData.MimeType)
	assert.Equal(t, "SELECT host, value FROM metrics WHERE host = ?", e.Request.PostData.Text)
	assert.Equal(t, int64(len("SELECT host, value FROM metrics WHERE host = ?")), e.Request.BodySize)

	// Bind arguments ride in the spec's params array, positionally named.
	require.Len(t, e.Request.PostData.Params, 1)
	assert.Equal(t, "$1", e.Request.PostData.Params[0].Name)
	assert.Equal(t, "host-a", e.Request.PostData.Params[0].Value)

	// A successful query is a 200 whose body counts what came back rather than repeating it.
	assert.Equal(t, 200, e.Response.Status)
	assert.Equal(t, "application/json", e.Response.Content.MimeType)
	assert.JSONEq(t, `{"frameCount":1,"rowCount":10}`, e.Response.Content.Text)
	assert.Equal(t, int64(len(e.Response.Content.Text)), e.Response.Content.Size)
	assert.Empty(t, e.Comment)

	assert.Equal(t, "2026-07-28T10:30:00Z", e.StartedDateTime)
	assert.Equal(t, float64(25), e.Time)

	require.NotNil(t, e.Query)
	assert.Equal(t, querycapture.KindSQLQuery, e.Query.Kind)
	assert.Equal(t, "P1234", e.Query.DatasourceUID)
	assert.Equal(t, "grafana-clickhouse-datasource", e.Query.DatasourceType)
	assert.Equal(t, "ClickHouse diag", e.Query.DatasourceName)
	assert.Equal(t, "A", e.Query.RefID)
	assert.Equal(t, 1, e.Query.FrameCount)
	assert.Equal(t, 10, e.Query.RowCount)
	assert.Empty(t, e.Query.Error)
}

func TestAddQueryInteraction_failedQueryIsRecorded(t *testing.T) {
	// A failed query is the most valuable case for diagnostics, so it must be present, and
	// distinguishable from a query that returned no rows.
	b := NewBuffer()
	b.AddQueryInteraction(querycapture.Interaction{
		Kind:          querycapture.KindSQLQuery,
		StartedAt:     time.Now(),
		DatasourceUID: "P1234",
		Statement:     "SELECT * FROM missing_table",
		Err:           "code: 60, message: Unknown table expression identifier 'missing_table'",
		RowCount:      -1,
	})

	doc := toDoc(t, b)
	require.Len(t, doc.Log.Entries, 1)
	e := doc.Log.Entries[0]

	assert.Equal(t, 0, e.Response.Status, "a failed query has no response, so status is zero, not 200")
	assert.Equal(t, int64(-1), e.Response.BodySize, "-1 is HAR 'unavailable'; 0 would claim an empty response body")
	assert.Empty(t, e.Response.Content.Text)
	assert.Contains(t, e.Comment, "query error: code: 60")
	require.NotNil(t, e.Query)
	assert.Contains(t, e.Query.Error, "Unknown table expression identifier")
	assert.Equal(t, -1, e.Query.RowCount)
}

func TestAddQueryInteraction_truncationIsReported(t *testing.T) {
	b := NewBuffer()
	b.AddQueryInteraction(querycapture.Interaction{
		Kind:               querycapture.KindSQLQuery,
		Statement:          strings.Repeat("x", 128),
		StatementTruncated: true,
		Args:               []string{"a"},
		ArgsTruncated:      true,
		RowCount:           3,
		FrameCount:         1,
	})

	doc := toDoc(t, b)
	require.Len(t, doc.Log.Entries, 1)
	e := doc.Log.Entries[0]
	assert.Equal(t, int64(-1), e.Request.BodySize, "a clipped statement reports an unavailable size, not the clipped length")
	require.NotNil(t, e.Query)
	assert.True(t, e.Query.StatementTruncated)
	assert.True(t, e.Query.ArgsTruncated)
}

func TestAddQueryInteraction_kindDrivesURLScheme(t *testing.T) {
	// The scheme comes from the capture kind so a future non-SQL capture point reads correctly without
	// this package knowing about it.
	for _, tc := range []struct {
		kind, want string
	}{
		{querycapture.KindSQLQuery, "sql://t/uid"},
		{"mongo.command", "mongo://t/uid"},
		{"redis", "redis://t/uid"},
		{"", "query://t/uid"},
	} {
		b := NewBuffer()
		b.AddQueryInteraction(querycapture.Interaction{Kind: tc.kind, DatasourceType: "t", DatasourceUID: "uid"})
		doc := toDoc(t, b)
		require.Len(t, doc.Log.Entries, 1)
		assert.Equal(t, tc.want, doc.Log.Entries[0].Request.URL, "kind %q", tc.kind)
	}
}

func TestNewRecorder_writesIntoBufferInterleavedWithHTTP(t *testing.T) {
	// The recorder shares the HTTP buffer, which is the whole point: a plugin whose driver speaks HTTP
	// (ClickHouse, BigQuery, Athena) produces both kinds in one request and must not have to choose.
	b := NewBuffer()
	rec := NewRecorder(b)
	rec.Record(querycapture.Interaction{Kind: querycapture.KindSQLQuery, Statement: "SELECT 1", DatasourceUID: "P1"})
	rec.Record(querycapture.Interaction{Kind: querycapture.KindSQLQuery, Statement: "SELECT 2", DatasourceUID: "P1"})

	assert.Equal(t, 2, b.Len())
	doc := toDoc(t, b)
	require.Len(t, doc.Log.Entries, 2)
	assert.Equal(t, "SELECT 1", doc.Log.Entries[0].Request.PostData.Text)
	assert.Equal(t, "SELECT 2", doc.Log.Entries[1].Request.PostData.Text, "entries keep the order they were recorded in")
}

func TestNewRecorder_nilBufferIsInert(t *testing.T) {
	// Capture must never be the reason a query fails, so a misconfigured recorder drops the
	// interaction rather than panicking on the query path.
	assert.NotPanics(t, func() {
		NewRecorder(nil).Record(querycapture.Interaction{Kind: querycapture.KindSQLQuery})
	})
}

func TestAddQueryInteraction_carriesReturnedRows(t *testing.T) {
	// The rows the database returned are the evidence that makes a bundle able to localize a wrong-data
	// report: without them, "the database returned the wrong rows" and "the plugin mangled correct rows"
	// look identical. They ride in the response body, exactly where an HTTP entry carries its body.
	b := NewBuffer()
	b.AddQueryInteraction(querycapture.Interaction{
		Kind:            querycapture.KindSQLQuery,
		DatasourceUID:   "P1234",
		Statement:       "SELECT host, value FROM metrics",
		ResultColumns:   []string{"host", "value"},
		ResultRows:      [][]*string{{strPtr("host-a"), strPtr("1")}, {strPtr("host-b"), nil}},
		ResultTotalRows: 2,
		FrameCount:      1,
		RowCount:        1,
	})

	doc := toDoc(t, b)
	require.Len(t, doc.Log.Entries, 1)
	e := doc.Log.Entries[0]
	assert.Equal(t, 200, e.Response.Status)
	assert.Equal(t, "application/json", e.Response.Content.MimeType)
	// A NULL stays null and is not flattened into an empty string; the plugin's own counts sit alongside
	// the rows so the comparison can be made inside one object.
	assert.JSONEq(t, `{
		"columns": ["host","value"],
		"rows": [["host-a","1"],["host-b",null]],
		"totalRows": 2,
		"frameCount": 1,
		"rowCount": 1
	}`, e.Response.Content.Text)
	assert.Equal(t, int64(len(e.Response.Content.Text)), e.Response.BodySize)
}

func TestAddQueryInteraction_clippedRowsAreReported(t *testing.T) {
	b := NewBuffer()
	b.AddQueryInteraction(querycapture.Interaction{
		Kind:                querycapture.KindSQLQuery,
		Statement:           "SELECT * FROM big",
		ResultColumns:       []string{"host"},
		ResultRows:          [][]*string{{strPtr("host-a")}},
		ResultRowsTruncated: true,
		ResultTotalRows:     900,
		FrameCount:          1,
		RowCount:            900,
	})

	doc := toDoc(t, b)
	e := doc.Log.Entries[0]
	assert.Equal(t, int64(-1), e.Response.BodySize,
		"a clipped result reports an unavailable body size, the same as an over-cap HTTP body")
	assert.Contains(t, e.Response.Content.Text, `"rowsTruncated":true`)
	assert.Contains(t, e.Response.Content.Text, `"totalRows":900`,
		"the total says how much there was, so a prefix is not mistaken for the whole result")
}

func TestAddQueryInteraction_unknownTotalIsOmitted(t *testing.T) {
	// A capture point that hands back an unconsumed result set cannot count it (-1); claiming zero would
	// assert an empty result that was never observed.
	b := NewBuffer()
	b.AddQueryInteraction(querycapture.Interaction{
		Kind:            querycapture.KindSQLQuery,
		Statement:       "SELECT 1",
		ResultTotalRows: -1,
		RowCount:        -1,
	})

	doc := toDoc(t, b)
	assert.NotContains(t, doc.Log.Entries[0].Response.Content.Text, "totalRows")
}

func TestAddQueryInteraction_failedQueryKeepsPartialRows(t *testing.T) {
	// A query that failed after the database had already returned rows: those rows are the most direct
	// evidence of what went wrong, so they are kept -- while the status stays zero, because the query
	// did not complete.
	b := NewBuffer()
	b.AddQueryInteraction(querycapture.Interaction{
		Kind:            querycapture.KindSQLQuery,
		Statement:       "SELECT host, value FROM metrics",
		ResultColumns:   []string{"host", "value"},
		ResultRows:      [][]*string{{strPtr("host-a"), strPtr("not-a-number")}},
		ResultTotalRows: 1,
		Err:             "converting driver.Value type string to a float64",
	})

	doc := toDoc(t, b)
	e := doc.Log.Entries[0]
	assert.Equal(t, 0, e.Response.Status)
	assert.Contains(t, e.Comment, "query error: converting driver.Value")
	assert.Contains(t, e.Response.Content.Text, "not-a-number")
}

func TestAddQueryInteraction_emptyResultIsDistinctFromUncaptured(t *testing.T) {
	// Zero rows is an answer; "nobody recorded a result" is not. The two must not look alike.
	emptyResult := NewBuffer()
	emptyResult.AddQueryInteraction(querycapture.Interaction{
		Kind:          querycapture.KindSQLQuery,
		Statement:     "SELECT host FROM metrics WHERE 1 = 0",
		ResultColumns: []string{"host"},
	})
	assert.Contains(t, toDoc(t, emptyResult).Log.Entries[0].Response.Content.Text, `"totalRows":0`,
		"a capture point that ran and saw no rows reports zero")

	uncaptured := NewBuffer()
	uncaptured.AddQueryInteraction(querycapture.Interaction{
		Kind:      querycapture.KindSQLQuery,
		Statement: "SELECT host FROM metrics",
	})
	assert.NotContains(t, toDoc(t, uncaptured).Log.Entries[0].Response.Content.Text, "totalRows",
		"an interaction with no result capture claims nothing about how many rows there were")
}
