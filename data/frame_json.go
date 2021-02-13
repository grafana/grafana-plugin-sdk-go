package data

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/apache/arrow/go/arrow"
	"github.com/apache/arrow/go/arrow/array"
	"github.com/apache/arrow/go/arrow/ipc"
	jsoniter "github.com/json-iterator/go"
	"github.com/mattetti/filebuffer"
)

const simpleTypeString = "string"
const simpleTypeNumber = "number"
const simpleTypeBool = "bool"
const simpleTypeTime = "time"

// FrameToJSON writes a frame to JSON
func FrameToJSON(frame *Frame, includeSchema bool, includeData bool) ([]byte, error) {
	cfg := jsoniter.ConfigCompatibleWithStandardLibrary
	stream := cfg.BorrowStream(nil)
	defer cfg.ReturnStream(stream)

	err := writeDataFrame(frame, stream, includeSchema, includeData)
	if err != nil {
		return nil, err
	}

	if stream.Error != nil {
		fmt.Println("error:", stream.Error)
		return nil, stream.Error
	}
	return stream.Buffer(), nil
}

func getSimpleTypeString(t FieldType) (string, bool) {
	if t.Time() {
		return simpleTypeTime, true
	}
	if t.Numeric() {
		return simpleTypeNumber, true
	}
	if t == FieldTypeBool || t == FieldTypeNullableBool {
		return simpleTypeBool, true
	}
	if t == FieldTypeString || t == FieldTypeNullableString {
		return simpleTypeString, true
	}

	return "", false
}

func getSimpleTypeStringForArrow(t arrow.DataType) string {
	switch t.ID() {
	case arrow.TIMESTAMP:
		return simpleTypeTime
	case arrow.UINT8:
		fallthrough
	case arrow.UINT16:
		fallthrough
	case arrow.UINT32:
		fallthrough
	case arrow.UINT64:
		fallthrough
	case arrow.INT8:
		fallthrough
	case arrow.INT16:
		fallthrough
	case arrow.INT32:
		fallthrough
	case arrow.INT64:
		fallthrough
	case arrow.FLOAT32:
		fallthrough
	case arrow.FLOAT64:
		return simpleTypeNumber
	case arrow.STRING:
		return simpleTypeString
	case arrow.BOOL:
		return simpleTypeBool
	default:
		return ""
	}
}

// export interface FieldValueEntityLookup {
// 	NaN?: number[];
// 	Undef?: number[]; // Missing because of absence or join
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

			if f.Labels != nil {
				if started {
					stream.WriteMore()
				}
				stream.WriteObjectField("labels")
				stream.WriteVal(f.Labels)
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

// ArrowToJSON writes a frame to JSON
func ArrowBufferToJSON(b []byte) ([]byte, error) {
	fB := filebuffer.New(b)
	fR, err := ipc.NewFileReader(fB)
	if err != nil {
		return nil, err
	}
	defer fR.Close()

	record, err := fR.Read()
	if errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("No records found")
	}
	if err != nil {
		return nil, err
	}
	// TODO?? multiple records in one file?

	return ArrowToJSON(record, true, true)
}

// ArrowToJSON writes a frame to JSON
func ArrowToJSON(record array.Record, includeSchema bool, includeData bool) ([]byte, error) {
	cfg := jsoniter.ConfigCompatibleWithStandardLibrary
	stream := cfg.BorrowStream(nil)
	defer cfg.ReturnStream(stream)

	err := writeArrowFrame(stream, record, includeSchema, includeData)
	if err != nil {
		return nil, err
	}

	if stream.Error != nil {
		fmt.Println("error:", stream.Error)
		return nil, stream.Error
	}
	return stream.Buffer(), nil
}

func writeArrowFrame(stream *jsoniter.Stream, record array.Record, includeSchema bool, includeData bool) error {
	started := false
	stream.WriteObjectStart()
	if includeSchema {
		metaData := record.Schema().Metadata()

		stream.WriteObjectField("schema")
		stream.WriteObjectStart()

		name, _ := getMDKey("name", metaData) // No need to check ok, zero value ("") is returned
		refID, _ := getMDKey("refId", metaData)

		if len(name) > 0 {
			stream.WriteObjectField("name")
			stream.WriteString(name)
			started = true
		}

		if len(refID) > 0 {
			if started {
				stream.WriteMore()
			}
			stream.WriteObjectField("refId")
			stream.WriteString(refID)
			started = true
		}

		if metaAsString, ok := getMDKey("meta", metaData); ok {
			if started {
				stream.WriteMore()
			}
			stream.WriteObjectField("meta")
			stream.WriteRaw(metaAsString)
			started = true
		}

		if started {
			stream.WriteMore()
		}
		stream.WriteObjectField("fields")
		stream.WriteArrayStart()
		for i, f := range record.Schema().Fields() {
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

			if started {
				stream.WriteMore()
			}
			stream.WriteObjectField("type")
			stream.WriteString(getSimpleTypeStringForArrow(f.Type))

			if labelsAsString, ok := getMDKey("labels", f.Metadata); ok {
				stream.WriteMore()
				stream.WriteObjectField("labels")
				stream.WriteRaw(labelsAsString)
			}
			if labelsAsString, ok := getMDKey("config", f.Metadata); ok {
				stream.WriteMore()
				stream.WriteObjectField("config")
				stream.WriteRaw(labelsAsString)
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
		fieldCount := len(record.Schema().Fields())

		stream.WriteObjectField("data")
		stream.WriteObjectStart()

		entities := make([]*fieldEntityLookup, fieldCount)
		entityCount := 0

		//	rowCount := int(record.NumRows())

		stream.WriteObjectField("fields")
		stream.WriteArrayStart()
		for fidx := 0; fidx < fieldCount; fidx++ {
			if fidx > 0 {
				stream.WriteMore()
			}
			col := record.Column(fidx)
			var ent *fieldEntityLookup

			switch col.DataType().ID() {
			case arrow.TIMESTAMP:
				writeArrowDataTIMESTAMP(stream, col)

			case arrow.UINT8:
				ent = writeArrowDataUint8(stream, col)
			case arrow.UINT16:
				ent = writeArrowDataUint16(stream, col)
			case arrow.UINT32:
				ent = writeArrowDataUint32(stream, col)
			case arrow.UINT64:
				ent = writeArrowDataUint64(stream, col)
			case arrow.INT8:
				ent = writeArrowDataInt8(stream, col)
			case arrow.INT16:
				ent = writeArrowDataInt16(stream, col)
			case arrow.INT32:
				ent = writeArrowDataInt32(stream, col)
			case arrow.INT64:
				ent = writeArrowDataInt64(stream, col)
			case arrow.FLOAT32:
				ent = writeArrowDataFloat32(stream, col)
			case arrow.FLOAT64:
				ent = writeArrowDataFloat64(stream, col)
			case arrow.STRING:
				ent = writeArrowDataString(stream, col)
			case arrow.BOOL:
				ent = writeArrowDataBool(stream, col)
			default:
				return fmt.Errorf("unsupported arrow type %s for JSON", col.DataType().ID())
			}

			if ent != nil {
				entities[fidx] = ent
				entityCount++
			}
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

// Custom timestamp extraction... assumes nanoseconds for everything now
func writeArrowDataTIMESTAMP(stream *jsoniter.Stream, col array.Interface) {
	count := col.Len()

	v := array.NewTimestampData(col.Data())
	stream.WriteArrayStart()
	for i := 0; i < count; i++ {
		if i > 0 {
			stream.WriteRaw(",")
		}
		if col.IsNull(i) {
			stream.WriteNil()
			continue
		}
		ns := v.Value(i)
		ms := int64(ns) / int64(time.Millisecond) // nanosecond assumption
		stream.WriteInt64(ms)

		if stream.Error != nil { // ???
			stream.Error = nil
			stream.WriteNil()
		}
	}
	stream.WriteArrayEnd()
}
