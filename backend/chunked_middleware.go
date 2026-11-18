package backend

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// Each query call retains only its last error per refID. Capture mode tracks
// the reserved refID instead. Neither mode retains frame data.
type chunkedQueryObserver struct {
	ChunkedDataWriter
	mu             sync.Mutex
	result         QueryDataResponse
	captureRefID   string
	captureRefUsed atomic.Bool
}

type rawChunkSink interface {
	OnChunk(*pluginv2.QueryChunkedDataResponse) error
}

type sourcedErrorSink interface {
	WriteErrorWithSource(context.Context, string, Status, error, ErrorSource) error
}

type rawQueryObserver struct {
	*chunkedQueryObserver
	raw rawChunkSink
}

func observeChunkedQuery(w ChunkedDataWriter, captureRefID string) (*chunkedQueryObserver, ChunkedDataWriter) {
	observer := &chunkedQueryObserver{ChunkedDataWriter: w, captureRefID: captureRefID}
	// A host can select a different transport when the writer accepts encoded chunks.
	if raw, ok := w.(rawChunkSink); ok {
		return observer, &rawQueryObserver{observer, raw}
	}
	return observer, observer
}

func (w *chunkedQueryObserver) markRef(refID string) {
	if w.captureRefID != "" && refID == w.captureRefID {
		w.captureRefUsed.Store(true)
	}
}

// Hold the lock until the write returns so failures for the same refID stay in wire order.
func (w *chunkedQueryObserver) recordError(refID string, status Status, err error, source ErrorSource, send func() error) error {
	w.markRef(refID)
	if err == nil || w.captureRefID != "" {
		return send()
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.result.Responses == nil {
		w.result.Responses = make(Responses)
	}
	if !source.IsValid() {
		source = ErrorSourceFromError(err)
	}
	if !status.IsValid() {
		status = statusFromError(err)
	}
	w.result.Responses[refID] = DataResponse{Error: err, Status: status, ErrorSource: source}
	return send()
}

func (w *chunkedQueryObserver) WriteFrame(ctx context.Context, refID, frameID string, f *data.Frame) error {
	w.markRef(refID)
	return w.ChunkedDataWriter.WriteFrame(ctx, refID, frameID, f)
}

func (w *chunkedQueryObserver) WriteError(ctx context.Context, refID string, status Status, err error) error {
	return w.recordError(refID, status, err, "", func() error {
		return w.ChunkedDataWriter.WriteError(ctx, refID, status, err)
	})
}

func (w *chunkedQueryObserver) WriteErrorWithSource(ctx context.Context, refID string, status Status, err error, source ErrorSource) error {
	return w.recordError(refID, status, err, source, func() error {
		if next, ok := w.ChunkedDataWriter.(sourcedErrorSink); ok {
			return next.WriteErrorWithSource(ctx, refID, status, err, source)
		}
		if err != nil && source.IsValid() {
			err = NewErrorWithSource(err, source)
		}
		return w.ChunkedDataWriter.WriteError(ctx, refID, status, err)
	})
}

// The wire carries error text. Keep that text for logs and use the same
// cancellation rule as RequestStatusFromProtoQueryDataResponse.
type rawQueryError string

func (e rawQueryError) Error() string { return string(e) }
func (e rawQueryError) Is(target error) bool {
	return target == context.Canceled && RequestStatusFromErrorString(string(e)) == RequestStatusCancelled
}

func (w *rawQueryObserver) OnChunk(c *pluginv2.QueryChunkedDataResponse) error {
	if c.GetError() != "" {
		return w.recordError(c.RefId, Status(c.Status), rawQueryError(c.Error), ErrorSource(c.ErrorSource), func() error {
			return w.raw.OnChunk(c)
		})
	}
	w.markRef(c.GetRefId())
	return w.raw.OnChunk(c)
}
