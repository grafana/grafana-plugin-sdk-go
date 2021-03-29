package backend

import (
	"fmt"
	"unsafe"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	jsoniter "github.com/json-iterator/go"
)

func init() { //nolint:gochecknoinits
	jsoniter.RegisterTypeEncoder("backend.QueryDataResults", &dataQueryResultsCodec{})
	jsoniter.RegisterTypeEncoder("backend.DataResponse", &dataResponseCodec{})
	jsoniter.RegisterTypeEncoder("backend.QueryDataResponse", &queryDataResponseCodec{})
}

// QueryDataResults adds key order as an input so that results can match query refId order
// This struct is not part of the gRPC API, rather a utility structure to help produce
// better looking JSON output
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

func (codec *dataQueryResultsCodec) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	qdr := QueryDataResults{}
	readQueryDataResultsJSON(&qdr, iter)
	*((*QueryDataResults)(ptr)) = qdr
}

// readQueryDataResultsJSON

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

func (codec *queryDataResponseCodec) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	qdr := QueryDataResults{}
	readQueryDataResultsJSON(&qdr, iter)
	*((*QueryDataResponse)(ptr)) = QueryDataResponse{
		Responses: qdr.Results,
	}
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
		wrote := make(map[string]struct{}, len(qdr.Results))
		stream.WriteObjectField("results")
		stream.WriteArrayStart()
		started := false
		if len(qdr.Order) > 0 {
			for _, id := range qdr.Order {
				_, ok := wrote[id]
				if ok {
					continue // already wrote that key
				}

				if started {
					stream.WriteMore()
				}
				res, ok := qdr.Results[id]
				if ok {
					writeDataResponseJSON(&res, id, stream)
					wrote[id] = struct{}{}
					started = true
				}
			}
		}

		// Make sure all keys in the result are written
		for id, res := range qdr.Results {
			_, ok := wrote[id]
			if ok {
				continue // already wrote that key
			}

			if started {
				stream.WriteMore()
			}
			obj := res // avoid implicit memory
			writeDataResponseJSON(&obj, id, stream)
			wrote[id] = struct{}{}
			started = true
		}
		stream.WriteArrayEnd()
	}
	stream.WriteObjectEnd()
}

//-----------------------------------------------------------------
// Private stream readers
//-----------------------------------------------------------------

func readQueryDataResultsJSON(qdr *QueryDataResults, iter *jsoniter.Iterator) {
	found := false

	for l1Field := iter.ReadObject(); l1Field != ""; l1Field = iter.ReadObject() {
		switch l1Field {
		case "results":
			if found {
				iter.ReportError("read results", "already found results")
				return
			}
			found = true

			qdr.Order = make([]string, 0)
			qdr.Results = make(Responses)

			for iter.ReadArray() {
				refID := ""
				res := DataResponse{}

				for l2Field := iter.ReadObject(); l2Field != ""; l2Field = iter.ReadObject() {
					switch l2Field {
					case "refId":
						refID = iter.ReadString()
						qdr.Order = append(qdr.Order, refID)

					case "error":
						res.Error = fmt.Errorf(iter.ReadString())

					case "frames":
						for iter.ReadArray() {
							frame := &data.Frame{}
							iter.ReadVal(frame)
							if iter.Error != nil {
								return
							}
							res.Frames = append(res.Frames, frame)
						}

					default:
						iter.ReportError("bind l2", "unexpected field: "+l1Field)
						return
					}
				}

				qdr.Results[refID] = res
			}

		default:
			iter.ReportError("bind l1", "unexpected field: "+l1Field)
			return
		}
	}
}
