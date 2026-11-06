package sqlexprtest

import (
	"fmt"
	"strings"
)

// Status distinguishes executed assertions from absent or excluded coverage.
type Status string

const (
	StatusPassed      Status = "passed"
	StatusFailed      Status = "failed"
	StatusUnsupported Status = "unsupported"
	StatusSkipped     Status = "skipped"
	StatusUntested    Status = "untested"
)

// SuiteReport holds the results in the order the suite lists them. Save it
// with encoding/json. SchemaVersion names the JSON format, which is
// experimental, so reject versions you don't know. The report holds
// predictions, not SQL results.
type SuiteReport struct {
	SchemaVersion int    `json:"schemaVersion"`
	Name          string `json:"name,omitempty"`
	Evidence      string `json:"evidence"`
	// MinGrafana is the minimum Grafana version the suite used. Empty means
	// none was set, so the checks assume no known fixes. JSON omits it then.
	MinGrafana       string           `json:"minGrafana,omitempty"`
	FailureThreshold string           `json:"failureThreshold"`
	Modes            []ModeResult     `json:"modes"`
	Scenarios        []ScenarioResult `json:"scenarios"`
}

// ModeResult summarizes the declared scenarios of a query mode.
type ModeResult struct {
	Name   string `json:"name"`
	Status Status `json:"status"`
	Reason string `json:"reason,omitempty"`
}

// ScenarioResult retains original findings and the allowances that matched them.
type ScenarioResult struct {
	Name            string        `json:"name"`
	Mode            string        `json:"mode"`
	Status          Status        `json:"status"`
	Reason          string        `json:"reason,omitempty"`
	ExpectedFailure bool          `json:"expectedFailure,omitempty"`
	Inputs          []InputResult `json:"inputs,omitempty"`
	Problems        []string      `json:"problems,omitempty"`
}

// InputResult associates a prediction and cell count with its request RefID.
type InputResult struct {
	RefID   string           `json:"refId"`
	Report  *Report          `json:"report"`
	Cells   int64            `json:"cells"`
	Allowed []AllowedFinding `json:"allowed,omitempty"`
}

// AllowedFinding identifies a retained finding and why it did not fail the test.
type AllowedFinding struct {
	FindingIndex   int    `json:"findingIndex"`
	AllowanceIndex int    `json:"allowanceIndex"`
	Reason         string `json:"reason"`
}

// Passed reports whether every listed mode and scenario passed. A skipped,
// unsupported or untested mode or scenario makes it false.
func (r *SuiteReport) Passed() bool {
	if len(r.Modes) == 0 {
		return false
	}
	for _, mode := range r.Modes {
		if mode.Status != StatusPassed {
			return false
		}
	}
	for _, scenario := range r.Scenarios {
		if scenario.Status != StatusPassed {
			return false
		}
	}
	return true
}

// HasFailures reports failed assertions or supported modes without a passing
// successful-input scenario. Skips are not failures when positive coverage
// exists for that mode. Explicitly unsupported modes are not failures.
func (r *SuiteReport) HasFailures() bool {
	for _, mode := range r.Modes {
		if mode.Status == StatusFailed || mode.Status == StatusUntested {
			return true
		}
	}
	for _, scenario := range r.Scenarios {
		if scenario.Status == StatusFailed || scenario.Status == StatusUntested {
			return true
		}
	}
	return false
}

func (r *SuiteReport) String() string {
	var out strings.Builder
	minimum := r.MinGrafana
	if minimum == "" {
		minimum = modeledMinimum(nil)
	}
	fmt.Fprintf(&out, "SQL Expressions suite %q: evidence %s, minimum %s, fail on %s\n", r.Name, r.Evidence, minimum, r.FailureThreshold)
	for _, mode := range r.Modes {
		fmt.Fprintf(&out, "mode %q: %s", mode.Name, mode.Status)
		if mode.Reason != "" {
			fmt.Fprintf(&out, " (%s)", mode.Reason)
		}
		out.WriteByte('\n')
	}
	for _, scenario := range r.Scenarios {
		fmt.Fprintf(&out, "scenario %q/%q: %s", scenario.Mode, scenario.Name, scenario.Status)
		if scenario.ExpectedFailure {
			out.WriteString(" (expected failure fixture)")
		}
		if scenario.Reason != "" {
			fmt.Fprintf(&out, " (%s)", scenario.Reason)
		}
		out.WriteByte('\n')
		for _, input := range scenario.Inputs {
			fmt.Fprintf(&out, "  RefID %q, input cells %d\n", input.RefID, input.Cells)
			if input.Report != nil {
				out.WriteString("    " + strings.ReplaceAll(input.Report.String(), "\n", "\n    ") + "\n")
			}
			for _, allowed := range input.Allowed {
				fmt.Fprintf(&out, "    allowed finding %d by allowance %d: %s\n", allowed.FindingIndex, allowed.AllowanceIndex, allowed.Reason)
			}
		}
		for _, problem := range scenario.Problems {
			fmt.Fprintf(&out, "  FAIL: %s\n", problem)
		}
	}
	return strings.TrimSuffix(out.String(), "\n")
}

func summarizeMode(mode QueryMode, scenarios []ScenarioResult) ModeResult {
	r := ModeResult{Name: mode.Name, Status: StatusUntested, Reason: "no successful-input scenario ran"}
	if mode.UnsupportedReason != "" {
		r.Status, r.Reason = StatusUnsupported, mode.UnsupportedReason
		return r
	}
	positive, skipped, failed := false, false, false
	for _, scenario := range scenarios {
		if scenario.Mode != mode.Name {
			continue
		}
		failed = failed || scenario.Status == StatusFailed
		skipped = skipped || scenario.Status == StatusSkipped
		positive = positive || scenario.Status == StatusPassed && !scenario.ExpectedFailure
	}
	switch {
	case failed:
		r.Status, r.Reason = StatusFailed, "one or more scenarios failed"
	case positive && skipped:
		r.Status, r.Reason = StatusSkipped, "one or more scenarios were skipped"
	case positive:
		r.Status, r.Reason = StatusPassed, ""
	}
	return r
}
