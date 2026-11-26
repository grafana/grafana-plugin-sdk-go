package backend

import (
	"slices"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
	grpcproto "google.golang.org/grpc/encoding/proto"
	"google.golang.org/grpc/mem"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// Field numbers in backend.proto that queryDataCodec writes itself.
const (
	queryDataResponsesField = 1
	mapEntryKeyField        = 1
	mapEntryValueField      = 2
	dataResponseFramesField = 1
)

// queryDataCodec is the plugin's gRPC server codec. It writes a QueryDataResponse from the frame bytes that the message
// already holds, so the payload is not copied, and uses the standard proto codec for every other message.
type queryDataCodec struct {
	base encoding.CodecV2
}

func newQueryDataCodec() queryDataCodec {
	return queryDataCodec{base: encoding.GetCodecV2(grpcproto.Name)}
}

func queryDataCodecOption() grpc.ServerOption {
	return grpc.ForceServerCodecV2(newQueryDataCodec())
}

func (c queryDataCodec) Name() string { return grpcproto.Name }

func (c queryDataCodec) Unmarshal(data mem.BufferSlice, v any) error {
	return c.base.Unmarshal(data, v)
}

func (c queryDataCodec) Marshal(v any) (mem.BufferSlice, error) {
	if m, ok := v.(*pluginv2.QueryDataResponse); ok {
		return marshalQueryDataResponse(m)
	}
	return c.base.Marshal(v)
}

// marshalQueryDataResponse writes the same bytes as a deterministic proto.Marshal of m. The frames come first because
// they are field 1 of DataResponse, so each frame is written as a reference to its bytes.
func marshalQueryDataResponse(m *pluginv2.QueryDataResponse) (mem.BufferSlice, error) {
	refIDs := make([]string, 0, len(m.Responses))
	for refID := range m.Responses {
		refIDs = append(refIDs, refID)
	}
	slices.Sort(refIDs)

	count := 0
	for _, dr := range m.Responses {
		count += 2 + 2*len(dr.GetFrames())
	}
	// One backing array holds every buffer, so the message costs two allocations and not two per frame.
	bufs := make([]mem.SliceBuffer, 0, count)
	for _, refID := range refIDs {
		dr := m.Responses[refID]
		rest, err := proto.MarshalOptions{Deterministic: true}.Marshal(withoutFrames(dr))
		if err != nil {
			return nil, err
		}
		framesLen, headersLen := 0, 0
		for _, f := range dr.GetFrames() {
			h := protowire.SizeTag(dataResponseFramesField) + protowire.SizeVarint(uint64(len(f)))
			framesLen += h + len(f)
			headersLen += h
		}
		drLen := framesLen + len(rest)
		entryLen := protowire.SizeTag(mapEntryKeyField) + protowire.SizeBytes(len(refID)) +
			protowire.SizeTag(mapEntryValueField) + protowire.SizeVarint(uint64(drLen)) + drLen

		var head []byte
		head = protowire.AppendTag(head, queryDataResponsesField, protowire.BytesType)
		head = protowire.AppendVarint(head, uint64(entryLen)) // #nosec G115 -- entryLen is a sum of lengths, never negative
		head = protowire.AppendTag(head, mapEntryKeyField, protowire.BytesType)
		head = protowire.AppendString(head, refID)
		head = protowire.AppendTag(head, mapEntryValueField, protowire.BytesType)
		head = protowire.AppendVarint(head, uint64(drLen))
		bufs = append(bufs, head)

		headers := make([]byte, 0, headersLen)
		for _, f := range dr.GetFrames() {
			start := len(headers)
			headers = protowire.AppendTag(headers, dataResponseFramesField, protowire.BytesType)
			headers = protowire.AppendVarint(headers, uint64(len(f)))
			bufs = append(bufs, headers[start:len(headers):len(headers)])
			if len(f) > 0 {
				bufs = append(bufs, f)
			}
		}
		if len(rest) > 0 {
			bufs = append(bufs, rest)
		}
	}
	out := make(mem.BufferSlice, len(bufs))
	for i := range bufs {
		out[i] = &bufs[i]
	}
	return out, nil
}

// withoutFrames returns a shallow copy of dr with every field except frames.
func withoutFrames(dr *pluginv2.DataResponse) proto.Message {
	src := dr.ProtoReflect()
	dst := src.New()
	src.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if fd.Number() != dataResponseFramesField {
			dst.Set(fd, v)
		}
		return true
	})
	return dst.Interface()
}
