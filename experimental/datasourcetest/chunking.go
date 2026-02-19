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

func AccumulateArrow(chunks iter.Seq2[*pluginv2.QueryChunkedDataResponse, error]) (*backend.QueryDataResponse, error) {
	// A refID identifies a response, while frameID identifies frames within it.
	// Frames with identical frameIDs represent chunked data and should be merged.
	// Frames with unique frameIDs represent distinct data and should be appended.
	type refState struct {
		dr      *backend.DataResponse
		frameID string
		frame   *data.Frame
	}

	responses := backend.Responses{}
	states := make(map[string]*refState)

	for chunk, err := range chunks {
		if err != nil {
			return nil, err
		}

		if chunk.Format != pluginv2.DataFrameFormat_ARROW {
			return nil, fmt.Errorf("expected arrow format")
		}

		f, err := data.UnmarshalArrowFrame(chunk.Frame)
		if err != nil {
			return nil, err
		}

		// First time we see this response?
		st, ok := states[chunk.RefId]
		if !ok {
			dr := &backend.DataResponse{
				Frames: data.Frames{f},
				Status: backend.Status(chunk.Status),
			}
			if chunk.Error != "" {
				dr.Error = errors.New(chunk.Error)
				dr.ErrorSource = backend.ErrorSource(chunk.ErrorSource)
			}

			st = &refState{
				dr:      dr,
				frameID: chunk.FrameId,
				frame:   f,
			}
			states[chunk.RefId] = st

			// Store a value copy for the final response map.
			responses[chunk.RefId] = *dr
			continue
		}

		// Frames with identical frameIDs represent chunked data and should be merged.
		if chunk.FrameId == st.frameID {
			if len(f.Fields) != len(st.frame.Fields) {
				return nil, errors.New("received chunked frame with mismatched field count")
			}
			for i, field := range f.Fields {
				st.frame.Fields[i].AppendAll(field)
			}
			continue
		}

		// Frames with unique frameIDs represent distinct data and should be appended.
		st.dr.Frames = append(st.dr.Frames, f)
		st.frameID = chunk.FrameId
		st.frame = f

		// Store a value copy for the final response map.
		responses[chunk.RefId] = *st.dr
	}

	return &backend.QueryDataResponse{Responses: responses}, nil
}

func AccumulateJSON(chunks iter.Seq2[*pluginv2.QueryChunkedDataResponse, error]) (*backend.QueryDataResponse, error) {
	responses := make(backend.Responses)
	frameByKey := make(map[string]*data.Frame)
	var frame *data.Frame

	for chunk, err := range chunks {
		if err != nil {
			return nil, err
		}

		if chunk.Format != pluginv2.DataFrameFormat_JSON {
			return nil, fmt.Errorf("expected json format")
		}

		rsp := responses[chunk.RefId]
		if len(chunk.Frame) > 0 {
			key := fmt.Sprintf("%s|%s", chunk.RefId, chunk.FrameId)
			frame = frameByKey[key]
			if frame != nil {
				if err = data.AppendJSONData(frame, chunk.Frame); err != nil {
					return nil, fmt.Errorf("error appending data %w", err)
				}
			} else {
				frame = &data.Frame{}
				if err = json.Unmarshal(chunk.Frame, frame); err != nil {
					return nil, fmt.Errorf("error parsing response %w", err)
				}
				frameByKey[key] = frame
				rsp.Frames = append(rsp.Frames, frame)
			}
		}

		rsp.Status = backend.Status(chunk.Status)
		if chunk.Error != "" {
			rsp.Error = errors.New(chunk.Error)
		}
		if chunk.ErrorSource != "" {
			rsp.ErrorSource = backend.ErrorSource(chunk.ErrorSource)
		}
		responses[chunk.RefId] = rsp
	}
	return &backend.QueryDataResponse{Responses: responses}, nil
}
