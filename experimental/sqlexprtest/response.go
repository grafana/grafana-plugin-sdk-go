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
// response has.
//
// The result has one report per RefID, including RefIDs the response has but
// no query asked for.
func CheckResponse(resp *backend.QueryDataResponse, refIDs ...string) map[string]*Report {
	reports := map[string]*Report{}
	var responses backend.Responses
	if resp != nil {
		responses = resp.Responses
	}
	if len(refIDs) == 0 {
		refIDs = slices.Sorted(maps.Keys(responses))
	}

	for _, refID := range refIDs {
		dr, ok := responses[refID]
		switch {
		case !ok:
			reports[refID] = &Report{RefID: refID, Mode: ModeNone, Findings: []Finding{{
				Code: CodeMissingRefID, Severity: SeverityWarning, Owner: OwnerPlugin, Frame: -1,
				Message: "the response has no entry for this RefID, so SQL sees a table with no columns, without an error. " +
					"Return a DataResponse for every query, keyed by its RefID.",
			}}}
		case dr.Error != nil:
			reports[refID] = &Report{RefID: refID, Mode: ModeNone, Findings: []Finding{{
				Code: CodeQueryError, Severity: SeverityError, Owner: OwnerPlugin, Frame: -1,
				Message: fmt.Sprintf("the query failed: %v", dr.Error),
			}}}
		default:
			r := Check(dr.Frames)
			r.RefID = refID
			reports[refID] = r
		}
	}

	for refID := range responses {
		if _, ok := reports[refID]; ok {
			continue
		}
		reports[refID] = &Report{RefID: refID, Mode: ModeNone, Findings: []Finding{{
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
// holds only that query and has the [HeaderFromExpression] header set. The
// copies share req's PluginContext. A handler error becomes a
// [CodeQueryError] finding, and a panic a [CodePanic] finding, for that query.
// It returns an error only when req is nil.
func CheckQueryData(ctx context.Context, h backend.QueryDataHandler, req *backend.QueryDataRequest) (map[string]*Report, error) {
	if req == nil {
		return nil, errors.New("sqlexprtest: nil QueryDataRequest")
	}

	reports := map[string]*Report{}
	for _, q := range req.Queries {
		one := *req
		one.Queries = []backend.DataQuery{q}
		one.Headers = maps.Clone(req.Headers)
		if one.Headers == nil {
			one.Headers = map[string]string{}
		}
		one.SetHTTPHeader(HeaderFromExpression, "true")

		resp, panicked, err := callQueryData(ctx, h, &one)
		switch {
		case panicked != nil:
			reports[q.RefID] = &Report{RefID: q.RefID, Mode: ModeNone, Findings: []Finding{{
				Code: CodePanic, Severity: SeverityError, Owner: OwnerPlugin, Frame: -1,
				Message: fmt.Sprintf("the QueryData handler panicked: %v. A panic fails the whole SQL Expression request, "+
					"for every query in it. Return an error in the DataResponse instead.", panicked),
			}}}
		case err != nil:
			reports[q.RefID] = &Report{RefID: q.RefID, Mode: ModeNone, Findings: []Finding{{
				Code: CodeQueryError, Severity: SeverityError, Owner: OwnerPlugin, Frame: -1,
				Message: fmt.Sprintf("the QueryData handler returned an error: %v", err),
			}}}
		default:
			reports[q.RefID] = checkOneResponse(resp, q.RefID)
		}
	}
	return reports, nil
}

// checkOneResponse checks the response to a request for one query. Entries for
// other RefIDs become an unexpected-refid finding on that query's report.
func checkOneResponse(resp *backend.QueryDataResponse, refID string) *Report {
	reports := CheckResponse(resp, refID)
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
