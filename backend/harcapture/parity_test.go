package harcapture

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	chhar "github.com/chromedp/cdproto/har"

	"github.com/grafana/grafana-plugin-sdk-go/backend/querycapture"
)

// TestHARParity_UnmarshalsIntoChromedpHAR verifies our hand-rolled HAR output parses into the
// canonical chromedp/cdproto/har model (what experimental/e2e/storage + the E2E fixture proxy use).
func TestHARParity_UnmarshalsIntoChromedpHAR(t *testing.T) {
	const reqBody = `{"x":1}`
	req, err := http.NewRequest(http.MethodPost, "https://api.example.com/query?a=1&b=2", strings.NewReader(reqBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Proto = "HTTP/1.1"
	req.Header.Set("Content-Type", "application/json")
	// Secure/HttpOnly/SameSite are set only to satisfy gosec G124; they are response-cookie
	// attributes and are dropped when this is serialized into the outgoing Cookie header.
	req.AddCookie(&http.Cookie{Name: "session", Value: "abc", Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode})

	rec := httptest.NewRecorder()
	rec.Header().Set("Content-Type", "application/json")
	rec.Header().Set("Set-Cookie", "sid=xyz")
	_, _ = rec.WriteString(`{"ok":true}`)
	resp := rec.Result()
	resp.Proto = "HTTP/1.1"

	buf := NewBuffer()
	buf.AddEntry(req, []byte(reqBody), false, resp, nil, time.Now(), 5*time.Millisecond)
	s, err := buf.ToHARString()
	if err != nil {
		t.Fatal(err)
	}

	var h chhar.HAR
	if err := json.Unmarshal([]byte(s), &h); err != nil {
		t.Fatalf("our HAR does NOT unmarshal into chromedp har.HAR: %v", err)
	}
	if h.Log == nil || len(h.Log.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %+v", h.Log)
	}
	e := h.Log.Entries[0]
	if e.Request == nil || e.Request.Method != "POST" {
		t.Fatalf("request method not parsed: %+v", e.Request)
	}
	if len(e.Request.Cookies) != 1 || e.Request.Cookies[0].Name != "session" {
		t.Errorf("request cookie not parsed: %+v", e.Request.Cookies)
	}
	if len(e.Response.Cookies) != 1 || e.Response.Cookies[0].Name != "sid" {
		t.Errorf("response cookie not parsed: %+v", e.Response.Cookies)
	}
	if e.Cache == nil {
		t.Errorf("cache object missing")
	}
	t.Logf("OUR HAR (round-trips into chromedp har.HAR):\n%s", s)
}

// TestHARParity_QueryEntriesUnmarshalIntoChromedpHAR is the same guarantee for the entries that
// describe a non-HTTP exchange, which is what makes putting them in a HAR defensible at all: a query
// entry fills the HTTP fields with either nothing (no httpVersion, headers or cookies) or the query's
// own equivalent (a "QUERY" method, a zero status for a failed call), and puts the rest under
// "_query". Each of those is somewhere a stricter reader could refuse the document, so assert that
// the canonical model takes them and that "_query" survives for a reader that wants it.
//
// The document mixes both producers deliberately: a plugin whose driver is itself HTTP-backed
// (ClickHouse, BigQuery, Athena) emits both kinds into one buffer, so that -- not a query-only
// document -- is the shape a consumer actually receives.
func TestHARParity_QueryEntriesUnmarshalIntoChromedpHAR(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://clickhouse.example.com/ping", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Proto = "HTTP/1.1"
	rec := httptest.NewRecorder()
	rec.Header().Set("Content-Type", "text/plain")
	_, _ = rec.WriteString("Ok.\n")
	resp := rec.Result()
	resp.Proto = "HTTP/1.1"

	buf := NewBuffer()
	buf.AddEntry(req, nil, false, resp, nil, time.Now(), 2*time.Millisecond)
	buf.AddQueryInteraction(querycapture.Interaction{
		Kind:           querycapture.KindSQLQuery,
		StartedAt:      time.Now(),
		Duration:       750 * time.Microsecond,
		DatasourceUID:  "P1234",
		DatasourceType: "grafana-clickhouse-datasource",
		RefID:          "A",
		Statement:      "SELECT host, value FROM metrics WHERE host = ?",
		Args:           []string{"host-a"},
		FrameCount:     1,
		RowCount:       10,
	})
	// The failed call is the awkward one to parse: a zero status, no content and the error in the
	// entry comment.
	buf.AddQueryInteraction(querycapture.Interaction{
		Kind:      querycapture.KindSQLQuery,
		StartedAt: time.Now(),
		RefID:     "B",
		Statement: "SELECT * FROM missing",
		RowCount:  -1,
		Err:       "table missing does not exist",
	})

	s, err := buf.ToHARString()
	if err != nil {
		t.Fatal(err)
	}

	var h chhar.HAR
	if err := json.Unmarshal([]byte(s), &h); err != nil {
		t.Fatalf("a document with query entries does NOT unmarshal into chromedp har.HAR: %v", err)
	}
	if h.Log == nil || len(h.Log.Entries) != 3 {
		t.Fatalf("expected 3 entries, got %+v", h.Log)
	}
	if got := h.Log.Entries[0].Request.Method; got != http.MethodGet {
		t.Errorf("HTTP entry did not survive alongside the query entries: method %q", got)
	}

	ok, failed := h.Log.Entries[1], h.Log.Entries[2]
	if ok.Request == nil || ok.Request.Method != queryMethod {
		t.Fatalf("query request not parsed: %+v", ok.Request)
	}
	if ok.Request.URL != "sql://grafana-clickhouse-datasource/P1234?refId=A" {
		t.Errorf("query URL not parsed: %q", ok.Request.URL)
	}
	// The statement is the evidence the entry exists for, so it has to reach a canonical reader.
	if ok.Request.PostData == nil || ok.Request.PostData.Text != "SELECT host, value FROM metrics WHERE host = ?" {
		t.Errorf("statement did not survive as postData.text: %+v", ok.Request.PostData)
	}
	if ok.Request.PostData != nil && len(ok.Request.PostData.Params) != 0 {
		t.Errorf("postData carries text, so params must stay absent: %+v", ok.Request.PostData.Params)
	}
	if ok.Response == nil || ok.Response.Status != 200 {
		t.Errorf("successful query response not parsed: %+v", ok.Response)
	}
	if failed.Response == nil || failed.Response.Status != 0 {
		t.Errorf("failed query response not parsed: %+v", failed.Response)
	}
	if failed.Comment != "query error: table missing does not exist" {
		t.Errorf("query error not parsed from the entry comment: %q", failed.Comment)
	}
	// Nil here would mean the canonical model saw no object at all, which the spec requires per entry.
	if ok.Cache == nil || ok.Timings == nil {
		t.Errorf("cache/timings object missing on a query entry: cache=%v timings=%v", ok.Cache, ok.Timings)
	}
	if ok.Timings.Wait != 0.75 {
		t.Errorf("sub-millisecond wait did not survive: %v", ok.Timings.Wait)
	}

	// The canonical model has no field for "_query" and drops it, which is the point -- but a consumer
	// that does read it must still find it on the entry, so check the wire form too.
	var raw struct {
		Log struct {
			Entries []map[string]json.RawMessage `json:"entries"`
		} `json:"log"`
	}
	if err := json.Unmarshal([]byte(s), &raw); err != nil {
		t.Fatal(err)
	}
	if _, present := raw.Log.Entries[0]["_query"]; present {
		t.Errorf("an HTTP entry must not carry a _query object")
	}
	for _, i := range []int{1, 2} {
		if _, present := raw.Log.Entries[i]["_query"]; !present {
			t.Errorf("entry %d lost its _query object", i)
		}
	}
	t.Logf("OUR HAR with query entries (round-trips into chromedp har.HAR):\n%s", s)
}
