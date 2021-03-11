package injest

import (
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/data/converters"
	"github.com/influxdata/telegraf"
)

type MetricFrameStream struct {
	id     uint64
	fields []fieldInfo
	Frame  *data.Frame
}

type fieldInfo struct {
	name    string
	convert data.Converter
}

// NewMetricFrameStream will return a new frame with length 1
func NewMetricFrameStream(m telegraf.Metric) (MetricFrameStream, error) {
	source := m.FieldList()
	s := MetricFrameStream{
		id:     m.HashID(),
		fields: make([]fieldInfo, 0),
	}

	// Prepare the tags
	tags := m.Tags()

	fields := make([]*data.Field, 0)
	field := data.NewField("time", nil, []time.Time{m.Time()})
	fields = append(fields, field)

	for _, f := range source {
		ft := data.FieldTypeFor(f.Value)
		if ft == data.FieldTypeUnknown {
			return s, fmt.Errorf("unknown type: %t", f.Value)
		}

		// Make all fields nullable
		ft = ft.NullableType()
		field = data.NewFieldFromFieldType(ft, 1)
		field.Name = f.Key
		field.Labels = tags

		info := fieldInfo{
			name:    f.Key,
			convert: func(v interface{}) (interface{}, error) { return v, nil },
		}

		switch ft {
		case data.FieldTypeNullableString:
			info.convert = converters.AnyToNullableString.Converter
		case data.FieldTypeNullableFloat64:
			info.convert = converters.JSONValueToNullableFloat64.Converter
		case data.FieldTypeNullableBool:
			info.convert = converters.BoolToNullableBool.Converter
		case data.FieldTypeNullableInt64:
			info.convert = converters.JSONValueToNullableInt64.Converter
		default:
			fmt.Printf("NO CONVERTER!!!! %s=%v (%T) %s\n", f.Key, f.Value, f.Value, ft.ItemTypeString())
		}

		v, err := info.convert(f.Value)
		if err != nil {
			field.Set(0, v)
		}

		//		fmt.Printf("%s=%v (%T) %s\n", f.Key, f.Value, v, field.Type().ItemTypeString())

		s.fields = append(s.fields, info)
		fields = append(fields, field)
	}

	s.Frame = data.NewFrame(m.Name(), fields...)
	return s, nil
}

// NullableType converts the FieldType to the corresponding nullable type.
func (s MetricFrameStream) Append(m telegraf.Metric) {
	count := len(s.fields)
	fields := s.Frame.Fields
	fields[0].Append(m.Time()) // first is always time

	for i := 0; i < count; i++ {
		info := s.fields[i]
		v, ok := m.GetField(info.name)
		if !ok {
			v = nil
		}
		val, err := info.convert(v)
		if err != nil {
			val = nil
		}
		fields[i+1].Append(val)
	}
}

// NullableType converts the FieldType to the corresponding nullable type.
func (s MetricFrameStream) Clear() {
	// ??????  better to just clear rows?
	s.Frame.ClearRows()
}
