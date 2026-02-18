package backend

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// Experimental: ChunkedDataCallback is called with each response
type ChunkedDataCallback = func(chunk *pluginv2.QueryChunkedDataResponse) error

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

	// Allow clients direct access to the raw response
	// Implementing this method in a client can avoid an additional encode/decode cycle
	// When writing a datasource plugin (server), this method should not be used
	WriteChunk(chunk *pluginv2.QueryChunkedDataResponse) error
}

func NewChunkedDataWriter(format DataFrameFormat, write ChunkedDataCallback) ChunkedDataWriter {
	return &chunkedDataWriter{format: format, write: write}
}

type chunkedDataWriter struct {
	mu     sync.Mutex
	sent   map[string]bool
	format DataFrameFormat
	write  ChunkedDataCallback
}

// WriteError implements [ChunkedDataWriter].
func (c *chunkedDataWriter) WriteError(ctx context.Context, refID string, status Status, err error) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	chunk := &pluginv2.QueryChunkedDataResponse{
		RefId:  refID,
		Status: int32(status), //nolint:gosec // disable G115
	}
	if err != nil {
		chunk.Error = err.Error()
	}
	return c.write(chunk)
}

// WriteFrame implements [ChunkedDataWriter].
func (c *chunkedDataWriter) WriteFrame(ctx context.Context, refID string, frameID string, f *data.Frame) (err error) {
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
		Format:  pluginv2.DataFrameFormat(c.format),
	}

	switch c.format {
	case DataFrameFormat_ARROW:
		chunk.Frame, err = f.MarshalArrow()

	case DataFrameFormat_JSON:
		if c.sent == nil {
			c.sent = make(map[string]bool)
		}
		// Only send the schema the first time we see the frame identifier
		include := data.IncludeDataOnly
		key := fmt.Sprintf("%s-%s", refID, frameID)
		if !c.sent[key] {
			include = data.IncludeAll
			c.sent[key] = true
		}
		chunk.Frame, err = data.FrameToJSON(f, include)
	}

	if err != nil {
		return err
	}

	return c.write(chunk)
}

// WriteFrame implements [ChunkedDataWriter].
func (c *chunkedDataWriter) WriteChunk(chunk *pluginv2.QueryChunkedDataResponse) error {
	return c.write(chunk)
}
