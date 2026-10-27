package sqlexprtest

import (
	"context"
	"maps"
	"slices"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type options struct {
	failOn     Severity
	ignore     map[Code]bool
	minGrafana *version
}

// Option changes what the checks predict, with [MinGrafana], or what the
// Assert functions treat as a failure, with [FailOn] and [Ignore].
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
	r := Check(frames, opts...)
	assertReport(tb, r, newOptions(opts))
	return r
}

// AssertResponse checks a QueryDataResponse with [CheckResponse], and marks
// the test as failed for each report with a finding at or above the FailOn
// severity. Pass refIDs to also fail on queries missing from the response.
// It fails when there is nothing to check: no refIDs and an empty response.
func AssertResponse(tb testing.TB, resp *backend.QueryDataResponse, refIDs []string, opts ...Option) map[string]*Report {
	tb.Helper()
	o := newOptions(opts)
	reports := CheckResponse(resp, refIDs, opts...)
	if len(reports) == 0 {
		tb.Errorf("SQL Expressions input check: the response has no queries to check. Pass the RefIDs the request asked for.")
		return reports
	}
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

// AssertQueryData calls the handler with [CheckQueryData], and marks the test
// as failed for each query with a finding at or above the FailOn severity.
func AssertQueryData(tb testing.TB, ctx context.Context, h backend.QueryDataHandler, req *backend.QueryDataRequest, opts ...Option) map[string]*Report {
	tb.Helper()
	reports, err := CheckQueryData(ctx, h, req, opts...)
	if err != nil {
		tb.Errorf("SQL Expressions input check: %v", err)
		return nil
	}
	o := newOptions(opts)
	for _, refID := range slices.Sorted(maps.Keys(reports)) {
		assertReport(tb, reports[refID], o)
	}
	return reports
}
