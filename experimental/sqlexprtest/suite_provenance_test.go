package sqlexprtest

import (
	"context"
	"errors"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuiteRequiresInputModeledMinimum(t *testing.T) {
	for _, tc := range []struct {
		name       string
		suiteOpts  []Option
		inputOpts  []Option
		wantPassed bool
	}{
		{"ignores supplied minimum", []Option{MinGrafana("13.3.0")}, nil, false},
		{"overrides supplied minimum", []Option{MinGrafana("13.3.0")}, []Option{MinGrafana("13.2.0")}, false},
		{"overrides default baseline", nil, []Option{MinGrafana("13.3.0")}, false},
		{"matching minimum", []Option{MinGrafana("13.3.0")}, []Option{MinGrafana("13.3.0")}, true},
		{"matching default", nil, nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := positiveScenario()
			s.Input = func(context.Context, ...Option) (map[string]*Report, error) {
				r := Check(data.Frames{tableFrame()}, tc.inputOpts...)
				r.RefID = "A"
				return map[string]*Report{"A": r}, nil
			}
			r, err := Evaluate(t.Context(), oneScenario(s), tc.suiteOpts...)
			require.NoError(t, err)
			assert.Equal(t, tc.wantPassed, r.Passed(), r.String())
			assert.Equal(t, !tc.wantPassed, r.HasFailures())
			if !tc.wantPassed {
				assert.Contains(t, r.String(), "the report's minimum Grafana is")
				assert.Contains(t, r.String(), "pass the options Input receives")
			}
		})
	}
}

func TestSuiteRejectsInvalidReportedMode(t *testing.T) {
	for _, mode := range []Mode{"", "magic"} {
		t.Run(string(mode), func(t *testing.T) {
			s := positiveScenario()
			s.Input = func(context.Context, ...Option) (map[string]*Report, error) {
				return map[string]*Report{"A": {RefID: "A", Mode: mode}}, nil
			}
			s.Want = map[string]Contract{"A": {Rows: new(0)}}
			r, err := Evaluate(t.Context(), oneScenario(s))
			require.NoError(t, err)
			assert.False(t, r.Passed())
			assert.True(t, r.HasFailures())
			assert.Contains(t, r.String(), "invalid conversion mode")
		})
	}
}

func TestResponseReportsPreserveModeledMinimum(t *testing.T) {
	opt := MinGrafana("13.3.0-pre")
	response := &backend.QueryDataResponse{Responses: backend.Responses{
		"A": {Frames: data.Frames{tableFrame()}},
		"B": {Error: errors.New("upstream unavailable")},
		"Z": {Frames: data.Frames{tableFrame()}},
	}}
	for _, reports := range []map[string]*Report{
		CheckResponse(response, []string{"A", "B", "C"}, opt),
		CheckResponse(nil, []string{"A"}, opt),
	} {
		for refID, report := range reports {
			require.NotNil(t, report.minGrafana, refID)
			assert.Equal(t, "13.3.0-pre", report.minGrafana.String(), refID)
		}
	}
	for _, tc := range []struct {
		name string
		h    handlerFunc
	}{
		{"response", func(context.Context, *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
			return response, nil
		}},
		{"nil response", func(context.Context, *backend.QueryDataRequest) (*backend.QueryDataResponse, error) { return nil, nil }},
		{"handler error", func(context.Context, *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
			return nil, errors.New("upstream unavailable")
		}},
		{"handler panic", func(context.Context, *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
			panic("upstream panic")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reports, err := CheckQueryData(t.Context(), tc.h, request("A", "B", "C"), opt)
			require.NoError(t, err)
			for refID, report := range reports {
				require.NotNil(t, report.minGrafana, refID)
				assert.Equal(t, "13.3.0-pre", report.minGrafana.String(), refID)
			}
		})
	}
}

func TestSuiteResponseFailuresAtSelectedMinimum(t *testing.T) {
	for _, tc := range []struct {
		name     string
		response *backend.QueryDataResponse
		code     Code
		mode     Mode
		negative bool
	}{
		{"nil response", nil, CodeNilResponse, ModeNone, true},
		{"missing response", &backend.QueryDataResponse{}, CodeMissingRefID, ModeTable, false},
		{"query error", &backend.QueryDataResponse{Responses: backend.Responses{"A": {Error: errors.New("upstream unavailable")}}}, CodeQueryError, ModeNone, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := positiveScenario()
			s.Name = tc.name
			s.Input = ResponseInput(tc.response, []string{"A"})
			s.Want = map[string]Contract{"A": {Mode: tc.mode, Rows: new(0)}}
			s.ExpectedFailure = tc.negative
			s.Allowances = []Allowance{{Code: tc.code, RefID: "A", Reason: "deliberate missing/error response fixture"}}
			suite := oneScenario(positiveScenario())
			suite.Scenarios = append(suite.Scenarios, s)
			r, err := Evaluate(t.Context(), suite, MinGrafana("13.3.0-pre"))
			require.NoError(t, err)
			assert.True(t, r.Passed(), r.String())
			assert.False(t, r.HasFailures())
		})
	}
}

func TestSuiteRejectsNegativeReportedRows(t *testing.T) {
	s := positiveScenario()
	s.Input = func(context.Context, ...Option) (map[string]*Report, error) {
		return map[string]*Report{"A": {RefID: "A", Mode: ModeTable, Rows: -1}}, nil
	}
	s.Want = map[string]Contract{"A": {Mode: ModeTable}}
	r, err := Evaluate(t.Context(), oneScenario(s))
	require.NoError(t, err)
	assert.False(t, r.Passed())
	assert.True(t, r.HasFailures())
	assert.Contains(t, r.String(), "report has a negative row count")
}
