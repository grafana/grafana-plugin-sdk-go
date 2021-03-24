package backend

import (
	"unsafe"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	jsoniter "github.com/json-iterator/go"
)

// This will make sure jsoniter uses a fast JSON serialization strategy
var (
	_ = initEncoders()
)

func initEncoders() struct{} {
	jsoniter.RegisterTypeEncoder("backend.DataResponse", &dataResponseCodec{})
	jsoniter.RegisterTypeEncoder("backend.QueryDataResponse", &queryDataResponseCodec{})
	return struct{}{} // 0 bytes
}

type dataResponseCodec struct{}

func (codec *dataResponseCodec) IsEmpty(ptr unsafe.Pointer) bool {
	dr := *((*DataResponse)(ptr))
	return dr.Error == nil && dr.Frames == nil
}

func (codec *dataResponseCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	dr := *((*DataResponse)(ptr))
	stream.WriteObjectStart()
	started := false
	if dr.Error != nil {
		stream.WriteObjectField("error")
		stream.WriteString(dr.Error.Error())
		started = true
	}

	if dr.Frames != nil {
		if started {
			stream.WriteMore()
		}

		stream.WriteObjectField("frames")
		stream.WriteArrayStart()
		for _, frame := range dr.Frames {
			err := data.WriteDataFrameJSON(frame, stream, data.WithSchmaAndData)
			if err != nil && stream.Error == nil {
				stream.Error = err
				return
			}
		}
		stream.WriteArrayEnd()
	}

	stream.WriteObjectEnd()
}

//------------------------------------------------------------
// QueryDataResponse
//------------------------------------------------------------

type queryDataResponseCodec struct{}

func (codec *queryDataResponseCodec) IsEmpty(ptr unsafe.Pointer) bool {
	qdr := *((*QueryDataResponse)(ptr))
	return qdr.Responses == nil
}

func (codec *queryDataResponseCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	qdr := *((*QueryDataResponse)(ptr))
	stream.WriteObjectStart()
	if qdr.Responses != nil {
		stream.WriteObjectField("responses")
		stream.WriteObjectStart()
		for id, res := range qdr.Responses {
			stream.WriteObjectField(id)
			stream.WriteVal(res)
		}
		stream.WriteObjectEnd()
	}
	stream.WriteObjectEnd()
}
