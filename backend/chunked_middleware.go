package backend

import (
	"context"
	"sync"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

type rawChunkSink interface {
	OnChunk(*pluginv2.QueryChunkedDataResponse) error
}

type observeWrite func(refID string, status Status, err error, source ErrorSource)

type observedChunkWriter struct {
	ChunkedDataWriter
	observe observeWrite
	errorMu sync.Mutex
}

type rawQueryObserver struct {
	*observedChunkWriter
	raw rawChunkSink
}

func observeChunkedWrites(w ChunkedDataWriter, observe observeWrite) ChunkedDataWriter {
	writer := &observedChunkWriter{ChunkedDataWriter: w, observe: observe}
	// A host can select a different transport when the writer accepts encoded chunks.
	if raw, ok := w.(rawChunkSink); ok {
		return &rawQueryObserver{writer, raw}
	}
	return writer
}

// ObserveChunkedErrors calls onError before forwarding each typed or encoded error to w.
// The callback receives the supplied status and the error source, if known.
// It runs once per error write, even if forwarding fails. Error writes run in delivery order.
// The callback must not write another error through the returned writer.
// The returned writer supports OnChunk only when w supports it.
func ObserveChunkedErrors(w ChunkedDataWriter, onError func(refID string, status Status, err error, source ErrorSource)) ChunkedDataWriter {
	return observeChunkedWrites(w, func(refID string, status Status, err error, source ErrorSource) {
		if err == nil {
			return
		}
		if !source.IsValid() {
			source = ErrorSourceFromError(err)
		}
		onError(refID, status, err, source)
	})
}

type chunkedQueryErrors struct {
	result QueryDataResponse
}

func recordChunkedErrors(w ChunkedDataWriter) (*chunkedQueryErrors, ChunkedDataWriter) {
	errs := &chunkedQueryErrors{}
	return errs, ObserveChunkedErrors(w, errs.record)
}

func (r *chunkedQueryErrors) record(refID string, status Status, err error, source ErrorSource) {
	if r.result.Responses == nil {
		r.result.Responses = make(Responses)
	}
	if !status.IsValid() {
		status = statusFromError(err)
	}
	r.result.Responses[refID] = DataResponse{Error: err, Status: status, ErrorSource: source}
}

// Read the result only after the handler returns and all writes finish.
func (r *chunkedQueryErrors) requestStatus(err error) RequestStatus {
	return RequestStatusFromQueryDataResponse(&r.result, err)
}

func (w *observedChunkWriter) WriteFrame(ctx context.Context, refID, frameID string, f *data.Frame) error {
	w.observe(refID, 0, nil, "")
	return w.ChunkedDataWriter.WriteFrame(ctx, refID, frameID, f)
}

func (w *observedChunkWriter) WriteError(ctx context.Context, refID string, status Status, err error) error {
	if err != nil {
		// Keep error observations in delivery order, including calls to OnChunk.
		w.errorMu.Lock()
		defer w.errorMu.Unlock()
	}
	w.observe(refID, status, err, "")
	return w.ChunkedDataWriter.WriteError(ctx, refID, status, err)
}

// The wire carries error text. Keep that text for logs and use the same
// cancellation rule as RequestStatusFromProtoQueryDataResponse.
type rawQueryError string

func (e rawQueryError) Error() string { return string(e) }
func (e rawQueryError) Is(target error) bool {
	return target == context.Canceled && RequestStatusFromErrorString(string(e)) == RequestStatusCancelled
}

func (w *rawQueryObserver) OnChunk(c *pluginv2.QueryChunkedDataResponse) error {
	var err error
	if c.GetError() != "" {
		w.errorMu.Lock()
		defer w.errorMu.Unlock()
		err = rawQueryError(c.Error)
	}
	w.observe(c.GetRefId(), Status(c.GetStatus()), err, ErrorSource(c.GetErrorSource()))
	return w.raw.OnChunk(c)
}
