package sqlexprtest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// recorder is a testing.TB that records failures instead of failing.
type recorder struct {
	testing.TB
	errors []string
}

func (r *recorder) Helper() {}

func (r *recorder) Errorf(format string, args ...any) {
	r.errors = append(r.errors, fmt.Sprintf(format, args...))
}

func droppedFieldFrames() data.Frames {
	return data.Frames{typed(data.FrameTypeTimeSeriesWide, data.NewFrame("",
		data.NewField("time", nil, []time.Time{t0}),
		data.NewField("state", nil, []string{"ok"}),
		data.NewField("cpu", nil, []float64{1}),
	))}
}

func TestAssertFrames(t *testing.T) {
	t.Run("fails on warnings by default", func(t *testing.T) {
		rec := &recorder{TB: t}
		AssertFrames(rec, droppedFieldFrames())
		assert.Len(t, rec.errors, 1)
		assert.Contains(t, rec.errors[0], "dropped-field")
	})

	t.Run("ignores advice by default", func(t *testing.T) {
		rec := &recorder{TB: t}
		AssertFrames(rec, data.Frames{data.NewFrame("", data.NewField("v", nil, []float64{1}))})
		assert.Empty(t, rec.errors)
	})

	t.Run("FailOn", func(t *testing.T) {
		rec := &recorder{TB: t}
		AssertFrames(rec, droppedFieldFrames(), FailOn(SeverityError))
		assert.Empty(t, rec.errors)
	})

	t.Run("Ignore", func(t *testing.T) {
		rec := &recorder{TB: t}
		r := AssertFrames(rec, droppedFieldFrames(), Ignore(CodeDroppedField))
		assert.Empty(t, rec.errors)
		assert.True(t, r.Has(CodeDroppedField), "ignoring a finding only suppresses the assertion")
	})

	t.Run("FailOn info fails advice", func(t *testing.T) {
		rec := &recorder{TB: t}
		r := AssertFrames(rec, data.Frames{data.NewFrame("", data.NewField("v", nil, []float64{1}))}, FailOn(SeverityInfo))
		require.Len(t, rec.errors, 1)
		assert.Equal(t, SeverityInfo, r.Worst())
		assert.Contains(t, rec.errors[0], "missing-type")
	})

	t.Run("FailOn error still fails errors", func(t *testing.T) {
		rec := &recorder{TB: t}
		r := AssertFrames(rec, data.Frames{nil}, FailOn(SeverityError))
		require.Len(t, rec.errors, 1)
		assert.Equal(t, SeverityError, r.Worst())
		assert.Contains(t, rec.errors[0], "nil-frame")
	})
}

func TestAssertResponse(t *testing.T) {
	rec := &recorder{TB: t}
	resp := &backend.QueryDataResponse{Responses: backend.Responses{
		"A": {Frames: droppedFieldFrames()},
		"B": {Frames: data.Frames{tableFrame()}},
	}}

	reports := AssertResponse(rec, resp, []string{"A", "B", "C"})

	assert.Len(t, reports, 3)
	assert.Len(t, rec.errors, 2, "one failure for A, one for the missing C")
}

func TestAssertResponseFailsWithNothingToCheck(t *testing.T) {
	for _, resp := range []*backend.QueryDataResponse{{}, backend.NewQueryDataResponse()} {
		rec := &recorder{TB: t}
		reports := AssertResponse(rec, resp, nil)
		assert.Empty(t, reports)
		require.Len(t, rec.errors, 1)
		assert.Contains(t, rec.errors[0], "no queries to check")
	}
}

func TestAssertQueryData(t *testing.T) {
	rec := &recorder{TB: t}
	h := handlerFunc(func(_ context.Context, r *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
		resp := backend.NewQueryDataResponse()
		resp.Responses[r.Queries[0].RefID] = backend.DataResponse{Frames: droppedFieldFrames()}
		return resp, nil
	})

	reports := AssertQueryData(rec, context.Background(), h, request("A"))

	assert.Len(t, reports, 1)
	assert.Len(t, rec.errors, 1)
}

func TestIgnoreLeavesUnrelatedFailures(t *testing.T) {
	rec := &recorder{TB: t}
	resp := &backend.QueryDataResponse{Responses: backend.Responses{
		"A": {Frames: droppedFieldFrames()},
		"B": {Error: fmt.Errorf("query unavailable")},
	}}
	reports := AssertResponse(rec, resp, []string{"A", "B"}, Ignore(CodeDroppedField))
	require.Len(t, rec.errors, 1)
	assert.Contains(t, rec.errors[0], "query-error")
	assert.True(t, reports["A"].Has(CodeDroppedField))
	assert.True(t, reports["B"].Has(CodeQueryError))
}

func TestAssertQueryDataInvalidRequest(t *testing.T) {
	for _, req := range []*backend.QueryDataRequest{nil, request(), request("A", "A")} {
		rec := &recorder{TB: t}
		reports := AssertQueryData(rec, t.Context(), handlerFunc(nil), req)
		assert.Nil(t, reports)
		assert.Len(t, rec.errors, 1)
	}
}
