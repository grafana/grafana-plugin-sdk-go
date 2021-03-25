package backend

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"
)

func init() { //nolint:gochecknoinits
	jsoniter.RegisterTypeEncoder("backend.DataResponse", &dataResponseCodec{})
	jsoniter.RegisterTypeEncoder("backend.QueryDataResponse", &queryDataResponseCodec{})
}

type dataResponseCodec struct{}

func (codec *dataResponseCodec) IsEmpty(ptr unsafe.Pointer) bool {
	dr := (*DataResponse)(ptr)
	return dr.Error == nil && dr.Frames == nil
}

func (codec *dataResponseCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	dr := (*DataResponse)(ptr)
	writeDataResponseJSON(dr, stream)
}

type queryDataResponseCodec struct{}

func (codec *queryDataResponseCodec) IsEmpty(ptr unsafe.Pointer) bool {
	qdr := *((*QueryDataResponse)(ptr))
	return qdr.Responses == nil
}

func (codec *queryDataResponseCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	qdr := (*QueryDataResponse)(ptr)
	writeQueryDataResponseJSON(qdr, stream)
}

//-----------------------------------------------------------------
// Private stream readers
//-----------------------------------------------------------------

func writeDataResponseJSON(dr *DataResponse, stream *jsoniter.Stream) {
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

		started = false
		stream.WriteObjectField("frames")
		stream.WriteArrayStart()
		for _, frame := range dr.Frames {
			if started {
				stream.WriteMore()
			}
			stream.WriteVal(frame)
			started = true
		}
		stream.WriteArrayEnd()
	}

	stream.WriteObjectEnd()
}

func writeQueryDataResponseJSON(qdr *QueryDataResponse, stream *jsoniter.Stream) {
	stream.WriteObjectStart()
	if qdr.Responses != nil {
		stream.WriteObjectField("responses")
		stream.WriteObjectStart()
		started := false
		for id, res := range qdr.Responses {
			if started {
				stream.WriteMore()
			}
			stream.WriteObjectField(id)
			stream.WriteVal(res)
			started = true
		}
		stream.WriteObjectEnd()
	}
	stream.WriteObjectEnd()
}
