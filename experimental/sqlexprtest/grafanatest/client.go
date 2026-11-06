// Package grafanatest runs SQL Expressions in a running Grafana, through its
// HTTP API, and checks the results. It is optional: sqlexprtest checks need no
// server. A result covers only the queries you sent, on that Grafana version.
package grafanatest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// Client connects to an already provisioned Grafana. It does not install plugins
// or configure the server. Supply authentication using Headers or HTTPClient.
// Redirects are rejected. A URL may include Grafana's subpath, but not credentials,
// query parameters, or a fragment. Do not mutate Client while queries run.
type Client struct {
	URL        string
	HTTPClient *http.Client
	Headers    http.Header
	// ExpectedVersion, when nonempty, must equal /api/health's version exactly.
	ExpectedVersion string
	// ExpectedCommit optionally pins /api/health's commit for unreleased builds.
	ExpectedCommit string
	// Timeout bounds health and query requests together. Zero uses 30 seconds.
	Timeout time.Duration
	// MaxResponseBytes bounds each response. Zero uses 16 MiB.
	MaxResponseBytes int64
}

// Request is Grafana's /api/ds/query request. Queries are complete query models,
// including refId and datasource. Keeping their JSON preserves plugin-specific
// fields. From and To are Grafana time strings, for example Unix milliseconds.
type Request struct {
	From    string            `json:"from"`
	To      string            `json:"to"`
	Queries []json.RawMessage `json:"queries"`
}

// SQL creates a SQL Expression query model. The expression should refer to the
// source queries' RefIDs. Use explicit ORDER BY when asserting result order.
func SQL(refID, expression string) json.RawMessage {
	q := struct {
		RefID      string `json:"refId"`
		Datasource struct {
			UID  string `json:"uid"`
			Type string `json:"type"`
		} `json:"datasource"`
		Type       string `json:"type"`
		Expression string `json:"expression"`
		Format     string `json:"format"`
	}{RefID: refID, Type: "sql", Expression: expression, Format: "table"}
	q.Datasource.UID, q.Datasource.Type = "__expr__", "__expr__"
	b, _ := json.Marshal(q) // This structure contains only strings.
	return b
}

// Result records actual HTTP execution, including version and per-query errors.
// An expected query error proves that error behavior, not successful SQL input.
type Result struct {
	GrafanaCommit   string                 `json:"grafanaCommit,omitempty"`
	ExecutionPath   string                 `json:"executionPath"`
	GrafanaVersion  string                 `json:"grafanaVersion"`
	HTTPStatus      int                    `json:"httpStatus"`
	RequestedRefIDs []string               `json:"requestedRefIDs"`
	Results         map[string]QueryResult `json:"results"`
}

// QueryResult is one entry in Grafana's query response. Frames are decoded with
// the SDK's typed frame decoder, retaining int64 values beyond float64 precision.
type QueryResult struct {
	Frames      data.Frames `json:"frames"`
	Error       string      `json:"error,omitempty"`
	Status      int         `json:"status,omitempty"`
	ErrorSource string      `json:"errorSource,omitempty"`
}

// Query verifies /api/health and executes req. Per-query errors are returned in
// Result for contract assertions, including Grafana's HTTP 400 query responses.
// Authentication, malformed responses, missing RefIDs, and transport failures
// return an error. Request validation runs before any HTTP request.
func (c *Client) Query(ctx context.Context, req Request) (*Result, error) {
	refs, err := validateRequest(req)
	if err != nil {
		return nil, err
	}
	base, timeout, limit, err := c.settings()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	b, status, err := c.request(ctx, base+"/api/health", nil, limit)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("grafanatest: health returned HTTP %d", status)
	}
	var health struct {
		Version string `json:"version"`
		Commit  string `json:"commit"`
	}
	if err := json.Unmarshal(b, &health); err != nil || health.Version == "" {
		return nil, errors.New("grafanatest: health response has no valid version")
	}
	if c.ExpectedVersion != "" && health.Version != c.ExpectedVersion {
		return nil, fmt.Errorf("grafanatest: expected Grafana %s, server reports %s", c.ExpectedVersion, health.Version)
	}
	if c.ExpectedCommit != "" && health.Commit != c.ExpectedCommit {
		return nil, fmt.Errorf("grafanatest: expected Grafana commit %s, server reports %s", c.ExpectedCommit, health.Commit)
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("grafanatest: encode request: %w", err)
	}
	b, status, err = c.request(ctx, base+"/api/ds/query", body, limit)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK && status != http.StatusMultiStatus && status != http.StatusBadRequest {
		return nil, fmt.Errorf("grafanatest: query returned HTTP %d", status)
	}
	var result Result
	if err := decodeResult(b, &result); err != nil {
		return nil, errors.New("grafanatest: query response is not valid typed frame JSON")
	}
	if len(result.Results) == 0 {
		return nil, fmt.Errorf("grafanatest: HTTP %d query response has no results", status)
	}
	for _, ref := range refs {
		if _, ok := result.Results[ref]; !ok {
			return nil, fmt.Errorf("grafanatest: query response is missing RefID %q", ref)
		}
	}
	if status == http.StatusBadRequest && !hasQueryError(result.Results) {
		return nil, errors.New("grafanatest: HTTP 400 response contains no query error")
	}
	result.GrafanaVersion, result.HTTPStatus, result.RequestedRefIDs = health.Version, status, refs
	result.GrafanaCommit, result.ExecutionPath = health.Commit, "direct-api"
	return &result, nil
}

func validateRequest(req Request) ([]string, error) {
	if req.From == "" || req.To == "" {
		return nil, errors.New("grafanatest: From and To are required")
	}
	if len(req.Queries) == 0 {
		return nil, errors.New("grafanatest: at least one query is required")
	}
	refs := make([]string, 0, len(req.Queries))
	seen := map[string]bool{}
	for i, raw := range req.Queries {
		var q struct {
			RefID string `json:"refId"`
		}
		if err := json.Unmarshal(raw, &q); err != nil || strings.TrimSpace(q.RefID) == "" {
			return nil, fmt.Errorf("grafanatest: query %d needs a valid JSON object and nonempty refId", i)
		}
		if seen[q.RefID] {
			return nil, fmt.Errorf("grafanatest: duplicate RefID %q", q.RefID)
		}
		seen[q.RefID] = true
		refs = append(refs, q.RefID)
	}
	return refs, nil
}

func (c *Client) settings() (string, time.Duration, int64, error) {
	if c == nil {
		return "", 0, 0, errors.New("grafanatest: nil Client")
	}
	u, err := url.Parse(c.URL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", 0, 0, errors.New("grafanatest: URL must be HTTP(S), without credentials, query, or fragment")
	}
	if c.Timeout < 0 || c.MaxResponseBytes < 0 {
		return "", 0, 0, errors.New("grafanatest: timeout and response limit cannot be negative")
	}
	timeout, limit := c.Timeout, c.MaxResponseBytes
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	if limit == 0 {
		limit = 16 << 20
	}
	// The extra byte used to detect oversized responses must not overflow.
	if limit == int64(^uint64(0)>>1) {
		return "", 0, 0, errors.New("grafanatest: response limit is too large")
	}
	return strings.TrimRight(u.String(), "/"), timeout, limit, nil
}

// requestError preserves errors.Is without displaying a potentially sensitive
// error string from a caller-provided transport.
type requestError struct{ cause error }

func (e requestError) Error() string { return "grafanatest: HTTP request failed" }
func (e requestError) Unwrap() error { return e.cause }

func (c *Client) request(ctx context.Context, endpoint string, body []byte, limit int64) ([]byte, int, error) {
	method := http.MethodGet
	if body != nil {
		method = http.MethodPost
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, 0, requestError{err}
	}
	req.Header = c.Headers.Clone()
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := http.Client{}
	if c.HTTPClient != nil {
		client = *c.HTTPClient
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, requestError{err}
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, resp.StatusCode, requestError{err}
	}
	if int64(len(b)) > limit {
		return nil, resp.StatusCode, fmt.Errorf("grafanatest: response exceeds %d bytes", limit)
	}
	return b, resp.StatusCode, nil
}

// A malformed frame can panic inside the SDK codec. Treat untrusted server
// payloads as protocol failures, never as a crash of the author's test process.
func decodeResult(b []byte, result *Result) (err error) {
	defer func() {
		if recover() != nil {
			err = errors.New("invalid frame JSON")
		}
	}()

	times, err := validateWireFrames(b)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, result); err != nil {
		return err
	}
	for _, query := range result.Results {
		for _, frame := range query.Frames {
			if frame == nil {
				return errors.New("nil frame")
			}
			if _, err := frame.RowLen(); err != nil {
				return err
			}
		}
	}
	for _, column := range times {
		field := result.Results[column.refID].Frames[column.frame].Fields[column.field]
		for row, value := range column.values {
			if value != nil {
				field.SetConcrete(row, *value)
			}
		}
	}
	return nil
}

// wireTimes holds the times of one time column, read from the wire without
// the SDK codec. The codec converts milliseconds to nanoseconds in int64,
// which overflows for dates outside 1677 to 2262.
type wireTimes struct {
	refID        string
	frame, field int
	values       []*time.Time
}

func hasQueryError(results map[string]QueryResult) bool {
	for _, result := range results {
		if result.Error != "" {
			return true
		}
	}
	return false
}

// /api/ds/query returns complete frames. The SDK also accepts schema-only
// fragments, so require the full wire shape before treating empty data as real.
func validateWireFrames(b []byte) ([]wireTimes, error) {
	var envelope struct {
		Results map[string]*struct {
			Frames []json.RawMessage `json:"frames"`
		} `json:"results"`
	}
	if err := json.Unmarshal(b, &envelope); err != nil {
		return nil, err
	}
	var times []wireTimes
	for refID, query := range envelope.Results {
		if query == nil {
			return nil, errors.New("query result must be a non-null object")
		}
		for frameIndex, raw := range query.Frames {
			var frame struct {
				Schema struct {
					Fields json.RawMessage `json:"fields"`
				} `json:"schema"`
				Data struct {
					Values   json.RawMessage    `json:"values"`
					Entities []map[string][]int `json:"entities"`
					Nanos    [][]int64          `json:"nanos"`
				} `json:"data"`
			}
			if err := json.Unmarshal(raw, &frame); err != nil {
				return nil, err
			}

			var arrays [2][]json.RawMessage
			for i, array := range []json.RawMessage{frame.Schema.Fields, frame.Data.Values} {
				array = bytes.TrimSpace(array)
				if len(array) == 0 || array[0] != '[' {
					return nil, errors.New("frame needs schema.fields and data.values arrays")
				}
				if err := json.Unmarshal(array, &arrays[i]); err != nil {
					return nil, err
				}
			}
			if len(arrays[0]) != len(arrays[1]) {
				return nil, errors.New("frame schema and value column counts differ")
			}
			for i, vector := range arrays[1] {
				vector = bytes.TrimSpace(vector)
				if len(vector) == 0 || vector[0] != '[' {
					return nil, errors.New("frame values must be arrays")
				}
				var entities map[string][]int
				if i < len(frame.Data.Entities) {
					entities = frame.Data.Entities[i]
				}
				if err := validateWireNulls(arrays[0][i], vector, entities); err != nil {
					return nil, err
				}
				var nanos []int64
				if i < len(frame.Data.Nanos) {
					nanos = frame.Data.Nanos[i]
				}
				values, err := validateWireValues(arrays[0][i], vector, nanos)
				if err != nil {
					return nil, err
				}
				if values != nil {
					times = append(times, wireTimes{refID: refID, frame: frameIndex, field: i, values: values})
				}
			}
		}
	}
	return times, nil
}

// The SDK codec replaces null in nonnullable scalar vectors with Go zero
// values. Reject that information loss before decoding. Float NaN/Inf values
// legitimately use null placeholders accompanied by entity row indexes.
func validateWireNulls(schema, vector json.RawMessage, entities map[string][]int) error {
	var field struct {
		TypeInfo struct {
			Frame    data.FieldType `json:"frame"`
			Nullable bool           `json:"nullable"`
		} `json:"typeInfo"`
	}
	if err := json.Unmarshal(schema, &field); err != nil {
		return err
	}
	ft := field.TypeInfo.Frame
	if field.TypeInfo.Nullable || ft.Nullable() || ft.JSON() {
		return nil
	}
	var values []json.RawMessage
	if err := json.Unmarshal(vector, &values); err != nil {
		return err
	}
	specialRows := make(map[int]bool)
	if ft == data.FieldTypeFloat32 || ft == data.FieldTypeFloat64 {
		for _, entity := range []string{"NaN", "Inf", "NegInf"} {
			for _, row := range entities[entity] {
				specialRows[row] = true
			}
		}
	}
	for row, value := range values {
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) && !specialRows[row] {
			return errors.New("null value in a nonnullable field without a float entity")
		}
	}
	return nil
}

// The SDK codec converts some wrong JSON types without an error: "257" in an
// int8 field becomes 1, 123 in a string field becomes "123", and 1e39 in a
// float32 field becomes +Inf. It also reads integers through float64, so 1.5
// becomes 1. Check that each value has the JSON type and range that Grafana
// writes for the field. For a time field, also return its times.
func validateWireValues(schema, vector json.RawMessage, nanos []int64) ([]*time.Time, error) {
	var field struct {
		TypeInfo struct {
			Frame data.FieldType `json:"frame"`
		} `json:"typeInfo"`
	}
	if err := json.Unmarshal(schema, &field); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(vector))
	decoder.UseNumber()
	var values []any
	if err := decoder.Decode(&values); err != nil {
		return nil, err
	}
	ft := field.TypeInfo.Frame.NonNullableType()
	if ft == data.FieldTypeJSON {
		return nil, nil
	}
	var times []*time.Time
	if ft == data.FieldTypeTime {
		times = make([]*time.Time, len(values))
	}
	for row, value := range values {
		if value == nil {
			continue // validateWireNulls checks nulls.
		}
		if err := validateWireValue(ft, value); err != nil {
			return nil, fmt.Errorf("row %d: %w", row, err)
		}
		if times != nil {
			ms, _ := strconv.ParseInt(string(value.(json.Number)), 10, 64)
			t := time.UnixMilli(ms).UTC()
			if row < len(nanos) {
				t = t.Add(time.Duration(nanos[row]))
			}
			times[row] = &t
		}
	}
	return times, nil
}

func validateWireValue(ft data.FieldType, value any) error {
	bad := fmt.Errorf("%v is not a %s", value, ft)
	switch ft {
	case data.FieldTypeString:
		if _, ok := value.(string); !ok {
			return bad
		}
		return nil
	case data.FieldTypeBool:
		if _, ok := value.(bool); !ok {
			return bad
		}
		return nil
	}
	number, ok := value.(json.Number)
	if !ok {
		return bad
	}
	var err error
	switch ft {
	case data.FieldTypeInt8:
		_, err = strconv.ParseInt(string(number), 10, 8)
	case data.FieldTypeInt16:
		_, err = strconv.ParseInt(string(number), 10, 16)
	case data.FieldTypeInt32:
		_, err = strconv.ParseInt(string(number), 10, 32)
	case data.FieldTypeInt64, data.FieldTypeTime:
		_, err = strconv.ParseInt(string(number), 10, 64)
	case data.FieldTypeUint8:
		_, err = strconv.ParseUint(string(number), 10, 8)
	case data.FieldTypeUint16, data.FieldTypeEnum:
		_, err = strconv.ParseUint(string(number), 10, 16)
	case data.FieldTypeUint32:
		_, err = strconv.ParseUint(string(number), 10, 32)
	case data.FieldTypeUint64:
		_, err = strconv.ParseUint(string(number), 10, 64)
	case data.FieldTypeFloat32:
		_, err = strconv.ParseFloat(string(number), 32)
	case data.FieldTypeFloat64:
		_, err = strconv.ParseFloat(string(number), 64)
	}
	if err != nil {
		return bad
	}
	return nil
}
