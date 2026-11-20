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
	// The frameID identifies one logical frame within the refID, and it must not be empty.
	// The first call with a frameID sends the schema, the metadata and the rows.
	// Later calls with the same frameID append their rows to that frame, so they must have
	// the same fields. The receiver ignores the name and the metadata of later calls.
	// Calls for different frameIDs can alternate.
	// To add fields to a frame, write a new frameID with all the fields, and set
	// f.Meta.Continues in its first call. The earlier frameID then takes no more calls.
	WriteFrame(ctx context.Context, refID string, frameID string, f *data.Frame) error

	// WriteError writes an error associated with the specified refID.
	// The frames that were written before the error stay in the response.
	WriteError(ctx context.Context, refID string, status Status, err error) error
}

func NewChunkedDataWriter(format DataFrameFormat, write ChunkedDataCallback) ChunkedDataWriter {
	return &chunkedDataWriter{format: format, write: write}
}

type chunkedDataWriter struct {
	mu   sync.Mutex
	sent map[sentKey]bool
	// continued has the frame ID that continued each frame.
	continued map[sentKey]string
	format    DataFrameFormat
	write     ChunkedDataCallback
}

// sentKey identifies a frame that the writer already sent. A joined string can collide,
// for example ("a-b", "c") and ("a", "b-c").
type sentKey struct {
	refID   string
	frameID string
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
		chunk.ErrorSource = string(ErrorSourceFromError(err))
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

	key := sentKey{refID: refID, frameID: frameID}
	if by, ok := c.continued[key]; ok {
		return fmt.Errorf("frame %q of refID %q was continued by frame %q, so it takes no more rows", frameID, refID, by)
	}
	first := !c.sent[key]
	var continues *data.FrameContinuation
	if first && f.Meta != nil {
		continues = f.Meta.Continues
	}
	if continues != nil {
		earlier := sentKey{refID: refID, frameID: continues.FrameID}
		if !c.sent[earlier] {
			return fmt.Errorf("frame %q of refID %q continues frame %q, which was not written", frameID, refID, continues.FrameID)
		}
		if by, ok := c.continued[earlier]; ok {
			return fmt.Errorf("frame %q of refID %q continues frame %q, which frame %q already continued", frameID, refID, continues.FrameID, by)
		}
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
		// Only send the schema the first time we see the frame identifier
		include := data.IncludeDataOnly
		if first {
			include = data.IncludeAll
		}
		chunk.Frame, err = data.FrameToJSON(f, include)
	}

	if err != nil {
		return err
	}

	// The state changes only after the write, so that a failed write can be sent again with the schema.
	if err := c.write(chunk); err != nil {
		return err
	}
	if c.sent == nil {
		c.sent = make(map[sentKey]bool)
	}
	c.sent[key] = true
	if continues != nil {
		if c.continued == nil {
			c.continued = make(map[sentKey]string)
		}
		c.continued[sentKey{refID: refID, frameID: continues.FrameID}] = frameID
	}
	return nil
}
