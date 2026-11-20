package datasourcetest

import (
	"encoding/json"
	"errors"
	"fmt"
	"iter"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// Experimental: AccumulateArrow collects a stream of Arrow chunks into one QueryDataResponse.
// It applies the frame ID rule of QueryChunkedDataResponse in proto/backend.proto.
func AccumulateArrow(chunks iter.Seq2[*pluginv2.QueryChunkedDataResponse, error]) (*backend.QueryDataResponse, error) {
	return accumulate(chunks, pluginv2.DataFrameFormat_ARROW, data.UnmarshalArrowFrame, appendArrowRows)
}

// Experimental: AccumulateJSON collects a stream of JSON chunks into one QueryDataResponse.
// It applies the frame ID rule of QueryChunkedDataResponse in proto/backend.proto.
func AccumulateJSON(chunks iter.Seq2[*pluginv2.QueryChunkedDataResponse, error]) (*backend.QueryDataResponse, error) {
	return accumulate(chunks, pluginv2.DataFrameFormat_JSON, unmarshalJSONFrame, appendJSONRows)
}

func accumulate(
	chunks iter.Seq2[*pluginv2.QueryChunkedDataResponse, error],
	format pluginv2.DataFrameFormat,
	decode func([]byte) (*data.Frame, error),
	appendRows func(*data.Frame, []byte) error,
) (*backend.QueryDataResponse, error) {
	// The frame ID is unique only within its refID.
	type frameKey struct {
		refID   string
		frameID string
	}

	responses := backend.Responses{}
	frames := make(map[frameKey]*data.Frame)
	// continued has the frame ID that continued each frame. A continued frame takes no more chunks.
	continued := make(map[frameKey]string)

	for chunk, err := range chunks {
		if err != nil {
			return nil, err
		}

		rsp, ok := responses[chunk.RefId]
		if !ok {
			rsp.Status = backend.StatusOK
		}
		// A new error replaces the earlier error with its own status and error source, and an error without a status
		// means 500. Only a chunk with an error can replace a status that is not 2xx, so a frame chunk with 200 keeps
		// an earlier failure.
		if chunk.Error != "" {
			rsp.Error = errors.New(chunk.Error)
			rsp.ErrorSource = backend.ErrorSource(chunk.ErrorSource)
			rsp.Status = backend.Status(chunk.Status)
			if chunk.Status == 0 {
				rsp.Status = backend.StatusInternal
			}
		} else if chunk.Status != 0 && rsp.Status >= 200 && rsp.Status < 300 {
			rsp.Status = backend.Status(chunk.Status)
		}

		// A chunk from WriteError has no frame, and its format is not set.
		if len(chunk.Frame) > 0 {
			if chunk.Format != format {
				return nil, fmt.Errorf("expected %s format, got %s", format, chunk.Format)
			}

			key := frameKey{refID: chunk.RefId, frameID: chunk.FrameId}
			if by, ok := continued[key]; ok {
				return nil, fmt.Errorf("frame %q of refID %q was continued by frame %q, so it takes no more chunks", chunk.FrameId, chunk.RefId, by)
			}
			if frame, ok := frames[key]; ok {
				if err := appendRows(frame, chunk.Frame); err != nil {
					return nil, fmt.Errorf("appending rows to frame %q of refID %q: %w", chunk.FrameId, chunk.RefId, err)
				}
			} else {
				frame, err := decode(chunk.Frame)
				if err != nil {
					return nil, err
				}
				if frame.Meta != nil && frame.Meta.Continues != nil {
					earlierID := frame.Meta.Continues.FrameID
					earlierKey := frameKey{refID: chunk.RefId, frameID: earlierID}
					if by, ok := continued[earlierKey]; ok {
						return nil, fmt.Errorf("frame %q of refID %q continues frame %q, which frame %q already continued", chunk.FrameId, chunk.RefId, earlierID, by)
					}
					earlier, ok := frames[earlierKey]
					if !ok {
						return nil, fmt.Errorf("frame %q of refID %q continues the unknown frame %q", chunk.FrameId, chunk.RefId, earlierID)
					}
					if err := data.JoinContinuation(earlier, frame); err != nil {
						return nil, fmt.Errorf("joining frame %q of refID %q to frame %q: %w", chunk.FrameId, chunk.RefId, earlierID, err)
					}
					continued[earlierKey] = chunk.FrameId
					// Later chunks of this frame ID append to the joined frame, which is already in the response.
					frame = earlier
				} else {
					rsp.Frames = append(rsp.Frames, frame)
				}
				// Without a frame ID, later chunks cannot refer to this frame.
				if chunk.FrameId != "" {
					frames[key] = frame
				}
			}
		}

		responses[chunk.RefId] = rsp
	}

	return &backend.QueryDataResponse{Responses: responses}, nil
}

// appendArrowRows appends the rows of an Arrow chunk to frame. Each Arrow chunk has its own schema, so the
// fields must match those of the first chunk. The name and the meta of the chunk are ignored.
func appendArrowRows(frame *data.Frame, b []byte) error {
	chunk, err := data.UnmarshalArrowFrame(b)
	if err != nil {
		return err
	}
	if len(chunk.Fields) != len(frame.Fields) {
		return fmt.Errorf("chunk has %d fields, frame has %d", len(chunk.Fields), len(frame.Fields))
	}
	for i, field := range chunk.Fields {
		if field.Type() != frame.Fields[i].Type() {
			return fmt.Errorf("field %d has type %s in the chunk and %s in the frame", i, field.Type(), frame.Fields[i].Type())
		}
	}
	for i, field := range chunk.Fields {
		frame.Fields[i].AppendAll(field)
	}
	return nil
}

func unmarshalJSONFrame(b []byte) (*data.Frame, error) {
	frame := &data.Frame{}
	if err := json.Unmarshal(b, frame); err != nil {
		return nil, fmt.Errorf("error parsing response %w", err)
	}
	return frame, nil
}

// appendJSONRows appends the rows of a JSON chunk to frame. The chunk has only data, so a chunk with too few
// columns shows up only as fields of different lengths.
func appendJSONRows(frame *data.Frame, b []byte) error {
	if err := data.AppendJSONData(frame, b); err != nil {
		return err
	}
	_, err := frame.RowLen()
	return err
}
