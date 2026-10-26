package sqlexprtest

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// HeaderFromExpression is the header Grafana declares to mark QueryData
// requests that come from server-side expressions, such as SQL Expressions.
// Some Grafana versions forward it only when the browser sends it, so a plugin
// should not depend on it alone.
const HeaderFromExpression = "X-Grafana-From-Expr"

// CheckResponse checks every query in a QueryDataResponse. refIDs are the
// RefIDs the request asked for. When refIDs is empty, it checks the RefIDs the
// response has. It uses only the [MinGrafana] option.
//
// The result has one report per RefID, including RefIDs the response has but
// no query asked for. A nil response gives an error report for each RefID in
// refIDs, or one report under the RefID "" when refIDs is empty.
func CheckResponse(resp *backend.QueryDataResponse, refIDs []string, opts ...Option) map[string]*Report {
	o := newOptions(opts)
	reports := map[string]*Report{}
	if resp == nil {
		if len(refIDs) == 0 {
			refIDs = []string{""}
		}
		for _, refID := range refIDs {
			reports[refID] = &Report{RefID: refID, minGrafana: o.minGrafana, Mode: ModeNone, Findings: []Finding{{
				Code: CodeNilResponse, Severity: SeverityError, Owner: OwnerPlugin, Frame: -1,
				Message: "the QueryData handler returned a nil response and no error, so Grafana has no results to read. " +
					"Return a QueryDataResponse, or an error.",
			}}}
		}
		return reports
	}
	responses := resp.Responses
	if len(refIDs) == 0 {
		refIDs = slices.Sorted(maps.Keys(responses))
	}

	for _, refID := range refIDs {
		dr, ok := responses[refID]
		switch {
		case !ok:
			reports[refID] = &Report{RefID: refID, minGrafana: o.minGrafana, Mode: ModeTable, Findings: []Finding{{
				Code: CodeMissingRefID, Severity: SeverityWarning, Owner: OwnerPlugin, Frame: -1,
				Message: "the response has no entry for this RefID, so SQL sees a table with no columns, without an error. " +
					"Return a DataResponse for every query, keyed by its RefID.",
			}}}
		case dr.Error != nil:
			reports[refID] = &Report{RefID: refID, minGrafana: o.minGrafana, Mode: ModeNone, Findings: []Finding{{
				Code: CodeQueryError, Severity: SeverityError, Owner: OwnerPlugin, Frame: -1,
				Message: fmt.Sprintf("the query failed: %v", dr.Error),
			}}}
		default:
			r := Check(dr.Frames, opts...)
			r.RefID = refID
			reports[refID] = r
		}
	}

	for refID := range responses {
		if _, ok := reports[refID]; ok {
			continue
		}
		reports[refID] = &Report{RefID: refID, minGrafana: o.minGrafana, Mode: ModeNone, Findings: []Finding{{
			Code: CodeUnexpectedRefID, Severity: SeverityWarning, Owner: OwnerPlugin, Frame: -1,
			Message: "the response has an entry for this RefID, but no query asked for it, so SQL never reads it. " +
				"Key each DataResponse by its query's RefID.",
		}}}
	}
	return reports
}

// CheckQueryData calls a QueryData handler the way SQL Expressions call it,
// then checks the response for each query in req.
//
// Grafana sends each SQL Expression input query in a request of its own, so
// CheckQueryData calls the handler once per query, with a copy of req that
// holds only that query. It passes req's headers unchanged. To test a request
// with the [HeaderFromExpression] header, set it on req. The copies share
// req's PluginContext. It doesn't model the sseGroupByDatasource feature
// flag, which sends a data source's queries in one request.
//
// A handler error becomes a [CodeQueryError] finding, and a panic a
// [CodePanic] finding, for that query. It uses only the [MinGrafana] option.
// It returns an error, without calling the handler, when req is nil or two
// queries share a RefID.
func CheckQueryData(ctx context.Context, h backend.QueryDataHandler, req *backend.QueryDataRequest, opts ...Option) (map[string]*Report, error) {
	if req == nil {
		return nil, errors.New("sqlexprtest: nil QueryDataRequest")
	}
	seen := make(map[string]struct{}, len(req.Queries))
	for _, q := range req.Queries {
		if _, ok := seen[q.RefID]; ok {
			return nil, fmt.Errorf("sqlexprtest: duplicate query RefID %q", q.RefID)
		}
		seen[q.RefID] = struct{}{}
	}

	o := newOptions(opts)
	reports := map[string]*Report{}
	for _, q := range req.Queries {
		one := *req
		one.Queries = []backend.DataQuery{q}
		one.Headers = maps.Clone(req.Headers)

		resp, panicked, err := callQueryData(ctx, h, &one)
		switch {
		case panicked != nil:
			reports[q.RefID] = &Report{RefID: q.RefID, minGrafana: o.minGrafana, Mode: ModeNone, Findings: []Finding{{
				Code: CodePanic, Severity: SeverityError, Owner: OwnerPlugin, Frame: -1,
				Message: fmt.Sprintf("the QueryData handler panicked: %v. Grafana gets no response for this query, and the panic can fail the whole request. "+
					"Return an error in the DataResponse instead.", panicked),
			}}}
		case err != nil:
			reports[q.RefID] = &Report{RefID: q.RefID, minGrafana: o.minGrafana, Mode: ModeNone, Findings: []Finding{{
				Code: CodeQueryError, Severity: SeverityError, Owner: OwnerPlugin, Frame: -1,
				Message: fmt.Sprintf("the QueryData handler returned an error: %v", err),
			}}}
		default:
			reports[q.RefID] = checkOneResponse(resp, q.RefID, opts)
		}
	}
	return reports, nil
}

// checkOneResponse checks the response to a request for one query. Entries for
// other RefIDs become an unexpected-refid finding on that query's report.
func checkOneResponse(resp *backend.QueryDataResponse, refID string, opts []Option) *Report {
	reports := CheckResponse(resp, []string{refID}, opts...)
	r := reports[refID]
	others := []string{}
	for id := range reports {
		if id != refID {
			others = append(others, id)
		}
	}
	slices.Sort(others)
	for _, id := range others {
		r.add(Finding{
			Code: CodeUnexpectedRefID, Severity: SeverityWarning, Owner: OwnerPlugin, Frame: -1, Field: id,
			Message: fmt.Sprintf("the response to this query also has an entry for RefID %q, which this request did not ask for, "+
				"so SQL never reads it. Key each DataResponse by its query's RefID.", id),
		})
	}
	return r
}

func callQueryData(ctx context.Context, h backend.QueryDataHandler, req *backend.QueryDataRequest) (resp *backend.QueryDataResponse, panicked any, err error) {
	defer func() {
		if p := recover(); p != nil {
			panicked = p
		}
	}()
	resp, err = h.QueryData(ctx, req)
	return resp, nil, err
}
