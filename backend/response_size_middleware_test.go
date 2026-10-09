package backend

import (
	"context"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runResponseSize(t *testing.T, responses Responses) *QueryDataResponse {
	t.Helper()
	next := &Handlers{
		QueryDataHandler: QueryDataHandlerFunc(func(context.Context, *QueryDataRequest) (*QueryDataResponse, error) {
			return &QueryDataResponse{Responses: responses}, nil
		}),
	}
	resp, err := ResponseSizeMiddleware().CreateHandlerMiddleware(next).QueryData(context.Background(), &QueryDataRequest{})
	require.NoError(t, err)
	return resp
}

func encodedLen(t *testing.T, f *data.Frame) float64 {
	t.Helper()
	b, err := f.MarshalArrow()
	require.NoError(t, err)
	return float64(len(b))
}

func TestResponseSizeMiddlewareAddsArrowEncodedSizeToFirstFrame(t *testing.T) {
	frame := data.NewFrame("f", data.NewField("value", nil, []int64{1, 2, 3}))
	want := encodedLen(t, frame)

	resp := runResponseSize(t, Responses{"A": {Frames: data.Frames{frame}}})

	stats := resp.Responses["A"].Frames[0].Meta.Stats
	require.Len(t, stats, 1)
	assert.Equal(t, ResponseSizeStatName, stats[0].DisplayName)
	assert.Equal(t, "bytes", stats[0].Unit)
	assert.Equal(t, want, stats[0].Value)
}

func TestResponseSizeMiddlewareSizesEachQueryOnItsOwn(t *testing.T) {
	frameA := data.NewFrame("f", data.NewField("value", nil, []int64{1, 2, 3}))
	frameB := data.NewFrame("f", data.NewField("value", nil, []int64{4, 5, 6, 7}))
	wantA, wantB := encodedLen(t, frameA), encodedLen(t, frameB)

	resp := runResponseSize(t, Responses{"A": {Frames: data.Frames{frameA}}, "B": {Frames: data.Frames{frameB}}})

	assert.Equal(t, wantA, resp.Responses["A"].Frames[0].Meta.Stats[0].Value)
	assert.Equal(t, wantB, resp.Responses["B"].Frames[0].Meta.Stats[0].Value)
}

func TestResponseSizeMiddlewareSkipsQueryWhenAFrameFailsToEncode(t *testing.T) {
	good := data.NewFrame("f", data.NewField("value", nil, []int64{1, 2, 3}))
	// Fields of different lengths make MarshalArrow fail.
	bad := data.NewFrame("f", data.NewField("a", nil, []int64{1, 2, 3}), data.NewField("b", nil, []int64{1}))

	resp := runResponseSize(t, Responses{"A": {Frames: data.Frames{good, bad}}})

	assert.Nil(t, resp.Responses["A"].Frames[0].Meta, "a partial total would understate the size")
}
