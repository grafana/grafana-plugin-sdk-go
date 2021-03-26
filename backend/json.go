package backend

import (
	"unsafe"

	jsoniter "github.com/json-iterator/go"
)

func init() { //nolint:gochecknoinits
	jsoniter.RegisterTypeEncoder("backend.QueryDataResults", &dataQueryResultsCodec{})
	jsoniter.RegisterTypeEncoder("backend.DataResponse", &dataResponseCodec{})
	jsoniter.RegisterTypeEncoder("backend.QueryDataResponse", &queryDataResponseCodec{})
}

// QueryDataResults is a flavor of
type QueryDataResults struct {
	Order   []string
	Results Responses
}

// MarshalJSON writes the results as json
func (r QueryDataResults) MarshalJSON() ([]byte, error) {
	cfg := jsoniter.ConfigCompatibleWithStandardLibrary
	stream := cfg.BorrowStream(nil)
	defer cfg.ReturnStream(stream)

	writeQueryDataResultsJSON(&r, stream)
	return stream.Buffer(), stream.Error
}

type dataQueryResultsCodec struct{}

func (codec *dataQueryResultsCodec) IsEmpty(ptr unsafe.Pointer) bool {
	qdr := (*QueryDataResults)(ptr)
	return qdr.Results == nil
}

func (codec *dataQueryResultsCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	qdr := (*QueryDataResults)(ptr)
	writeQueryDataResultsJSON(qdr, stream)
}

type dataResponseCodec struct{}

func (codec *dataResponseCodec) IsEmpty(ptr unsafe.Pointer) bool {
	dr := (*DataResponse)(ptr)
	return dr.Error == nil && dr.Frames == nil
}

func (codec *dataResponseCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	dr := (*DataResponse)(ptr)
	writeDataResponseJSON(dr, "", stream)
}

type queryDataResponseCodec struct{}

func (codec *queryDataResponseCodec) IsEmpty(ptr unsafe.Pointer) bool {
	qdr := *((*QueryDataResponse)(ptr))
	return qdr.Responses == nil
}

func (codec *queryDataResponseCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	qdr := (*QueryDataResponse)(ptr)
	r := &QueryDataResults{
		Results: qdr.Responses,
	}
	writeQueryDataResultsJSON(r, stream)
}

//-----------------------------------------------------------------
// Private stream readers
//-----------------------------------------------------------------

func writeDataResponseJSON(dr *DataResponse, refID string, stream *jsoniter.Stream) {
	stream.WriteObjectStart()
	started := false

	if refID != "" {
		stream.WriteObjectField("refId")
		stream.WriteString(refID)
		started = true
	}

	if dr.Error != nil {
		if started {
			stream.WriteMore()
		}
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

func writeQueryDataResultsJSON(qdr *QueryDataResults, stream *jsoniter.Stream) {
	stream.WriteObjectStart()
	if qdr.Results != nil {
		stream.WriteObjectField("results")
		stream.WriteArrayStart()
		started := false
		if len(qdr.Order) > 0 {
			for _, id := range qdr.Order {
				if started {
					stream.WriteMore()
				}
				res, ok := qdr.Results[id]
				if ok {
					writeDataResponseJSON(&res, id, stream)
					started = true
				}
			}
		} else {
			for id, res := range qdr.Results {
				if started {
					stream.WriteMore()
				}
				obj := res // avoid implicit memory
				writeDataResponseJSON(&obj, id, stream)
				started = true
			}
		}
		stream.WriteArrayEnd()
	}
	stream.WriteObjectEnd()
}
