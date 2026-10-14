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

	reports := CheckResponse(resp, "A", "B", "C")

	require.Len(t, reports, 4)
	assert.Empty(t, reports["A"].Findings)
	assert.Equal(t, "A", reports["A"].RefID)
	assert.Equal(t, []Code{CodeQueryError}, codes(reports["B"]))
	assert.Contains(t, reports["B"].Findings[0].Message, "boom")
	assert.Equal(t, []Code{CodeMissingRefID}, codes(reports["C"]))
	assert.Equal(t, []Code{CodeUnexpectedRefID}, codes(reports["Z"]))
}

func TestCheckResponseWithoutRefIDs(t *testing.T) {
	resp := &backend.QueryDataResponse{Responses: backend.Responses{"A": {Frames: data.Frames{tableFrame()}}}}

	reports := CheckResponse(resp)

	require.Len(t, reports, 1)
	assert.Empty(t, reports["A"].Findings)
}

func TestCheckResponseNil(t *testing.T) {
	reports := CheckResponse(nil, "A")

	assert.Equal(t, []Code{CodeMissingRefID}, codes(reports["A"]))
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
	assert.Equal(t, "true", got.GetHTTPHeader(HeaderFromExpression))
	assert.Equal(t, "true", got.Headers["FromAlert"], "keeps the request's headers")
	assert.Empty(t, req.GetHTTPHeader(HeaderFromExpression), "does not change the caller's request")
	assert.Empty(t, reports["A"].Findings)
	assert.Equal(t, []Code{CodeMissingRefID}, codes(reports["B"]))
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

	require.EqualError(t, err, "unavailable")
	assert.Nil(t, reports)
}
