package datasource

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func TestQueryTypeMux(t *testing.T) {
	mux := NewQueryTypeMux()
	aHandler := &testHandler{}
	mux.Handle("a", aHandler)
	bHandler := &testHandler{}
	mux.Handle("b", bHandler)

	res, err := mux.QueryData(context.Background(), &backend.QueryDataRequest{
		Queries: []backend.DataQuery{
			{
				RefID:     "A",
				QueryType: "a",
			},
			{
				RefID:     "B",
				QueryType: "b",
			},
			{
				RefID:     "C",
				QueryType: "a",
			},
			{
				RefID:     "D",
				QueryType: "d",
			},
		},
	})

	require.NoError(t, err)
	require.Equal(t, 1, aHandler.callCount)
	require.Len(t, aHandler.request.Queries, 2)
	require.Equal(t, "A", aHandler.request.Queries[0].RefID)
	require.Equal(t, "C", aHandler.request.Queries[1].RefID)

	require.Equal(t, 1, bHandler.callCount)
	require.Len(t, bHandler.request.Queries, 1)
	require.Equal(t, "B", bHandler.request.Queries[0].RefID)
	require.Len(t, res.Responses, 4)
	require.Equal(t, "no handler found for query type 'd'", res.Responses["D"].Error.Error())

	t.Run("When overriding fallback handler should call fallback handler", func(t *testing.T) {
		errBoom := errors.New("BOOM")
		mux.HandleFunc("", func(_ context.Context, _ *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
			return nil, errBoom
		})
		res, err = mux.QueryData(context.Background(), &backend.QueryDataRequest{
			Queries: []backend.DataQuery{
				{
					RefID:     "A",
					QueryType: "unhandled",
				},
			},
		})

		require.Nil(t, res)
		require.Error(t, err)
		require.Equal(t, errBoom, err)
	})
}

type testHandler struct {
	callCount int
	request   *backend.QueryDataRequest
}

func (th *testHandler) QueryData(_ context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	th.callCount++
	th.request = req
	responses := backend.Responses{}
	for _, q := range req.Queries {
		responses[q.RefID] = backend.DataResponse{}
	}

	return &backend.QueryDataResponse{
		Responses: responses,
	}, nil
}

func TestQueryTypeMuxQueryChunkedData(t *testing.T) {
	errBoom := errors.New("BOOM")
	newMux := func() *QueryTypeMux {
		mux := NewQueryTypeMux()
		mux.HandleFunc("frames", func(_ context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
			rsp := backend.NewQueryDataResponse()
			for _, q := range req.Queries {
				rsp.Responses[q.RefID] = backend.DataResponse{Frames: data.Frames{data.NewFrame("a"), data.NewFrame("b")}}
			}
			return rsp, nil
		})
		mux.HandleFunc("error", func(_ context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
			rsp := backend.NewQueryDataResponse()
			for _, q := range req.Queries {
				rsp.Responses[q.RefID] = backend.ErrDataResponse(backend.StatusBadRequest, "bad query")
			}
			return rsp, nil
		})
		mux.HandleChunkedQueryType("chunked", func(_ context.Context, _ *backend.QueryChunkedDataRequest, _ backend.ChunkedDataWriter) error {
			return errBoom
		})
		return mux
	}
	request := func(queries ...backend.DataQuery) *backend.QueryChunkedDataRequest {
		return &backend.QueryChunkedDataRequest{Queries: queries}
	}

	t.Run("writes the frames and the errors of the fallback", func(t *testing.T) {
		w := &recordingChunkedWriter{}
		err := newMux().QueryChunkedData(context.Background(), request(
			backend.DataQuery{RefID: "A", QueryType: "frames"},
			backend.DataQuery{RefID: "B", QueryType: "frames"},
			backend.DataQuery{RefID: "C", QueryType: "error"},
		), w)

		require.NoError(t, err)
		require.ElementsMatch(t, []string{"A/f0", "A/f1", "B/f0", "B/f1"}, w.frames)
		require.Equal(t, map[string]backend.Status{"C": backend.StatusBadRequest}, w.errors)
	})

	t.Run("returns the WriteFrame errors of all queries", func(t *testing.T) {
		w := &recordingChunkedWriter{frameErr: errBoom}
		err := newMux().QueryChunkedData(context.Background(), request(
			backend.DataQuery{RefID: "A", QueryType: "frames"},
			backend.DataQuery{RefID: "B", QueryType: "frames"},
		), w)

		require.ErrorIs(t, err, errBoom)
		require.ErrorContains(t, err, "write A")
		require.ErrorContains(t, err, "write B")
		// Each query stops at its first failed write.
		require.ElementsMatch(t, []string{"A/f0", "B/f0"}, w.frames)
	})

	t.Run("returns the WriteError errors of all queries", func(t *testing.T) {
		w := &recordingChunkedWriter{errorErr: errBoom}
		err := newMux().QueryChunkedData(context.Background(), request(
			backend.DataQuery{RefID: "A", QueryType: "error"},
			backend.DataQuery{RefID: "B", QueryType: "error"},
		), w)

		require.ErrorIs(t, err, errBoom)
		require.ErrorContains(t, err, "write A")
		require.ErrorContains(t, err, "write B")
	})

	t.Run("returns the error of a chunked handler", func(t *testing.T) {
		w := &recordingChunkedWriter{}
		err := newMux().QueryChunkedData(context.Background(), request(
			backend.DataQuery{RefID: "A", QueryType: "chunked"},
			backend.DataQuery{RefID: "B", QueryType: "frames"},
		), w)

		require.ErrorIs(t, err, errBoom)
		require.ElementsMatch(t, []string{"B/f0", "B/f1"}, w.frames)
	})
}

// recordingChunkedWriter records the writes, and fails them with frameErr and errorErr.
// The mux calls it from several goroutines, so it uses a lock.
type recordingChunkedWriter struct {
	mu       sync.Mutex
	frameErr error
	errorErr error
	frames   []string
	errors   map[string]backend.Status
}

func (w *recordingChunkedWriter) WriteFrame(_ context.Context, refID string, frameID string, f *data.Frame) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if f.RefID != refID {
		return fmt.Errorf("frame has refID %q, want %q", f.RefID, refID)
	}
	w.frames = append(w.frames, refID+"/"+frameID)
	if w.frameErr != nil {
		return fmt.Errorf("write %s: %w", refID, w.frameErr)
	}
	return nil
}

func (w *recordingChunkedWriter) WriteError(_ context.Context, refID string, status backend.Status, _ error) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.errors == nil {
		w.errors = map[string]backend.Status{}
	}
	w.errors[refID] = status
	if w.errorErr != nil {
		return fmt.Errorf("write %s: %w", refID, w.errorErr)
	}
	return nil
}
