package sdata

import (
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type NumericCollectionWriter interface {
	AddMetric(metricName string, l data.Labels, value interface{}) error
	SetMetricMD(metricName string, l data.Labels, fc data.FieldConfig)
}

type NumericCollection interface {
	NumericCollectionWriter
	NumericCollectionReader
}

type NumericCollectionReader interface {
	Validate() (isEmpty bool, errors []error)
	GetMetricRefs() []NumericMetricRef
}

type NumericMetricRef struct {
	ValueField *data.Field
}

type MultiFrameNumeric []*data.Frame

func (mfn *MultiFrameNumeric) AddMetric(metricName string, l data.Labels, value interface{}) error {
	fType := data.FieldTypeFor(value)
	if !fType.Numeric() {
		return fmt.Errorf("unsupported value type %T, must be numeric", value)
	}
	field := data.NewFieldFromFieldType(fType, 1)
	field.Name = metricName
	field.Labels = l
	field.Set(0, value)
	*mfn = append(*mfn, data.NewFrame("", field).SetMeta(&data.FrameMeta{
		Type: data.FrameType("numeric_multi"), // TODO: make type
	}))
	return nil
}

func (mfn *MultiFrameNumeric) GetMetricRefs() []NumericMetricRef {
	panic("not implemented")
}

func (mfn *MultiFrameNumeric) Validate() (isEmpty bool, errors []error) {
	panic("not implemented")
}

func (mfn *MultiFrameNumeric) SetMetricMD(metricName string, l data.Labels, fc data.FieldConfig) {
	panic("not implemented")
}
