package backend

import (
	"fmt"
	"sort"
	"unsafe"

	jsoniter "github.com/json-iterator/go"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	sdkjsoniter "github.com/grafana/grafana-plugin-sdk-go/data/utils/jsoniter"
)

func init() { //nolint:gochecknoinits
	sdkjsoniter.RegisterTypeEncoder("backend.DataResponse", &dataResponseCodec{})
	sdkjsoniter.RegisterTypeEncoder("backend.QueryDataResponse", &queryDataResponseCodec{})
}

type dataResponseCodec struct{}

func (codec *dataResponseCodec) IsEmpty(ptr unsafe.Pointer) bool {
	dr := (*DataResponse)(ptr)
	return dr.Error == nil && dr.Frames == nil
}

func (codec *dataResponseCodec) Encode(ptr unsafe.Pointer, stream *sdkjsoniter.Stream) {
	dr := (*DataResponse)(ptr)
	writeDataResponseJSON(dr, stream)
}

type queryDataResponseCodec struct{}

func (codec *queryDataResponseCodec) IsEmpty(ptr unsafe.Pointer) bool {
	qdr := *((*QueryDataResponse)(ptr))
	return qdr.Responses == nil
}

func (codec *queryDataResponseCodec) Encode(ptr unsafe.Pointer, stream *sdkjsoniter.Stream) {
	qdr := (*QueryDataResponse)(ptr)
	writeQueryDataResponseJSON(qdr, stream)
}

func (codec *queryDataResponseCodec) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	qdr := QueryDataResponse{}
	readQueryDataResultsJSON(&qdr, sdkjsoniter.NewIterator(iter))
	*((*QueryDataResponse)(ptr)) = qdr
}

// -----------------------------------------------------------------
// Private stream readers
// -----------------------------------------------------------------

func writeDataResponseJSON(dr *DataResponse, stream *sdkjsoniter.Stream) {
	stream.WriteObjectStart()
	started := false

	status := dr.Status

	if dr.Error != nil {
		stream.WriteObjectField("error")
		stream.WriteString(dr.Error.Error())
		started = true

		if !status.IsValid() {
			status = statusFromError(dr.Error)
		}

		stream.WriteMore()
		stream.WriteObjectField("errorSource")
		stream.WriteString(string(dr.ErrorSource))
	}

	if status.IsValid() || status == 0 {
		if started {
			stream.WriteMore()
		}
		stream.WriteObjectField("status")
		if status.IsValid() {
			stream.WriteInt32(int32(status))
		} else if status == 0 {
			stream.WriteInt32(int32(StatusOK))
		}
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

func writeQueryDataResponseJSON(qdr *QueryDataResponse, stream *sdkjsoniter.Stream) {
	stream.WriteObjectStart()
	stream.WriteObjectField("results")
	stream.WriteObjectStart()
	started := false

	refIDs := []string{}
	for refID := range qdr.Responses {
		refIDs = append(refIDs, refID)
	}
	sort.Strings(refIDs)

	// Make sure all keys in the result are written
	for _, refID := range refIDs {
		res := qdr.Responses[refID]

		if started {
			stream.WriteMore()
		}
		stream.WriteObjectField(refID)
		obj := res // avoid implicit memory
		writeDataResponseJSON(&obj, stream)
		started = true
	}
	stream.WriteObjectEnd()

	stream.WriteObjectEnd()
}

// -----------------------------------------------------------------
// Private stream readers
// -----------------------------------------------------------------

func readQueryDataResultsJSON(qdr *QueryDataResponse, iter *sdkjsoniter.Iterator) {
	found := false

	for l1Field, _ := iter.ReadObject(); l1Field != ""; l1Field, _ = iter.ReadObject() {
		switch l1Field {
		case "results":
			if found {
				_ = iter.ReportError("read results", "already found results")
				return
			}
			found = true

			qdr.Responses = make(Responses)

			for l2Field, _ := iter.ReadObject(); l2Field != ""; l2Field, _ = iter.ReadObject() {
				dr := DataResponse{}
				readDataResponseJSON(&dr, iter)
				qdr.Responses[l2Field] = dr
			}

		default:
			_ = iter.ReportError("bind l1", "unexpected field: "+l1Field)
			return
		}
	}
}

func readDataResponseJSON(rsp *DataResponse, iter *sdkjsoniter.Iterator) {
	for l2Field, _ := iter.ReadObject(); l2Field != ""; l2Field, _ = iter.ReadObject() {
		switch l2Field {
		case "error":
			rsp.Error = fmt.Errorf(iter.ReadString())

		case "status":
			s, _ := iter.ReadInt32()
			rsp.Status = Status(s)

		case "errorSource":
			src, _ := iter.ReadString()
			rsp.ErrorSource = ErrorSource(src)

		case "frames":
			for iter.CanReadArray() {
				frame := &data.Frame{}
				err := iter.ReadVal(frame)
				if err != nil {
					return
				}
				rsp.Frames = append(rsp.Frames, frame)
			}

		default:
			_ = iter.ReportError("bind l2", "unexpected field: "+l2Field)
			return
		}
	}
}
