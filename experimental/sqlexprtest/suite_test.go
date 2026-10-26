package sqlexprtest

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func positiveScenario() Scenario {
	return Scenario{Name: "populated", Mode: "table", Input: FramesInput("A", data.Frames{tableFrame()}), Want: map[string]Contract{
		"A": {Mode: ModeTable, Rows: new(1), Columns: []Column{{Name: "value", Type: data.FieldTypeFloat64, SQLType: "DOUBLE"}}},
	}}
}

func oneScenario(s Scenario) Suite {
	return Suite{Name: "plugin", Modes: []QueryMode{{Name: "table"}}, Scenarios: []Scenario{s}}
}

func TestSuiteContracts(t *testing.T) {
	for _, tc := range []struct {
		name string
		want Contract
		fail string
	}{
		{"exact", positiveScenario().Want["A"], ""},
		{"mode mismatch", Contract{Mode: ModeFullLong}, "conversion mode"},
		{"row mismatch", Contract{Rows: new(0)}, "rows: want 0, got 1"},
		{"empty schema is asserted", Contract{Columns: []Column{}}, "ordered columns"},
		{"field type matters", Contract{Columns: []Column{{Name: "value", Type: data.FieldTypeInt64, SQLType: "DOUBLE"}}}, "ordered columns"},
		{"SQL type matters", Contract{Columns: []Column{{Name: "value", Type: data.FieldTypeFloat64, SQLType: "BIGINT"}}}, "ordered columns"},
		{"cell boundary", Contract{MaxCells: 1}, ""},
		{"negative cell limit is unlimited", Contract{Mode: ModeTable, MaxCells: -1}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := positiveScenario()
			s.Want["A"] = tc.want
			r, err := Evaluate(t.Context(), oneScenario(s), MinGrafana("13.3.0-pre"))
			require.NoError(t, err)
			assert.Equal(t, "13.3.0-pre", r.MinGrafana)
			assert.Equal(t, "predicted-sql-input", r.Evidence)
			assert.Equal(t, int64(1), r.Scenarios[0].Inputs[0].Cells)
			if tc.fail == "" {
				assert.True(t, r.Passed(), r.String())
				assert.False(t, r.HasFailures())
			} else {
				assert.False(t, r.Passed())
				assert.True(t, r.HasFailures())
				assert.Contains(t, r.String(), tc.fail)
			}
		})
	}
	t.Run("names and order matter", func(t *testing.T) {
		s := positiveScenario()
		s.Input = FramesInput("A", data.Frames{typed(data.FrameTypeTable, data.NewFrame("", data.NewField("a", nil, []int64{1}), data.NewField("b", nil, []int64{2})))})
		s.Want["A"] = Contract{Columns: []Column{{Name: "b", Type: data.FieldTypeInt64, SQLType: "BIGINT"}, {Name: "a", Type: data.FieldTypeInt64, SQLType: "BIGINT"}}, MaxCells: 1}
		r, err := Evaluate(t.Context(), oneScenario(s))
		require.NoError(t, err)
		assert.Contains(t, r.String(), "ordered columns")
		assert.Contains(t, r.String(), "input cells: 2 exceeds")
	})
}

func TestSuiteInventoryDoesNotClaimSkippedOrNegativeCoverage(t *testing.T) {
	s := Suite{Modes: []QueryMode{{Name: "table"}, {Name: "metrics"}, {Name: "browser", UnsupportedReason: "frames are constructed only in the browser"}, {Name: "logs"}}, Scenarios: []Scenario{
		positiveScenario(), {Name: "capture", Mode: "logs", SkipReason: "fixture capture pending"},
	}}
	r, err := Evaluate(t.Context(), s)
	require.NoError(t, err)
	assert.Equal(t, []Status{StatusPassed, StatusUntested, StatusUnsupported, StatusUntested}, []Status{r.Modes[0].Status, r.Modes[1].Status, r.Modes[2].Status, r.Modes[3].Status})
	assert.False(t, r.Passed())
	assert.True(t, r.HasFailures(), "undeclared fixture coverage fails")
	s.Modes = append(s.Modes[:1], s.Modes[2:]...)
	logs := positiveScenario()
	logs.Mode = "logs"
	s.Scenarios = append(s.Scenarios, logs)
	r, err = Evaluate(t.Context(), s)
	require.NoError(t, err)
	assert.False(t, r.Passed(), "explicit exclusions still are not passes")
	assert.False(t, r.HasFailures(), "optional skips do not fail a mode with positive coverage")
	assert.Equal(t, StatusSkipped, r.Modes[2].Status)
	negative := Scenario{Name: "upstream error", Mode: "table", ExpectedFailure: true,
		Input: ResponseInput(&backend.QueryDataResponse{Responses: backend.Responses{"A": {Error: errors.New("upstream unavailable")}}}, []string{"A"}),
		Want:  map[string]Contract{"A": {Mode: ModeNone}}, Allowances: []Allowance{{Code: CodeQueryError, RefID: "A", Reason: "intentional upstream failure"}},
	}
	r, err = Evaluate(t.Context(), oneScenario(negative))
	require.NoError(t, err)
	assert.Equal(t, StatusPassed, r.Scenarios[0].Status)
	assert.Equal(t, StatusUntested, r.Modes[0].Status)
	assert.Contains(t, r.String(), "expected failure fixture")
	s = oneScenario(positiveScenario())
	s.Scenarios = append(s.Scenarios, negative)
	r, err = Evaluate(t.Context(), s)
	require.NoError(t, err)
	assert.True(t, r.Passed(), r.String())
	negative.ExpectedFailure = false
	r, err = Evaluate(t.Context(), oneScenario(negative))
	require.NoError(t, err)
	assert.True(t, r.HasFailures())
	assert.Contains(t, r.String(), "SQL input failed:")
	positive := positiveScenario()
	positive.ExpectedFailure = true
	r, err = Evaluate(t.Context(), oneScenario(positive))
	require.NoError(t, err)
	assert.Contains(t, r.String(), "produced no error finding")
}

func TestSuiteCannotSkipAllPositiveCoverage(t *testing.T) {
	for _, withNegative := range []bool{false, true} {
		t.Run(strconv.FormatBool(withNegative), func(t *testing.T) {
			suite := oneScenario(Scenario{Name: "capture", Mode: "table", SkipReason: "fixture capture pending"})
			if withNegative {
				suite.Scenarios = append(suite.Scenarios, Scenario{
					Name: "upstream error", Mode: "table", ExpectedFailure: true,
					Input:      ResponseInput(&backend.QueryDataResponse{Responses: backend.Responses{"A": {Error: errors.New("upstream unavailable")}}}, []string{"A"}),
					Want:       map[string]Contract{"A": {Mode: ModeNone}},
					Allowances: []Allowance{{Code: CodeQueryError, RefID: "A", Reason: "intentional upstream failure"}},
				})
			}
			rec := &recorder{TB: t}
			r := AssertSuite(rec, t.Context(), suite)
			require.NotNil(t, r)
			assert.Equal(t, StatusUntested, r.Modes[0].Status)
			assert.Equal(t, StatusSkipped, r.Scenarios[0].Status)
			assert.True(t, r.HasFailures())
			assert.False(t, r.Passed())
			require.Len(t, rec.errors, 1)
			assert.Contains(t, rec.errors[0], "no successful-input scenario ran")
		})
	}
}

func TestSuiteScopedAllowances(t *testing.T) {
	input := func(secondName string) Input {
		return FramesInput("A", data.Frames{typed(data.FrameTypeTimeSeriesWide, data.NewFrame("", data.NewField("t", nil, []time.Time{t0}), data.NewField("status", nil, []string{"ok"}), data.NewField(secondName, nil, []string{"x"}), data.NewField("cpu", nil, []float64{1})))})
	}
	s := positiveScenario()
	s.Input = input("new_field")
	s.Want["A"] = Contract{Mode: ModeFullLong, Rows: new(1)}
	s.Allowances = []Allowance{{Code: CodeDroppedField, RefID: "A", Frame: new(0), Field: new("status"), Reason: "status is not part of this mode's SQL contract"}}
	r, err := Evaluate(t.Context(), oneScenario(s))
	require.NoError(t, err)
	assert.True(t, r.HasFailures(), "allowing status does not hide new_field")
	require.Len(t, r.Scenarios[0].Inputs[0].Report.Findings, 2)
	require.Len(t, r.Scenarios[0].Inputs[0].Allowed, 1)
	assert.Contains(t, strings.Join(r.Scenarios[0].Problems, "\n"), "new_field")
	s.Allowances = append(s.Allowances, Allowance{Code: CodeDroppedField, RefID: "A", Frame: new(0), Field: new("new_field"), Reason: "deliberately not exposed to SQL"})
	r, err = Evaluate(t.Context(), oneScenario(s))
	require.NoError(t, err)
	assert.True(t, r.Passed(), r.String())
	s.Allowances = append(s.Allowances, s.Allowances[0])
	r, err = Evaluate(t.Context(), oneScenario(s))
	require.NoError(t, err)
	assert.Contains(t, r.String(), "unused allowance 2")
	s.Allowances = s.Allowances[:1]
	s.Allowances[0].Frame = new(1)
	r, err = Evaluate(t.Context(), oneScenario(s))
	require.NoError(t, err)
	assert.Contains(t, r.String(), "unused allowance 0")
	assert.Empty(t, r.Scenarios[0].Inputs[0].Allowed)
}

func TestSuiteAllowanceDoesNotCrossQueryBoundary(t *testing.T) {
	s := positiveScenario()
	s.Input = ResponseInput(&backend.QueryDataResponse{Responses: backend.Responses{
		"A": {Frames: droppedFieldFrames()}, "B": {Frames: droppedFieldFrames()},
	}}, []string{"A", "B"})
	s.Want = map[string]Contract{"A": {Mode: ModeFullLong}, "B": {Mode: ModeFullLong}}
	s.Allowances = []Allowance{{Code: CodeDroppedField, RefID: "A", Field: new("state"), Reason: "A intentionally drops the status"}}
	r, err := Evaluate(t.Context(), oneScenario(s))
	require.NoError(t, err)
	assert.True(t, r.HasFailures())
	assert.Len(t, r.Scenarios[0].Inputs[0].Allowed, 1)
	assert.Empty(t, r.Scenarios[0].Inputs[1].Allowed)
	assert.Contains(t, strings.Join(r.Scenarios[0].Problems, "\n"), `RefID "B"`)
}

func TestSuiteExtraResponseMatchesAcrossInputs(t *testing.T) {
	both := &backend.QueryDataResponse{Responses: backend.Responses{
		"A": {Frames: data.Frames{tableFrame()}}, "B": {Frames: data.Frames{tableFrame()}},
	}}
	h := handlerFunc(func(context.Context, *backend.QueryDataRequest) (*backend.QueryDataResponse, error) { return both, nil })
	allow := []Allowance{{Code: CodeUnexpectedRefID, Reason: "the fixture also returns B"}}
	for _, tc := range []struct {
		name  string
		input Input
		want  map[string]Contract
	}{
		{"query data", QueryDataInput(h, request("A")), nil},
		{"response", ResponseInput(both, []string{"A"}), nil},
		{"response with contract for B", ResponseInput(both, []string{"A"}), map[string]Contract{"B": {Mode: ModeNone}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := positiveScenario()
			s.Input = tc.input
			maps.Copy(s.Want, tc.want)
			r, err := Evaluate(t.Context(), oneScenario(s))
			require.NoError(t, err)
			assert.True(t, r.HasFailures(), "the finding needs an allowance")
			s.Allowances = allow
			r, err = Evaluate(t.Context(), oneScenario(s))
			require.NoError(t, err)
			assert.True(t, r.Passed(), r.String())
		})
	}
	t.Run("extra responses alone are not coverage", func(t *testing.T) {
		s := positiveScenario()
		s.Input = func(_ context.Context, opts ...Option) (map[string]*Report, error) {
			reports := CheckResponse(both, []string{"C"}, opts...)
			delete(reports, "C")
			return reports, nil
		}
		s.Want = map[string]Contract{"A": {Mode: ModeNone}, "B": {Mode: ModeNone}}
		s.Allowances = allow
		r, err := Evaluate(t.Context(), oneScenario(s))
		require.NoError(t, err)
		assert.True(t, r.HasFailures())
		assert.Contains(t, r.String(), "only reports for RefIDs that no query asked for")
		assert.Equal(t, StatusFailed, r.Modes[0].Status)
	})
}

func TestSuiteInputFailuresCannotPass(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input Input
		want  string
	}{
		{"empty", func(context.Context, ...Option) (map[string]*Report, error) { return nil, nil }, "input returned no query reports"},
		{"nil report", func(context.Context, ...Option) (map[string]*Report, error) { return map[string]*Report{"A": nil}, nil }, "nil report"},
		{"error", func(context.Context, ...Option) (map[string]*Report, error) { return nil, errors.New("fixture failed") }, "fixture failed"},
		{"panic", func(context.Context, ...Option) (map[string]*Report, error) { panic("fixture panicked") }, "scenario input panicked"},
		{"wrong key", FramesInput("B", data.Frames{tableFrame()}), "missing report for requested RefID"},
		{"extra report", ResponseInput(&backend.QueryDataResponse{Responses: backend.Responses{"A": {Frames: data.Frames{tableFrame()}}, "B": {Frames: data.Frames{tableFrame()}}}}, nil), "unexpected report for RefID"},
		{"inconsistent key", func(context.Context, ...Option) (map[string]*Report, error) {
			return map[string]*Report{"A": {RefID: "B", Mode: ModeTable, Rows: 1}}, nil
		}, "disagrees with report RefID"},
		{"no table without diagnostic", func(context.Context, ...Option) (map[string]*Report, error) {
			return map[string]*Report{"A": {RefID: "A", Mode: ModeNone}}, nil
		}, "no SQL input was constructed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := positiveScenario()
			s.Input = tc.input
			r, err := Evaluate(t.Context(), oneScenario(s))
			require.NoError(t, err)
			assert.True(t, r.HasFailures())
			assert.Contains(t, r.String(), tc.want)
		})
	}
	t.Run("cancellation prevents invocation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		s := positiveScenario()
		s.Input = func(context.Context, ...Option) (map[string]*Report, error) { t.Fatal("must not run"); return nil, nil }
		r, err := Evaluate(ctx, oneScenario(s))
		require.NoError(t, err)
		assert.Contains(t, r.String(), "context canceled")
	})
}

func TestSuiteConfigurationValidatedBeforeInputs(t *testing.T) {
	for _, mutate := range []struct {
		name string
		fn   func(*Suite)
	}{
		{"empty inventory", func(s *Suite) { s.Modes = nil }},
		{"duplicate mode", func(s *Suite) { s.Modes = append(s.Modes, s.Modes[0]) }},
		{"duplicate scenario", func(s *Suite) { s.Scenarios = append(s.Scenarios, s.Scenarios[0]) }},
		{"unknown mode", func(s *Suite) { s.Scenarios[1].Mode = "missing" }},
		{"no input", func(s *Suite) { s.Scenarios[1].Input = nil }},
		{"no expectations", func(s *Suite) { s.Scenarios[1].Want = nil }},
		{"vacuous contract", func(s *Suite) { s.Scenarios[1].Want = map[string]Contract{"A": {}} }},
		{"negative rows", func(s *Suite) { s.Scenarios[1].Want = map[string]Contract{"A": {Rows: new(-1)}} }},
		{"unknown conversion", func(s *Suite) { s.Scenarios[1].Want = map[string]Contract{"A": {Mode: "magic"}} }},
		{"unknown code", func(s *Suite) { s.Scenarios[1].Allowances = []Allowance{{Code: "typo", Reason: "known issue"}} }},
		{"no reason", func(s *Suite) { s.Scenarios[1].Allowances = []Allowance{{Code: CodeMissingType}} }},
		{"wrong RefID", func(s *Suite) {
			s.Scenarios[1].Allowances = []Allowance{{Code: CodeMissingType, RefID: "B", Reason: "known issue"}}
		}},
		{"invalid frame", func(s *Suite) {
			s.Scenarios[1].Allowances = []Allowance{{Code: CodeMissingType, Frame: new(-2), Reason: "known issue"}}
		}},
		{"blank skip reason", func(s *Suite) { s.Scenarios[1].SkipReason = " " }},
		{"unsupported with scenarios", func(s *Suite) { s.Modes[0].UnsupportedReason = "browser only" }},
	} {
		t.Run(mutate.name, func(t *testing.T) {
			calls := 0
			s := oneScenario(positiveScenario())
			s.Scenarios[0].Input = func(context.Context, ...Option) (map[string]*Report, error) { calls++; return nil, nil }
			later := positiveScenario()
			later.Name = "later"
			s.Scenarios = append(s.Scenarios, later)
			mutate.fn(&s)
			r, err := Evaluate(t.Context(), s)
			require.Error(t, err)
			assert.Nil(t, r)
			assert.Zero(t, calls)
		})
	}
	_, err := Evaluate(t.Context(), oneScenario(positiveScenario()), Ignore(CodeEnumField))
	require.ErrorContains(t, err, "Ignore is not supported")
	_, err = Evaluate(t.Context(), oneScenario(positiveScenario()), FailOn(SeverityNone))
	require.ErrorContains(t, err, "invalid failure threshold")
}

func TestSuiteHandlerAndAssertion(t *testing.T) {
	queries := 0
	h := handlerFunc(func(_ context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
		queries++
		require.Len(t, req.Queries, 1)
		return &backend.QueryDataResponse{Responses: backend.Responses{req.Queries[0].RefID: {Frames: data.Frames{tableFrame()}}}}, nil
	})
	s := positiveScenario()
	s.Input = QueryDataInput(h, request("A"))
	rec := &recorder{TB: t}
	r := AssertSuite(rec, t.Context(), oneScenario(s))
	require.NotNil(t, r)
	assert.Empty(t, rec.errors)
	assert.True(t, r.Passed())
	assert.Equal(t, 1, queries)
	s.Want["A"] = Contract{Rows: new(2)}
	AssertSuite(rec, t.Context(), oneScenario(s))
	assert.Len(t, rec.errors, 1)
	r = AssertSuite(rec, t.Context(), Suite{})
	assert.Nil(t, r)
	assert.Len(t, rec.errors, 2)
}

func TestSuiteFailureThreshold(t *testing.T) {
	s := positiveScenario()
	s.Input = FramesInput("A", data.Frames{data.NewFrame("", data.NewField("value", nil, []float64{1}))})
	r, err := Evaluate(t.Context(), oneScenario(s))
	require.NoError(t, err)
	assert.True(t, r.Passed())
	require.True(t, r.Scenarios[0].Inputs[0].Report.Has(CodeMissingType))
	r, err = Evaluate(t.Context(), oneScenario(s), FailOn(SeverityInfo))
	require.NoError(t, err)
	assert.True(t, r.HasFailures())
	s.Allowances = []Allowance{{Code: CodeMissingType, RefID: "A", Reason: "metadata is intentionally unspecified"}}
	r, err = Evaluate(t.Context(), oneScenario(s), FailOn(SeverityInfo))
	require.NoError(t, err)
	assert.True(t, r.Passed(), r.String())
	r, err = Evaluate(t.Context(), oneScenario(s))
	require.NoError(t, err)
	assert.Contains(t, r.String(), "unused allowance 0", "findings below the threshold no longer need an allowance")
	s.Allowances = nil
	s.Input = FramesInput("A", droppedFieldFrames())
	s.Want["A"] = Contract{Mode: ModeFullLong}
	r, err = Evaluate(t.Context(), oneScenario(s), FailOn(SeverityError))
	require.NoError(t, err)
	assert.True(t, r.Passed(), r.String())
	require.True(t, r.Scenarios[0].Inputs[0].Report.Has(CodeDroppedField))
}

func TestSuiteDeterministicJSONAndText(t *testing.T) {
	s := positiveScenario()
	s.Want["B"] = s.Want["A"]
	s.Input = ResponseInput(&backend.QueryDataResponse{Responses: backend.Responses{"B": {Frames: data.Frames{tableFrame()}}, "A": {Frames: data.Frames{tableFrame()}}}}, []string{"B", "A"})
	a, err := Evaluate(t.Context(), oneScenario(s))
	require.NoError(t, err)
	b, err := Evaluate(t.Context(), oneScenario(s))
	require.NoError(t, err)
	aj, err := json.Marshal(a)
	require.NoError(t, err)
	bj, err := json.Marshal(b)
	require.NoError(t, err)
	assert.Equal(t, string(aj), string(bj))
	assert.Equal(t, a.String(), b.String())
	assert.Equal(t, "A", a.Scenarios[0].Inputs[0].RefID)
	assert.Contains(t, string(aj), `"evidence":"predicted-sql-input"`)
	assert.Equal(t, 1, a.SchemaVersion)
	assert.Contains(t, string(aj), `"schemaVersion":1`)
	assert.Empty(t, a.MinGrafana)
	assert.NotContains(t, string(aj), `"minGrafana"`)
	assert.Contains(t, a.String(), "minimum none")
}

func TestReportCells(t *testing.T) {
	frame := typed(data.FrameTypeTimeSeriesWide, data.NewFrame("", data.NewField("time", nil, make([]time.Time, 1000))))
	for range 50 {
		frame.Fields = append(frame.Fields, data.NewField("metric", nil, make([]float64, 1000)))
	}
	report := Check(data.Frames{frame})
	require.Empty(t, report.Findings)
	assert.Equal(t, 50000, report.Rows)
	assert.Equal(t, int64(150000), report.Cells())
	assert.Zero(t, (&Report{Mode: ModeNone, Rows: 5, Columns: make([]Column, 2)}).Cells())
	assert.Zero(t, (&Report{Mode: ModeTable, Rows: 0, Columns: make([]Column, 2)}).Cells())
	if strconv.IntSize == 64 {
		assert.Equal(t, int64(math.MaxInt64), (&Report{Mode: ModeTable, Rows: math.MaxInt, Columns: make([]Column, 2)}).Cells())
		assert.Contains(t, compareContract(&Report{Mode: ModeTable, Rows: math.MaxInt, Columns: make([]Column, 2)}, Contract{Mode: ModeTable}), "predicted cell count overflows int64")
	}
}
