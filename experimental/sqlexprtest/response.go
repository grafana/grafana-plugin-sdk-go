package sqlexprtest

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// HeaderFromExpression is the header Grafana sets on QueryData requests that
// come from server-side expressions, such as SQL Expressions.
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
				Code: CodeMissingRefID, Severity: SeverityError, Owner: OwnerPlugin, Frame: -1,
				Message: "the response has no entry for this RefID. Grafana then treats the query as having no data, " +
					"without an error. Return a DataResponse for every query, keyed by its RefID.",
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
// It sends a copy of req with the [HeaderFromExpression] header set. A panic
// in the handler becomes a [CodePanic] finding for every query, because a
// panic fails the whole SQL Expression request. The returned error is the
// handler's own error.
func CheckQueryData(ctx context.Context, h backend.QueryDataHandler, req *backend.QueryDataRequest) (map[string]*Report, error) {
	r := *req
	r.Headers = maps.Clone(req.Headers)
	if r.Headers == nil {
		r.Headers = map[string]string{}
	}
	r.SetHTTPHeader(HeaderFromExpression, "true")

	refIDs := make([]string, len(req.Queries))
	for i, q := range req.Queries {
		refIDs[i] = q.RefID
	}

	resp, panicked, err := callQueryData(ctx, h, &r)
	if panicked != nil {
		reports := map[string]*Report{}
		for _, refID := range refIDs {
			reports[refID] = &Report{RefID: refID, Mode: ModeNone, Findings: []Finding{{
				Code: CodePanic, Severity: SeverityError, Owner: OwnerPlugin, Frame: -1,
				Message: fmt.Sprintf("the QueryData handler panicked: %v. A panic fails the whole SQL Expression request, "+
					"for every query in it. Return an error in the DataResponse instead.", panicked),
			}}}
		}
		return reports, nil
	}
	if err != nil {
		return nil, err
	}
	return CheckResponse(resp, refIDs...), nil
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
