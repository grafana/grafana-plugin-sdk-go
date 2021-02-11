package data

import (
	"fmt"
	"time"

	jsoniter "github.com/json-iterator/go"
)

// FrameToJSON writes a frame to JSON
func FrameToJSON(frame *Frame, includeSchema bool, includeData bool) ([]byte, error) {
	cfg := jsoniter.ConfigCompatibleWithStandardLibrary
	stream := cfg.BorrowStream(nil)
	defer cfg.ReturnStream(stream)

	writeDataFrame(frame, stream, includeSchema, includeData)

	if stream.Error != nil {
		fmt.Println("error:", stream.Error)
		return nil, stream.Error
	}
	return stream.Buffer(), nil
}

func getSimpleTypeString(t FieldType) (string, bool) {
	if t.Time() {
		return "time", true
	}
	if t.Numeric() {
		return "number", true
	}
	if t == FieldTypeBool || t == FieldTypeNullableBool {
		return "bool", true
	}
	if t == FieldTypeString || t == FieldTypeNullableString {
		return "string", true
	}

	return "", false
}

// export interface FieldValueEntityLookup {
// 	NaN?: number[];
// 	Undef?: number[]; // Missing because of absense or join
// 	Inf?: number[];
// 	NegInf?: number[];
//   }

type fieldEntityLookup struct {
	NaN    []int `json:"NaN,omitempty"`
	Inf    []int `json:"Inf,omitempty"`
	NegInf []int `json:"NegInf,omitempty"`
}

func (f *fieldEntityLookup) add(str string, idx int) {
	switch str {
	case "+Inf":
		f.Inf = append(f.Inf, idx)
	case "-Inf":
		f.NegInf = append(f.NegInf, idx)
	case "NaN":
		f.NaN = append(f.NaN, idx)
	}
}

func writeDataFrame(frame *Frame, stream *jsoniter.Stream, includeSchema bool, includeData bool) error {
	started := false
	stream.WriteObjectStart()
	if includeSchema {
		stream.WriteObjectField("schema")
		stream.WriteObjectStart()

		if len(frame.Name) > 0 {
			stream.WriteObjectField("name")
			stream.WriteString(frame.Name)
			started = true
		}

		if len(frame.RefID) > 0 {
			if started {
				stream.WriteMore()
			}
			stream.WriteObjectField("refId")
			stream.WriteString(frame.RefID)
			started = true
		}

		if frame.Meta != nil {
			if started {
				stream.WriteMore()
			}
			stream.WriteObjectField("meta")
			stream.WriteVal(frame.Meta)
			started = true
		}

		if started {
			stream.WriteMore()
		}
		stream.WriteObjectField("fields")
		stream.WriteArrayStart()
		for i, f := range frame.Fields {
			if i > 0 {
				stream.WriteMore()
			}
			started = false
			stream.WriteObjectStart()
			if len(f.Name) > 0 {
				stream.WriteObjectField("name")
				stream.WriteString(f.Name)
				started = true
			}

			t, ok := getSimpleTypeString(f.Type())
			if ok {
				if started {
					stream.WriteMore()
				}
				stream.WriteObjectField("type")
				stream.WriteString(t)
				started = true
			}

			if f.Config != nil {
				if started {
					stream.WriteMore()
				}
				stream.WriteObjectField("config")
				stream.WriteVal(f.Config)
				started = true
			}

			stream.WriteObjectEnd()
		}
		stream.WriteArrayEnd()

		stream.WriteObjectEnd()
	}

	if includeData {
		if includeSchema {
			stream.WriteMore()
		}

		rowCount, err := frame.RowLen()
		if err != nil {
			return err
		}

		stream.WriteObjectField("data")
		stream.WriteObjectStart()

		entities := make([]*fieldEntityLookup, len(frame.Fields))
		entityCount := 0

		stream.WriteObjectField("fields")
		stream.WriteArrayStart()
		for fidx, f := range frame.Fields {
			if fidx > 0 {
				stream.WriteMore()
			}
			isTime := f.Type().Time()

			stream.WriteArrayStart()
			for i := 0; i < rowCount; i++ {
				if i > 0 {
					stream.WriteRaw(",")
				}
				v, ok := f.ConcreteAt(i)
				if ok {
					if isTime {
						v = v.(time.Time).UnixNano() / int64(time.Millisecond) // Millisconds precision
					}

					stream.WriteVal(v)
					if stream.Error != nil { // NaN +Inf/-Inf
						txt := fmt.Sprintf("%v", v)
						if entities[fidx] == nil {
							entities[fidx] = &fieldEntityLookup{}
						}
						entities[fidx].add(txt, i)
						entityCount++
						stream.Error = nil
						stream.WriteNil()
					}
				} else {
					stream.WriteNil()
				}
			}
			stream.WriteArrayEnd()
		}
		stream.WriteArrayEnd()

		if entityCount > 0 {
			if started {
				stream.WriteMore()
			}
			stream.WriteObjectField("entities")
			stream.WriteVal(entities)
		}

		stream.WriteObjectEnd()
	}
	stream.WriteObjectEnd()
	return nil
}
