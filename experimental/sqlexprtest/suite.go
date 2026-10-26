package sqlexprtest

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"
)

// QueryMode is a query mode you list, such as table or range metrics. Set
// UnsupportedReason to mark it unsupported. An unsupported mode never counts
// as tested.
type QueryMode struct {
	Name              string
	UnsupportedReason string
}

// Scenario checks one named test case. Names must be unique within a query
// mode. Want lists every RefID that Input must report, and each contract must
// check at least one thing. SkipReason skips the scenario.
//
// ExpectedFailure marks a scenario that should fail, such as an upstream
// error. Its error findings still need allowances. It doesn't count as a test
// of the mode with valid input.
type Scenario struct {
	Name            string
	Mode            string
	Input           Input
	Want            map[string]Contract
	Allowances      []Allowance
	SkipReason      string
	ExpectedFailure bool
}

// Suite lists the query modes and scenarios you test. It covers only what
// you list, not every query the plugin can run.
type Suite struct {
	Name      string
	Modes     []QueryMode
	Scenarios []Scenario
}

// Evaluate runs the scenarios one at a time and checks their contracts. It
// checks the whole suite for mistakes before it runs any Input, and returns
// those mistakes as an error. Handler and Input failures go in the report.
// It accepts MinGrafana and FailOn. It rejects Ignore; use Allowances. It
// doesn't run SQL.
func Evaluate(ctx context.Context, suite Suite, opts ...Option) (*SuiteReport, error) {
	o := newOptions(opts)
	if err := validateSuite(suite, o); err != nil {
		return nil, err
	}
	result := &SuiteReport{SchemaVersion: 1, Name: suite.Name, Evidence: "predicted-sql-input", FailureThreshold: o.failOn.String()}
	if o.minGrafana != nil {
		result.MinGrafana = o.minGrafana.String()
	}
	for _, scenario := range suite.Scenarios {
		result.Scenarios = append(result.Scenarios, evaluateScenario(ctx, scenario, o, opts))
	}
	for _, mode := range suite.Modes {
		result.Modes = append(result.Modes, summarizeMode(mode, result.Scenarios))
	}
	return result, nil
}

// AssertSuite evaluates a suite and fails tb for failed or untested modes and
// scenarios. Each supported mode needs a passing successful-input scenario,
// even when other scenarios are skipped. Skips and unsupported modes remain
// incomplete: SuiteReport.Passed is false. Invalid configuration returns nil.
func AssertSuite(tb testing.TB, ctx context.Context, suite Suite, opts ...Option) *SuiteReport {
	tb.Helper()
	r, err := Evaluate(ctx, suite, opts...)
	if err != nil {
		tb.Errorf("SQL Expressions suite: %v", err)
		return nil
	}
	if r.HasFailures() {
		tb.Errorf("SQL Expressions suite failed:\n%s", r)
	} else {
		tb.Log(r.String())
	}
	return r
}

func validateSuite(s Suite, o options) error {
	if len(o.ignore) != 0 {
		return fmt.Errorf("suite: Ignore is not supported; use scoped Allowances with a reason")
	}
	if o.failOn < SeverityInfo || o.failOn > SeverityError {
		return fmt.Errorf("suite: invalid failure threshold %v", o.failOn)
	}
	if len(s.Modes) == 0 {
		return fmt.Errorf("suite: declare at least one query mode")
	}
	modes := make(map[string]QueryMode, len(s.Modes))
	for _, mode := range s.Modes {
		if strings.TrimSpace(mode.Name) == "" {
			return fmt.Errorf("suite: query mode needs a name")
		}
		if _, exists := modes[mode.Name]; exists {
			return fmt.Errorf("suite: duplicate query mode %q", mode.Name)
		}
		if mode.UnsupportedReason != "" && strings.TrimSpace(mode.UnsupportedReason) == "" {
			return fmt.Errorf("suite: query mode %q needs a nonblank unsupported reason", mode.Name)
		}
		modes[mode.Name] = mode
	}
	seen := make(map[string]bool, len(s.Scenarios))
	for _, scenario := range s.Scenarios {
		if err := validateScenario(scenario, modes, seen); err != nil {
			return fmt.Errorf("scenario %q/%q: %w", scenario.Mode, scenario.Name, err)
		}
	}
	return nil
}

func validateScenario(s Scenario, modes map[string]QueryMode, seen map[string]bool) error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("name is required")
	}
	mode, exists := modes[s.Mode]
	if !exists {
		return fmt.Errorf("undeclared query mode")
	}
	if mode.UnsupportedReason != "" {
		return fmt.Errorf("unsupported query modes cannot contain scenarios")
	}
	key := s.Mode + "\x00" + s.Name
	if seen[key] {
		return fmt.Errorf("duplicate scenario name within query mode")
	}
	seen[key] = true
	if s.SkipReason != "" {
		if strings.TrimSpace(s.SkipReason) == "" {
			return fmt.Errorf("skip reason must not be blank")
		}
		return nil
	}
	if s.Input == nil || len(s.Want) == 0 {
		return fmt.Errorf("executable scenarios require Input and explicit RefID contracts")
	}
	for _, refID := range slices.Sorted(maps.Keys(s.Want)) {
		if strings.TrimSpace(refID) == "" {
			return fmt.Errorf("contract RefID must not be blank")
		}
		if err := validateContract(s.Want[refID]); err != nil {
			return fmt.Errorf("contract %q: %w", refID, err)
		}
	}
	return validateAllowances(s.Allowances, s.Want)
}

func evaluateScenario(ctx context.Context, s Scenario, o options, opts []Option) ScenarioResult {
	r := ScenarioResult{Name: s.Name, Mode: s.Mode, Status: StatusPassed, ExpectedFailure: s.ExpectedFailure}
	if s.SkipReason != "" {
		r.Status, r.Reason = StatusSkipped, s.SkipReason
		return r
	}
	reports, err := invokeInput(ctx, s.Input, opts)
	if err != nil {
		r.Status = StatusFailed
		r.Problems = append(r.Problems, err.Error())
		return r
	}
	if len(reports) == 0 {
		r.Problems = append(r.Problems, "input returned no query reports")
	}
	for _, refID := range slices.Sorted(maps.Keys(s.Want)) {
		if reports[refID] == nil {
			r.Problems = append(r.Problems, fmt.Sprintf("missing report for requested RefID %q", refID))
		}
	}
	used := make([]bool, len(s.Allowances))
	hadError := false
	for _, refID := range slices.Sorted(maps.Keys(reports)) {
		input, problems := evaluateReport(refID, reports[refID], s, o, used)
		r.Inputs = append(r.Inputs, input)
		r.Problems = append(r.Problems, problems...)
		if input.Report != nil && input.Report.Worst() == SeverityError {
			hadError = true
		}
	}
	if s.ExpectedFailure && !hadError {
		r.Problems = append(r.Problems, "expected failure fixture produced no error finding")
	}
	for i, matched := range used {
		if !matched {
			r.Problems = append(r.Problems, fmt.Sprintf("unused allowance %d for %s: %s", i, s.Allowances[i].Code, s.Allowances[i].Reason))
		}
	}
	if len(r.Problems) > 0 {
		r.Status = StatusFailed
	}
	return r
}

func invokeInput(ctx context.Context, input Input, opts []Option) (reports map[string]*Report, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("scenario input panicked: %v", p)
		}
	}()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return input(ctx, opts...)
}

func modeledMinimum(v *version) string {
	if v == nil {
		return "none"
	}
	return v.String()
}

func evaluateReport(refID string, report *Report, s Scenario, o options, used []bool) (InputResult, []string) {
	input := InputResult{RefID: refID, Report: report}
	if report == nil {
		return input, []string{fmt.Sprintf("RefID %q has a nil report", refID)}
	}
	var problems []string
	if report.RefID != refID {
		problems = append(problems, fmt.Sprintf("report key %q disagrees with report RefID %q", refID, report.RefID))
	}
	if modeledMinimum(report.minGrafana) != modeledMinimum(o.minGrafana) {
		problems = append(problems, fmt.Sprintf("RefID %q: the report's minimum Grafana is %s, but the suite's is %s; pass the options Input receives to Check, CheckResponse or CheckQueryData", refID, modeledMinimum(report.minGrafana), modeledMinimum(o.minGrafana)))
	}
	switch report.Mode {
	case ModeNone, ModeTable, ModeFullLong:
	default:
		problems = append(problems, fmt.Sprintf("RefID %q: report has invalid conversion mode %q; return a report produced by Check, CheckResponse, or CheckQueryData", refID, report.Mode))
	}
	input.Cells = report.Cells()
	if report.Mode == ModeNone && !s.ExpectedFailure {
		problems = append(problems, fmt.Sprintf("RefID %q: no SQL input was constructed for this successful-input scenario; correct the response", refID))
	}
	if contract, exists := s.Want[refID]; exists {
		for _, problem := range compareContract(report, contract) {
			problems = append(problems, fmt.Sprintf("RefID %q: %s", refID, problem))
		}
	} else {
		problems = append(problems, fmt.Sprintf("unexpected report for RefID %q; declare its contract", refID))
	}
	for i, finding := range report.Findings {
		if finding.Severity == SeverityError && !s.ExpectedFailure {
			problems = append(problems, fmt.Sprintf("RefID %q: SQL input failed: %s", refID, finding))
			continue
		}
		if finding.Severity < o.failOn {
			continue
		}
		index, allowed := matchAllowance(refID, finding, s.Allowances, used)
		if !allowed {
			problems = append(problems, fmt.Sprintf("RefID %q: %s", refID, finding))
		} else {
			input.Allowed = append(input.Allowed, AllowedFinding{FindingIndex: i, AllowanceIndex: index, Reason: s.Allowances[index].Reason})
		}
	}
	return input, problems
}
