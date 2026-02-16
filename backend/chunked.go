package backend

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

type ChunkedDataCallback = func(evt *pluginv2.QueryChunkedDataResponse) error

// Experimental: QueryChunkedQueryRawClient allows raw access to the chunked results
type QueryChunkedQueryRawClient interface {
	QueryChunkedRaw(ctx context.Context, req *QueryChunkedDataRequest, cb ChunkedDataCallback) error
}

// Experimental: ChunkedDataWriter defines the interface for writing data frames and errors
// back to the client in chunks.
type ChunkedDataWriter interface {
	// WriteFrame writes a data frame (f) for the given query refID.
	// The first time the frameID is written, the metadata and rows will be included.
	// Subsequent calls with the same frameID will append the rows to the existing frame
	// with a matching frameID. The metadata structure must match the initial request.
	WriteFrame(ctx context.Context, refID string, frameID string, f *data.Frame) error

	// WriteError writes an error associated with the specified refID.
	WriteError(ctx context.Context, refID string, status Status, err error) error
}

var (
	_ ChunkedDataWriter = (*chunkedDataCallback)(nil)
)

type chunkedDataCallback struct {
	mu     sync.Mutex // thread safety
	cb     func(evt *pluginv2.QueryChunkedDataResponse) error
	sent   map[string]bool
	asJSON bool
}

func NewChunkedDataCallback(req *QueryChunkedDataRequest, cb func(evt *pluginv2.QueryChunkedDataResponse) error) ChunkedDataWriter {
	return &chunkedDataCallback{cb: cb, asJSON: false, sent: make(map[string]bool)}
}

// WriteError implements [ChunkedDataWriter].
func (c *chunkedDataCallback) WriteError(ctx context.Context, refID string, status Status, err error) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	chunk := &pluginv2.QueryChunkedDataResponse{
		RefId:  refID,
		Status: int32(status), //nolint:gosec // disable G115
	}
	if err != nil {
		chunk.Error = err.Error()
	}
	return c.cb(chunk)
}

// WriteFrame implements [ChunkedDataWriter].
func (c *chunkedDataCallback) WriteFrame(ctx context.Context, refID string, frameID string, f *data.Frame) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if refID == "" {
		return fmt.Errorf("missing refID identifier")
	}

	if frameID == "" {
		return fmt.Errorf("missing frame identifier")
	}

	f.SetRefID(refID)

	chunk := &pluginv2.QueryChunkedDataResponse{
		RefId:   refID,
		FrameId: frameID,
		Status:  http.StatusOK,
	}

	if c.asJSON {
		// Only send the schema the first time we see the frame identifier
		include := data.IncludeDataOnly
		key := fmt.Sprintf("%s-%s", refID, frameID)
		if !c.sent[key] {
			include = data.IncludeAll
			c.sent[key] = true
		}
		chunk.Frame, err = data.FrameToJSON(f, include)
	} else {
		chunk.Frame, err = f.MarshalArrow()
	}
	if err != nil {
		return err
	}

	return c.cb(chunk)
}
