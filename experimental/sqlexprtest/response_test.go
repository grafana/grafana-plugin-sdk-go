package sqlexprtest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func tableFrame() *data.Frame {
	return typed(data.FrameTypeTable, data.NewFrame("", data.NewField("value", nil, []float64{1})))
}

func TestCheckResponse(t *testing.T) {
	resp := &backend.QueryDataResponse{Responses: backend.Responses{
		"A": {Frames: data.Frames{tableFrame()}},
		"B": {Error: errors.New("boom")},
		"Z": {Frames: data.Frames{tableFrame()}},
	}}

	reports := CheckResponse(resp, []string{"A", "B", "C"})

	require.Len(t, reports, 4)
	assert.Empty(t, reports["A"].Findings)
	assert.Equal(t, "A", reports["A"].RefID)
	assert.Equal(t, []Code{CodeQueryError}, codes(reports["B"]))
	assert.Contains(t, reports["B"].Findings[0].Message, "boom")
	assert.Equal(t, []Code{CodeMissingRefID}, codes(reports["C"]))
	assert.Equal(t, ModeTable, reports["C"].Mode)
	assert.Zero(t, reports["C"].Rows)
	assert.Empty(t, reports["C"].Columns)
	assert.Equal(t, []Code{CodeUnexpectedRefID}, codes(reports["Z"]))
}

func TestCheckResponseWithoutRefIDs(t *testing.T) {
	resp := &backend.QueryDataResponse{Responses: backend.Responses{"A": {Frames: data.Frames{tableFrame()}}}}

	reports := CheckResponse(resp, nil)

	require.Len(t, reports, 1)
	assert.Empty(t, reports["A"].Findings)
}

func TestCheckResponseNil(t *testing.T) {
	for _, refIDs := range [][]string{{"A", "B"}, nil} {
		reports := CheckResponse(nil, refIDs)
		if len(refIDs) == 0 {
			refIDs = []string{""}
		}
		require.Len(t, reports, len(refIDs))
		for _, refID := range refIDs {
			r := reports[refID]
			assert.Equal(t, []Code{CodeNilResponse}, codes(r))
			assert.Equal(t, ModeNone, r.Mode)
			assert.Equal(t, SeverityError, r.Worst())
		}
	}
}

type handlerFunc func(context.Context, *backend.QueryDataRequest) (*backend.QueryDataResponse, error)

func (f handlerFunc) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	return f(ctx, req)
}

func request(refIDs ...string) *backend.QueryDataRequest {
	req := &backend.QueryDataRequest{Headers: map[string]string{"FromAlert": "true"}}
	for _, id := range refIDs {
		req.Queries = append(req.Queries, backend.DataQuery{RefID: id, TimeRange: backend.TimeRange{From: t0, To: t0.Add(time.Hour)}})
	}
	return req
}

func TestCheckQueryData(t *testing.T) {
	req := request("A", "B")
	var got *backend.QueryDataRequest
	h := handlerFunc(func(_ context.Context, r *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
		got = r
		return &backend.QueryDataResponse{Responses: backend.Responses{"A": {Frames: data.Frames{tableFrame()}}}}, nil
	})

	reports, err := CheckQueryData(context.Background(), h, req)

	require.NoError(t, err)
	assert.Empty(t, got.GetHTTPHeader(HeaderFromExpression), "does not manufacture a header")
	assert.Equal(t, "true", got.Headers["FromAlert"], "keeps the request's headers")
	assert.Empty(t, req.GetHTTPHeader(HeaderFromExpression), "does not change the caller's request")
	require.Len(t, reports, 2)
	assert.Empty(t, reports["A"].Findings)
	assert.Equal(t, []Code{CodeMissingRefID, CodeUnexpectedRefID}, codes(reports["B"]))
	assert.Equal(t, "A", reports["B"].Findings[1].Field)
}

func TestCheckQueryDataSendsOneQueryPerRequest(t *testing.T) {
	var got [][]string
	h := handlerFunc(func(_ context.Context, r *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
		ids := []string{}
		resp := backend.NewQueryDataResponse()
		for _, q := range r.Queries {
			ids = append(ids, q.RefID)
			resp.Responses[q.RefID] = backend.DataResponse{Frames: data.Frames{tableFrame()}}
		}
		got = append(got, ids)
		return resp, nil
	})

	reports, err := CheckQueryData(context.Background(), h, request("A", "B"))

	require.NoError(t, err)
	assert.Equal(t, [][]string{{"A"}, {"B"}}, got)
	assert.Empty(t, reports["A"].Findings)
	assert.Empty(t, reports["B"].Findings)
}

func TestCheckQueryDataNilRequest(t *testing.T) {
	_, err := CheckQueryData(context.Background(), handlerFunc(nil), nil)
	require.Error(t, err)
}

func TestCheckQueryDataPreservesHeaders(t *testing.T) {
	for _, value := range []string{"", "false", "true"} {
		t.Run("expression header "+value, func(t *testing.T) {
			req := request("A", "B")
			if value != "" {
				req.SetHTTPHeader(HeaderFromExpression, value)
			}
			calls := 0
			h := handlerFunc(func(_ context.Context, one *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
				calls++
				require.Equal(t, value, one.GetHTTPHeader(HeaderFromExpression))
				require.Equal(t, "true", one.Headers["FromAlert"])
				frames := data.Frames{data.NewFrame("", data.NewField("v", nil, []float64{1})), data.NewFrame("", data.NewField("v", nil, []float64{2}))}
				if one.GetHTTPHeader(HeaderFromExpression) == "true" {
					frames = data.Frames{tableFrame()}
				}
				one.Headers["FromAlert"] = "changed by handler"
				one.SetHTTPHeader(HeaderFromExpression, "changed by handler")
				return &backend.QueryDataResponse{Responses: backend.Responses{one.Queries[0].RefID: {Frames: frames}}}, nil
			})
			reports, err := CheckQueryData(t.Context(), h, req)
			require.NoError(t, err)
			require.Equal(t, 2, calls)
			for _, refID := range []string{"A", "B"} {
				if value == "true" {
					assert.Equal(t, ModeTable, reports[refID].Mode)
					assert.Empty(t, reports[refID].Findings)
				} else {
					assert.Equal(t, ModeNone, reports[refID].Mode)
					assert.Equal(t, []Code{CodeMultipleTableFrames}, codes(reports[refID]))
				}
			}
			assert.Equal(t, "true", req.Headers["FromAlert"])
			assert.Equal(t, value, req.GetHTTPHeader(HeaderFromExpression))
		})
	}

	t.Run("nil headers", func(t *testing.T) {
		req := request("A")
		req.Headers = nil
		h := handlerFunc(func(_ context.Context, one *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
			require.Nil(t, one.Headers)
			return &backend.QueryDataResponse{Responses: backend.Responses{"A": {Frames: data.Frames{tableFrame()}}}}, nil
		})
		_, err := CheckQueryData(t.Context(), h, req)
		require.NoError(t, err)
		assert.Nil(t, req.Headers)
	})
}

func TestCheckQueryDataDuplicateRefIDs(t *testing.T) {
	calls := 0
	h := handlerFunc(func(context.Context, *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
		calls++
		return nil, nil
	})
	reports, err := CheckQueryData(t.Context(), h, request("A", "B", "A"))
	require.ErrorContains(t, err, `duplicate query RefID "A"`)
	assert.Nil(t, reports)
	assert.Zero(t, calls, "validate all RefIDs before executing any query")
}

func TestCheckQueryDataNilResponse(t *testing.T) {
	h := handlerFunc(func(context.Context, *backend.QueryDataRequest) (*backend.QueryDataResponse, error) { return nil, nil })
	reports, err := CheckQueryData(t.Context(), h, request("A"))
	require.NoError(t, err)
	assert.Equal(t, []Code{CodeNilResponse}, codes(reports["A"]))
	assert.Equal(t, SeverityError, reports["A"].Worst())
}

func TestCheckQueryDataPanic(t *testing.T) {
	h := handlerFunc(func(context.Context, *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
		panic("no frames to read")
	})

	reports, err := CheckQueryData(context.Background(), h, request("A", "B"))

	require.NoError(t, err)
	assert.Equal(t, []Code{CodePanic}, codes(reports["A"]))
	assert.Equal(t, []Code{CodePanic}, codes(reports["B"]))
	assert.Contains(t, reports["A"].Findings[0].Message, "no frames to read")
}

func TestCheckQueryDataError(t *testing.T) {
	h := handlerFunc(func(context.Context, *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
		return nil, errors.New("unavailable")
	})

	reports, err := CheckQueryData(context.Background(), h, request("A"))

	require.NoError(t, err)
	assert.Equal(t, []Code{CodeQueryError}, codes(reports["A"]))
	assert.Contains(t, reports["A"].Findings[0].Message, "unavailable")
}
