package sqlexprtest

import (
	"maps"
	"slices"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type options struct {
	failOn Severity
	ignore map[Code]bool
}

// Option changes what the Assert functions treat as a failure.
type Option func(*options)

// FailOn fails the test on findings at or above s. The default is
// SeverityWarning, so lost data fails the test, and advice does not.
func FailOn(s Severity) Option {
	return func(o *options) { o.failOn = s }
}

// Ignore skips findings with these codes, for example a known Grafana
// limitation the plugin accepts.
func Ignore(codes ...Code) Option {
	return func(o *options) {
		for _, c := range codes {
			o.ignore[c] = true
		}
	}
}

func newOptions(opts []Option) options {
	o := options{failOn: SeverityWarning, ignore: map[Code]bool{}}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// AssertFrames checks one query's frames with [Check]. When a finding is at or
// above the FailOn severity, it marks the test as failed, without stopping it,
// and prints the whole report. It returns the report.
func AssertFrames(tb testing.TB, frames data.Frames, opts ...Option) *Report {
	tb.Helper()
	r := Check(frames)
	assertReport(tb, r, newOptions(opts))
	return r
}

// AssertResponse checks a QueryDataResponse with [CheckResponse], and marks
// the test as failed for each report with a finding at or above the FailOn
// severity. Pass refIDs to also fail on queries missing from the response.
func AssertResponse(tb testing.TB, resp *backend.QueryDataResponse, refIDs []string, opts ...Option) map[string]*Report {
	tb.Helper()
	o := newOptions(opts)
	reports := CheckResponse(resp, refIDs...)
	for _, refID := range slices.Sorted(maps.Keys(reports)) {
		assertReport(tb, reports[refID], o)
	}
	return reports
}

func assertReport(tb testing.TB, r *Report, o options) {
	tb.Helper()
	failed := false
	for _, f := range r.Findings {
		if !o.ignore[f.Code] && f.Severity >= o.failOn {
			failed = true
			break
		}
	}
	if failed {
		tb.Errorf("SQL Expressions input check failed:\n%s", r)
	}
}
