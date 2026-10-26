package grafanatest

import (
	"encoding/json"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/require"
)

func oneResult(f *data.Frame) *Result {
	return &Result{GrafanaVersion: "13.3.0", Results: map[string]QueryResult{"B": {Frames: data.Frames{f}}}}
}

func TestContractsDetectMutations(t *testing.T) {
	expected := func() *data.Frame {
		return data.NewFrame("", data.NewField("n", nil, []int64{9007199254740993}), data.NewField("nullable", nil, []*float64{nil}))
	}
	for _, tc := range []struct {
		name   string
		mutate func(*data.Frame)
		kind   string
	}{
		{"rounded integer", func(f *data.Frame) { f.Fields[0].Set(0, int64(9007199254740992)) }, "value"},
		{"wrong column", func(f *data.Frame) { f.Fields[0].Name = "changed" }, "column-name"},
		{"wrong type", func(f *data.Frame) { f.Fields[0] = data.NewField("n", nil, []float64{9007199254740992}) }, "column-type"},
		{"null became zero", func(f *data.Frame) { f.Fields[1].Set(0, new(float64(0))) }, "value"},
		{"extra row", func(f *data.Frame) { f.AppendRow(int64(2), new(float64(3))) }, "row-count"},
		{"dropped field", func(f *data.Frame) { f.Fields = f.Fields[:1] }, "column-count"},
		{"malformed", func(f *data.Frame) { f.Fields[0] = nil }, "frame"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := expected()
			tc.mutate(f)
			bad := Check(oneResult(f), map[string]Expectation{"B": {Frames: data.Frames{expected()}}})
			found := false
			for _, m := range bad {
				if m.Kind == tc.kind {
					found = true
					require.Contains(t, m.String(), "query")
				}
			}
			require.True(t, found, "expected %s in %v", tc.kind, bad)
		})
	}
}

func TestContractsFailClosed(t *testing.T) {
	f := data.NewFrame("", data.NewField("n", nil, []int64{1}))
	r := oneResult(f)
	require.NotEmpty(t, Check(nil, map[string]Expectation{"B": {Frames: data.Frames{f}}}))
	require.NotEmpty(t, Check(r, nil))
	require.NotEmpty(t, Check(r, map[string]Expectation{"B": {}}))
	require.NotEmpty(t, Check(r, map[string]Expectation{"B": {Frames: data.Frames{f}, ErrorContains: "bad"}}))
	require.NotEmpty(t, Check(r, map[string]Expectation{"C": {Frames: data.Frames{f}}}))
	r.Results["A"] = QueryResult{Error: "source failed"}
	require.NotEmpty(t, Check(r, map[string]Expectation{"B": {Frames: data.Frames{f}}}))
	r.Results["A"] = QueryResult{Status: 500}
	require.NotEmpty(t, Check(r, map[string]Expectation{"B": {Frames: data.Frames{f}}}))
}

func TestContractsNoticeAndTruncation(t *testing.T) {
	notice := data.Notice{Severity: data.NoticeSeverityWarning, Text: "Query exceeded max output cells (2). Only 2 cells returned."}
	f := data.NewFrame("", data.NewField("n", nil, []int64{1, 2}))
	f.Meta = &data.FrameMeta{Notices: []data.Notice{notice}}
	r := oneResult(f)
	require.NotEmpty(t, Check(r, map[string]Expectation{"B": {Frames: data.Frames{f}}}))
	require.Empty(t, Check(r, map[string]Expectation{"B": {Frames: data.Frames{f}, Notices: []data.Notice{notice}}}))
	r.Results["A"] = QueryResult{Frames: data.Frames{f}}
	require.NotEmpty(t, Check(r, map[string]Expectation{"B": {Frames: data.Frames{f}, Notices: []data.Notice{notice}}}))
}

func TestJSONComparisonPreservesPrecision(t *testing.T) {
	require.True(t, equalJSON(json.RawMessage(`{"b":null,"a":9007199254740993}`), json.RawMessage(`{"a":9007199254740993, "b":null}`)))
	require.False(t, equalJSON(json.RawMessage(`9007199254740993`), json.RawMessage(`9007199254740992`)))
	require.True(t, equalJSON(nil, json.RawMessage(`null`)))
	require.False(t, equalJSON(json.RawMessage{}, json.RawMessage(`null`)))
	require.False(t, equalJSON(json.RawMessage(`{"a":1} garbage`), json.RawMessage(`{"a":1}`)))
}
