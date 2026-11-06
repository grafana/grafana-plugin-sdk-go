package grafanatest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/require"
)

func testRequest() Request {
	return Request{From: "1000", To: "2000", Queries: []json.RawMessage{SQL("B", "SELECT 1 AS n")}}
}

func responseJSON(t *testing.T, results map[string]QueryResult) []byte {
	t.Helper()
	b, err := json.Marshal(struct {
		Results map[string]QueryResult `json:"results"`
	}{results})
	require.NoError(t, err)
	return b
}

func serveResult(t *testing.T, status int, body []byte) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/health" {
			_, _ = w.Write([]byte(`{"version":"13.3.0","commit":"abc123"}`))
			return
		}
		require.Equal(t, "/api/ds/query", r.URL.Path)
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(s.Close)
	return s
}

func TestClientTypedFramesAndSubpath(t *testing.T) {
	frame := data.NewFrame("server-frame", data.NewField("big", nil, []int64{9007199254740993}), data.NewField("nullable", nil, []*float64{nil}), data.NewField("doc", nil, []json.RawMessage{json.RawMessage(`{"big":9007199254740993}`)}))
	frame.RefID = "B"
	expected := data.NewFrame("ignored-name", data.NewField("big", nil, []int64{9007199254740993}), data.NewField("nullable", nil, []*float64{nil}), data.NewField("doc", nil, []json.RawMessage{json.RawMessage(`{ "big":9007199254740993 }`)}))
	body := responseJSON(t, map[string]QueryResult{"B": {Frames: data.Frames{frame}}})
	calls := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		require.Empty(t, r.Header.Get("X-Grafana-From-Expr"))
		if r.URL.Path == "/grafana/api/health" {
			_, _ = w.Write([]byte(`{"version":"13.3.0","commit":"abc123"}`))
			return
		}
		require.Equal(t, "/grafana/api/ds/query", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		var req Request
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		require.JSONEq(t, string(testRequest().Queries[0]), string(req.Queries[0]))
		_, _ = w.Write(body)
	}))
	defer s.Close()
	headers := http.Header{"Authorization": {"Bearer test-token"}}
	c := Client{URL: s.URL + "/grafana/", Headers: headers, ExpectedVersion: "13.3.0", ExpectedCommit: "abc123", HTTPClient: s.Client()}
	result, err := c.Query(t.Context(), testRequest())
	require.NoError(t, err)
	require.Equal(t, 2, calls)
	require.Equal(t, "13.3.0", result.GrafanaVersion)
	require.Equal(t, "abc123", result.GrafanaCommit)
	require.Equal(t, "direct-api", result.ExecutionPath)
	require.Equal(t, []string{"B"}, result.RequestedRefIDs)
	require.Equal(t, int64(9007199254740993), result.Results["B"].Frames[0].Fields[0].At(0))
	require.Empty(t, Check(result, map[string]Expectation{"B": {Frames: data.Frames{expected}}}))
	require.Equal(t, http.Header{"Authorization": {"Bearer test-token"}}, headers)
}

func TestClientResponseFailures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"authentication", 401, `{"message":"Bearer secret"}`, "HTTP 401"},
		{"invalid JSON", 200, `{`, "not valid typed frame JSON"},
		{"empty", 200, `{"results":{}}`, "no results"},
		{"null query result", 200, `{"results":{"B":null}}`, "not valid typed frame JSON"},
		{"missing ref", 200, `{"results":{"A":{}}}`, "missing RefID"},
		{"missing frame schema", 200, `{"results":{"B":{"frames":[{}]}}}`, "not valid typed frame JSON"},
		{"missing frame data", 200, `{"results":{"B":{"frames":[{"schema":{"fields":[]}}]}}}`, "not valid typed frame JSON"},
		{"missing value vector", 200, `{"results":{"B":{"frames":[{"schema":{"fields":[{"name":"n","type":"number","typeInfo":{"frame":"int64"}}]},"data":{"values":[]}}]}}}`, "not valid typed frame JSON"},
		{"invalid frame", 200, `{"results":{"B":{"frames":[{"schema":{"fields":[{"name":"n"}]},"data":{"values":[[1]]}}]}}}`, "not valid typed frame JSON"},
		{"false HTTP400 success", 400, `{"results":{"B":{"frames":[]}}}`, "contains no query error"},
		{"panicking wire", 200, `{"results":{"B":{"frames":[{"schema":{"fields":[]},"data":{"entities":[{"NaN":[0]}]}}]}}}`, "not valid typed frame JSON"},
		{"ordinary HTTP400", 400, `{"message":"bad request"}`, "no results"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := serveResult(t, tc.status, []byte(tc.body))
			_, err := (&Client{URL: s.URL}).Query(t.Context(), testRequest())
			require.ErrorContains(t, err, tc.want)
			require.NotContains(t, err.Error(), "secret")
		})
	}
}

func TestClientPerQueryErrors(t *testing.T) {
	s := serveResult(t, http.StatusBadRequest, responseJSON(t, map[string]QueryResult{"B": {Error: "column host not found", Status: 400}}))
	result, err := (&Client{URL: s.URL}).Query(t.Context(), testRequest())
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, result.HTTPStatus)
	require.Empty(t, Check(result, map[string]Expectation{"B": {ErrorContains: "host not found"}}))
	require.NotEmpty(t, Check(result, map[string]Expectation{"B": {Frames: data.Frames{}}}))
}

func TestClientVersionAndLimit(t *testing.T) {
	s := serveResult(t, http.StatusOK, responseJSON(t, map[string]QueryResult{"B": {Frames: data.Frames{}}}))
	_, err := (&Client{URL: s.URL, ExpectedVersion: "13.2.0"}).Query(t.Context(), testRequest())
	require.ErrorContains(t, err, "expected Grafana 13.2.0, server reports 13.3.0")
	_, err = (&Client{URL: s.URL, ExpectedCommit: "wrong"}).Query(t.Context(), testRequest())
	require.ErrorContains(t, err, "expected Grafana commit wrong, server reports abc123")
	_, err = (&Client{URL: s.URL, MaxResponseBytes: 8}).Query(t.Context(), testRequest())
	require.ErrorContains(t, err, "exceeds 8 bytes")
}

func TestClientRejectsInvalidRequestBeforeHTTP(t *testing.T) {
	calls := 0
	s := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
	defer s.Close()
	for _, req := range []Request{{}, {From: "1", To: "2"}, {From: "1", To: "2", Queries: []json.RawMessage{SQL("", "SELECT 1")}}, {From: "1", To: "2", Queries: []json.RawMessage{SQL("B", "SELECT 1"), SQL("B", "SELECT 2")}}, {From: "1", To: "2", Queries: []json.RawMessage{json.RawMessage(`[]`)}}} {
		_, err := (&Client{URL: s.URL}).Query(t.Context(), req)
		require.Error(t, err)
	}
	require.Zero(t, calls)
	for _, url := range []string{"", "file:///etc/passwd", "https://user:secret@example.com", "https://example.com?token=secret"} {
		_, err := (&Client{URL: url}).Query(t.Context(), testRequest())
		require.Error(t, err)
		require.NotContains(t, err.Error(), "secret")
	}
}

func TestClientTimeout(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer s.Close()
	_, err := (&Client{URL: s.URL, Timeout: 20 * time.Millisecond}).Query(t.Context(), testRequest())
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestClientDoesNotFollowRedirects(t *testing.T) {
	leaked := false
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { leaked = true }))
	defer target.Close()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer s.Close()
	_, err := (&Client{URL: s.URL, Headers: http.Header{"Authorization": {"Bearer secret"}, "X-API-Key": {"secret"}}}).Query(t.Context(), testRequest())
	require.ErrorContains(t, err, "HTTP 307")
	require.False(t, leaked)
}

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestClientRedactsTransportErrors(t *testing.T) {
	cause := errors.New("Bearer secret from custom transport")
	client := &http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) { return nil, cause })}
	_, err := (&Client{URL: "https://example.invalid", HTTPClient: client}).Query(t.Context(), testRequest())
	require.ErrorIs(t, err, cause)
	require.False(t, strings.Contains(fmt.Sprint(err), "secret"))
}

func TestDecodeResultRejectsNullDefaults(t *testing.T) {
	for _, ft := range append(data.NumericFieldTypes(), data.FieldTypeString, data.FieldTypeBool, data.FieldTypeTime, data.FieldTypeEnum) {
		if ft.Nullable() {
			continue
		}
		for _, position := range []int{0, 1} {
			t.Run(fmt.Sprintf("%s/column%d", ft, position), func(t *testing.T) {
				fields := []*data.Field{data.NewField("prefix", nil, []string{"present"})}
				field := data.NewFieldFromFieldType(ft, 1)
				field.Name = "n"
				fields = append(fields[:position], field)
				wire := responseJSON(t, map[string]QueryResult{"B": {Frames: data.Frames{data.NewFrame("", fields...)}}})
				var body map[string]any
				require.NoError(t, json.Unmarshal(wire, &body))
				frames := body["results"].(map[string]any)["B"].(map[string]any)["frames"].([]any)
				values := frames[0].(map[string]any)["data"].(map[string]any)["values"].([]any)
				values[position].([]any)[0] = nil
				wire, err := json.Marshal(body)
				require.NoError(t, err)
				require.ErrorContains(t, decodeResult(wire, &Result{}), "null value in a nonnullable field")
			})
		}
	}
}

func TestDecodeResultPreservesValidNullsAndFloatEntities(t *testing.T) {
	fields := []*data.Field{
		data.NewField("float32", nil, []float32{float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1))}),
		data.NewField("float64", nil, []float64{math.NaN(), math.Inf(1), math.Inf(-1)}),
		data.NewField("nullable_float", nil, []*float64{nil, new(math.NaN()), new(math.Inf(1))}),
		data.NewField("json", nil, []json.RawMessage{nil, json.RawMessage("null"), json.RawMessage(`{"n":null}`)}),
	}
	for _, ft := range append(data.NumericFieldTypes(), data.FieldTypeString, data.FieldTypeBool, data.FieldTypeTime, data.FieldTypeEnum, data.FieldTypeJSON) {
		field := data.NewFieldFromFieldType(ft.NullableType(), 3)
		field.Name = ft.String()
		fields = append(fields, field)
	}
	frame := data.NewFrame("", fields...)
	wire := responseJSON(t, map[string]QueryResult{"B": {Frames: data.Frames{frame}}})
	result := &Result{GrafanaVersion: "13.3.0"}
	require.NoError(t, decodeResult(wire, result))
	require.Empty(t, Check(result, map[string]Expectation{"B": {Frames: data.Frames{frame}}}))
}
