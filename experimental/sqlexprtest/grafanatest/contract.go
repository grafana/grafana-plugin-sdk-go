package grafanatest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// Expectation asserts one query's result. Set Frames (including an explicitly
// empty slice for no frames) for success, or ErrorContains for an expected
// failure. Never set both. Frame order, field names/types/labels, rows, and values
// are exact; frame names, RefIDs and presentation metadata are ignored.
// Notices must match exactly in frame order, including info notices. This makes
// output-limit warnings fail by default instead of silently accepting truncation.
type Expectation struct {
	Frames        data.Frames
	ErrorContains string
	Notices       []data.Notice
}

// Mismatch identifies a failed runtime contract. Frame and Row are -1 when
// they do not identify one frame or row. Details can include fixture data;
// choose sanitized fixtures when saving test output in CI.
type Mismatch struct {
	RefID  string `json:"refId"`
	Frame  int    `json:"frame"`
	Field  string `json:"field,omitempty"`
	Row    int    `json:"row"`
	Kind   string `json:"kind"`
	Detail string `json:"detail"`
}

func (m Mismatch) String() string {
	where := fmt.Sprintf("query %q", m.RefID)
	if m.Frame >= 0 {
		where += fmt.Sprintf(" frame %d", m.Frame)
	}
	if m.Field != "" {
		where += fmt.Sprintf(" field %q", m.Field)
	}
	if m.Row >= 0 {
		where += fmt.Sprintf(" row %d", m.Row)
	}
	return where + " [" + m.Kind + "]: " + m.Detail
}

// Check compares actual SQL results with expectations. At least one expectation
// is required. Errors and notices on every returned query are checked, even if
// only a SQL expression's values have an expectation. This prevents a healthy
// result from masking a failed source query or output truncation.
func Check(result *Result, expected map[string]Expectation) []Mismatch {
	var bad []Mismatch
	add := func(ref, kind, detail string) { bad = append(bad, Mismatch{ref, -1, "", -1, kind, detail}) }
	if result == nil {
		add("", "execution", "no Grafana result")
		return bad
	}
	if result.GrafanaVersion == "" {
		add("", "execution", "result has no observed Grafana version")
	}
	if len(expected) == 0 {
		add("", "expectation", "at least one query expectation is required")
	}
	for _, ref := range slices.Sorted(maps.Keys(expected)) {
		if _, ok := result.Results[ref]; !ok {
			add(ref, "missing-query", "expected query has no result")
		}
	}
	for _, ref := range slices.Sorted(maps.Keys(result.Results)) {
		actual := result.Results[ref]
		want, hasExpectation := expected[ref]
		if hasExpectation && (want.Frames == nil) == (want.ErrorContains == "") {
			add(ref, "expectation", "set Frames or ErrorContains, but not both")
			continue
		}
		switch {
		case want.ErrorContains != "":
			if actual.Error == "" || !strings.Contains(actual.Error, want.ErrorContains) {
				add(ref, "query-error", fmt.Sprintf("want error containing %q, got %q", want.ErrorContains, actual.Error))
			}
		case actual.Error != "":
			add(ref, "query-error", actual.Error)
		case actual.Status >= 400:
			add(ref, "query-status", fmt.Sprintf("query returned status %d without an error message", actual.Status))
		case hasExpectation:
			bad = append(bad, compareFrames(ref, actual.Frames, want.Frames)...)
		}
		notices := frameNotices(actual.Frames)
		if !slices.Equal(notices, want.Notices) {
			add(ref, "notices", fmt.Sprintf("want %v, got %v", want.Notices, notices))
		}
	}
	return bad
}

// AssertResult fails tb for each mismatch, and returns the mismatches for your
// own reporting. It doesn't stop the test. An expected error shows only that
// the query failed as expected, not that SQL accepts valid input.
func AssertResult(tb testing.TB, result *Result, expected map[string]Expectation) []Mismatch {
	tb.Helper()
	bad := Check(result, expected)
	for _, m := range bad {
		tb.Errorf("SQL Expression runtime contract: %s", m)
	}
	return bad
}

func frameNotices(frames data.Frames) []data.Notice {
	var notices []data.Notice
	for _, frame := range frames {
		if frame != nil && frame.Meta != nil {
			notices = append(notices, frame.Meta.Notices...)
		}
	}
	return notices
}

func compareFrames(ref string, actual, expected data.Frames) []Mismatch {
	var bad []Mismatch
	if len(actual) != len(expected) {
		bad = append(bad, Mismatch{ref, -1, "", -1, "frame-count", fmt.Sprintf("want %d, got %d", len(expected), len(actual))})
	}
	for i := range min(len(actual), len(expected)) {
		bad = append(bad, compareFrame(ref, i, actual[i], expected[i])...)
	}
	return bad
}

func compareFrame(ref string, index int, actual, expected *data.Frame) []Mismatch {
	var bad []Mismatch
	add := func(field string, row int, kind, detail string) {
		bad = append(bad, Mismatch{ref, index, field, row, kind, detail})
	}
	if actual == nil || expected == nil {
		add("", -1, "frame", "actual and expected frames must be non-nil")
		return bad
	}
	if _, err := actual.RowLen(); err != nil {
		add("", -1, "frame", "actual frame is malformed")
		return bad
	}
	if _, err := expected.RowLen(); err != nil {
		add("", -1, "expectation", "expected frame is malformed")
		return bad
	}
	if len(actual.Fields) != len(expected.Fields) {
		add("", -1, "column-count", fmt.Sprintf("want %d, got %d", len(expected.Fields), len(actual.Fields)))
	}
	if actual.Rows() != expected.Rows() {
		add("", -1, "row-count", fmt.Sprintf("want %d, got %d", expected.Rows(), actual.Rows()))
	}
	for i := range min(len(actual.Fields), len(expected.Fields)) {
		a, e := actual.Fields[i], expected.Fields[i]
		if a.Name != e.Name {
			add(e.Name, -1, "column-name", fmt.Sprintf("column %d: want %q, got %q", i, e.Name, a.Name))
		}
		if a.Type() != e.Type() {
			add(e.Name, -1, "column-type", fmt.Sprintf("want %s, got %s", e.Type(), a.Type()))
		}
		if !maps.Equal(a.Labels, e.Labels) {
			add(e.Name, -1, "column-labels", fmt.Sprintf("want %v, got %v", e.Labels, a.Labels))
		}
		for row := range min(a.Len(), e.Len()) {
			av, aOK := a.ConcreteAt(row)
			ev, eOK := e.ConcreteAt(row)
			if aOK != eOK || aOK && !equalValue(av, ev) {
				add(e.Name, row, "value", fmt.Sprintf("want %v (present=%t), got %v (present=%t)", ev, eOK, av, aOK))
			}
		}
	}
	return bad
}

func equalValue(actual, expected any) bool {
	switch a := actual.(type) {
	case time.Time:
		e, ok := expected.(time.Time)
		return ok && a.Equal(e)
	case json.RawMessage:
		e, ok := expected.(json.RawMessage)
		return ok && equalJSON(a, e)
	case float64:
		if e, ok := expected.(float64); ok && math.IsNaN(a) && math.IsNaN(e) {
			return true
		}
	case float32:
		if e, ok := expected.(float32); ok && math.IsNaN(float64(a)) && math.IsNaN(float64(e)) {
			return true
		}
	}
	return reflect.DeepEqual(actual, expected)
}

func equalJSON(a, b json.RawMessage) bool {
	// UseNumber preserves integer JSON values above 2^53 during comparison.
	decode := func(raw json.RawMessage) (any, error) {
		if raw == nil {
			raw = json.RawMessage("null")
		}
		if !json.Valid(raw) {
			return nil, fmt.Errorf("invalid JSON")
		}
		var value any
		d := json.NewDecoder(bytes.NewReader(raw))
		d.UseNumber()
		err := d.Decode(&value)
		return value, err
	}
	av, ae := decode(a)
	bv, be := decode(b)
	return ae == nil && be == nil && reflect.DeepEqual(av, bv)
}
