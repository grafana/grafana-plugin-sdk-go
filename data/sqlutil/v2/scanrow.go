package sqlutil

import (
	"reflect"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type ScanRow struct {
	Data    []interface{}
	Columns []string
	Types   []reflect.Type
}

func NewScanRow(length int) *ScanRow {
	return &ScanRow{
		Columns: make([]string, length),
		Types:   make([]reflect.Type, length),
	}
}

func (s *ScanRow) Append(value interface{}, name string, colType reflect.Type) {
	s.Data = append(s.Data, value)
	s.Columns = append(s.Columns, name)
	s.Types = append(s.Types, colType)
}

func (s *ScanRow) Set(i int, value interface{}, name string, colType reflect.Type) {
	s.Data[i] = value
	s.Columns[i] = name
	s.Types[i] = colType
}

// NewScannableRow creates a list where each element is an instance of the provided
func (s *ScanRow) NewScannableRow() []interface{} {
	values := make([]interface{}, len(s.Types))

	for i, v := range s.Types {
		if v.Kind() == reflect.Ptr {
			values[i] = reflect.New(v)
		} else {
			values[i] = reflect.New(v).Interface()
		}
	}

	return values
}

func NewFrame(converters ...Converter) *data.Frame {
	fields := make(data.Fields, len(converters))

	for i, v := range converters {
		fields[i] = data.NewFieldFromFieldType(v.FrameConverter.FieldType, 0)
	}

	return data.NewFrame("results", fields...)
}

func Append(frame *data.Frame, row []interface{}, converters ...Converter) error {
	d := make([]interface{}, len(row))
	for i, v := range row {
		value, err := converters[i].FrameConverter.ConverterFunc(v)
		if err != nil {
			return err
		}
		d[i] = value
	}

	frame.AppendRow(d...)
	return nil
}
