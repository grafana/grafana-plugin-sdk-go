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

// AssertOption changes what the Assert functions treat as a failure, with
// [FailOn] and [Ignore]. Every [CheckOption] is also an AssertOption.
type AssertOption interface {
	apply(*options)
}

// CheckOption changes what the checks predict, with [MinGrafana]. The Check
// functions take only CheckOption values, so passing [FailOn] or [Ignore] to
// them doesn't compile.
type CheckOption interface {
	AssertOption
	checkOption()
}

type assertOption func(*options)

func (f assertOption) apply(o *options) { f(o) }

type checkOption func(*options)

func (f checkOption) apply(o *options) { f(o) }
func (checkOption) checkOption()       {}

// FailOn fails the test on findings at or above s. The default is
// SeverityWarning, so lost data fails the test, and advice does not.
func FailOn(s Severity) AssertOption {
	return assertOption(func(o *options) { o.failOn = s })
}

// Ignore skips findings with these codes, for example a known Grafana
// limitation the plugin accepts.
func Ignore(codes ...Code) AssertOption {
	return assertOption(func(o *options) {
		for _, c := range codes {
			o.ignore[c] = true
		}
	})
}

func newOptions[T AssertOption](opts []T) options {
	o := options{failOn: SeverityWarning, ignore: map[Code]bool{}}
	for _, opt := range opts {
		opt.apply(&o)
	}
	return o
}

// checkOptions keeps the options that change what the checks predict.
func checkOptions(opts []AssertOption) []CheckOption {
	var out []CheckOption
	for _, opt := range opts {
		if c, ok := opt.(CheckOption); ok {
			out = append(out, c)
		}
	}
	return out
}

// AssertFrames checks one query's frames with [Check]. When a finding is at or
// above the FailOn severity, it marks the test as failed, without stopping it,
// and prints the whole report. It returns the report.
func AssertFrames(tb testing.TB, frames data.Frames, opts ...AssertOption) *Report {
	tb.Helper()
	r := Check(frames, checkOptions(opts)...)
	assertReport(tb, r, newOptions(opts))
	return r
}

// AssertResponse checks a QueryDataResponse with [CheckResponse], and marks
// the test as failed for each report with a finding at or above the FailOn
// severity. Pass refIDs to also fail on queries missing from the response.
// It fails when there is nothing to check: no refIDs and an empty response.
func AssertResponse(tb testing.TB, resp *backend.QueryDataResponse, refIDs []string, opts ...AssertOption) map[string]*Report {
	tb.Helper()
	o := newOptions(opts)
	reports := CheckResponse(resp, refIDs, checkOptions(opts)...)
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
func AssertQueryData(tb testing.TB, ctx context.Context, h backend.QueryDataHandler, req *backend.QueryDataRequest, opts ...AssertOption) map[string]*Report {
	tb.Helper()
	reports, err := CheckQueryData(ctx, h, req, checkOptions(opts)...)
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
