package parsers

import (
	"fmt"
	"strconv"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	jsoniter "github.com/json-iterator/go"
)

// Take any prometheus json result and
func ReadPrometheusResult(iter *jsoniter.Iterator) *backend.DataResponse {
	var rsp *backend.DataResponse
	status := "unknown"

	for l1Field := iter.ReadObject(); l1Field != ""; l1Field = iter.ReadObject() {
		switch l1Field {
		case "status":
			status = iter.ReadString()

		case "data":
			rsp = readPrometheusData(iter)

		// case "error":
		// case "errorType":
		// case "warnings":
		default:
			v := iter.Read()
			fmt.Printf("[ROOT] TODO, support key: %s / %v\n", l1Field, v)
		}
	}

	if status != "success" {
		fmt.Printf("ERROR: %s\n", status)
	}

	return rsp
}

func readPrometheusData(iter *jsoniter.Iterator) *backend.DataResponse {
	t := iter.WhatIsNext()
	if t == jsoniter.ArrayValue {
		f := readArrayAsFrame(iter)
		return &backend.DataResponse{
			Frames: data.Frames{f},
		}
	}

	if t != jsoniter.ObjectValue {
		return &backend.DataResponse{
			Error: fmt.Errorf("expected object type"),
		}
	}

	resultType := ""
	var rsp *backend.DataResponse

	for l1Field := iter.ReadObject(); l1Field != ""; l1Field = iter.ReadObject() {
		switch l1Field {
		case "resultType":
			resultType = iter.ReadString()

		case "result":
			switch resultType {
			case "matrix":
				rsp = readMatrixOrVector(iter)
			case "vector":
				rsp = readMatrixOrVector(iter)
			case "stream":
				rsp = readStream(iter)
			default:
				iter.Skip()
				rsp = &backend.DataResponse{
					Error: fmt.Errorf("unknown result type: %s", resultType),
				}
			}

		case "stats":
			v := iter.Read()
			fmt.Printf("[data] TODO, support stats: %v\n", v)

		default:
			v := iter.Read()
			fmt.Printf("[data] TODO, support key: %s / %v\n", l1Field, v)
		}
	}

	fmt.Printf("result: %s\n", resultType)
	return rsp
}

// will always return strings for now
func readArrayAsFrame(iter *jsoniter.Iterator) *data.Frame {
	field := data.NewFieldFromFieldType(data.FieldTypeString, 0)
	field.Name = "Value"
	for iter.ReadArray() {
		v := ""
		t := iter.WhatIsNext()
		if t == jsoniter.StringValue {
			v = iter.ReadString()
		} else {
			ext := iter.ReadAny() // skip nills
			v = fmt.Sprintf("%v", ext)
		}
		field.Append(v)
	}
	return data.NewFrame("", field)
}

func readMatrixOrVector(iter *jsoniter.Iterator) *backend.DataResponse {
	rsp := &backend.DataResponse{}

	for iter.ReadArray() {
		timeField := data.NewFieldFromFieldType(data.FieldTypeTime, 0) // for now!
		valueField := data.NewFieldFromFieldType(data.FieldTypeFloat64, 0)
		valueField.Labels = data.Labels{}

		for l1Field := iter.ReadObject(); l1Field != ""; l1Field = iter.ReadObject() {
			switch l1Field {
			case "metric":
				iter.ReadVal(&valueField.Labels)

			case "value":
				t, v, err := readTimeValuePair(iter)
				if err == nil {
					timeField.Append(t)
					valueField.Append(v)
				}

			case "values":
				for iter.ReadArray() {
					t, v, err := readTimeValuePair(iter)
					if err == nil {
						timeField.Append(t)
						valueField.Append(v)
					}
				}
			}
		}

		valueField.Name = valueField.Labels["__name__"]
		delete(valueField.Labels, "__name__")

		frame := data.NewFrame("", timeField, valueField)
		frame.Meta = &data.FrameMeta{
			Type: data.FrameTypeTimeSeriesMany,
		}
		rsp.Frames = append(rsp.Frames, frame)
	}

	return rsp
}

func readTimeValuePair(iter *jsoniter.Iterator) (time.Time, float64, error) {
	iter.ReadArray()
	t := iter.ReadFloat64()
	iter.ReadArray()
	v := iter.ReadString()
	iter.ReadArray()

	tt := time.Unix(int64(t), 0) // HELP! only second precision!!!
	fv, err := strconv.ParseFloat(v, 64)
	return tt, fv, err
}

func readStream(iter *jsoniter.Iterator) *backend.DataResponse {
	panic("TODO stream")
}
