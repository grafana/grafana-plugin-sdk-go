package sdata

import (
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// need to think about pointers here and elsewhere
type WideFrameSeries struct {
	*data.Frame
}

func NewWideFrameSeries(timeName string, t []time.Time) WideFrameSeries {
	tF := data.NewField(timeName, nil, t)
	f := data.NewFrame("", tF)
	f.SetMeta(&data.FrameMeta{Type: data.FrameTypeTimeSeriesWide})
	return WideFrameSeries{f}
}

func (wf WideFrameSeries) AddMetric(metricName string, l data.Labels, values interface{}) error {
	if !data.ValidFieldType(values) {
		return fmt.Errorf("type %T is not a valid data frame field type", values)
	}

	if wf.Frame == nil {
		return fmt.Errorf("missing frame, NewWideFrameSeries must be called first")
	}

	if len(wf.Frame.Fields) == 0 || wf.Frame.Fields[0].Type() != data.FieldTypeTime {
		return fmt.Errorf("frame is missing time field or time field is not first, NewWideFrameSeries must be called first")
	}

	valueField := data.NewField(metricName, l, values)

	if valueField.Len() != wf.Frame.Fields[0].Len() {
		return fmt.Errorf("value field length must match time field length, but got length %v for time and %v for values",
			wf.Frame.Fields[0].Len(), valueField.Len())
	}

	wf.Fields = append(wf.Fields, valueField)

	return nil
}

func (wf WideFrameSeries) GetMetricRefs() ([]TimeSeriesMetricRef, []FrameFieldIndex) {
	refs := []TimeSeriesMetricRef{}
	var ignoredFields []FrameFieldIndex

	if wf.Frame == nil {
		return nil, nil
	}

	ignoreAllFields := func() {
		if len(wf.Fields) == 0 {
			ignoredFields = append(ignoredFields, FrameFieldIndex{0, -1})
		}
		for fieldIdx := range wf.Fields {
			ignoredFields = append(ignoredFields, FrameFieldIndex{0, fieldIdx})
		}
	}

	timeFields := wf.TypeIndices(data.FieldTypeTime)
	valueFieldIndicies := wf.TypeIndices(ValidValueFields()...)

	if len(timeFields) == 0 || len(valueFieldIndicies) == 0 {
		ignoreAllFields()
		return refs, ignoredFields
	}

	timeField := wf.Fields[timeFields[0]]

	if len(timeFields) > 1 {
		for _, fieldIdx := range timeFields[1:] {
			ignoredFields = append(ignoredFields, FrameFieldIndex{0, fieldIdx})
		}
	}

	for _, fieldIdx := range valueFieldIndicies {
		refs = append(refs, TimeSeriesMetricRef{
			TimeField:  timeField,
			ValueField: wf.Fields[fieldIdx],
		})
	}
	sortTimeSeriesMetricRef(refs)
	return refs, ignoredFields
}

func (wf *WideFrameSeries) SetMetricMD(metricName string, l data.Labels, fc data.FieldConfig) {
	panic("not implemented")
}

func (wf *WideFrameSeries) Validate() (ignoredFields []FrameFieldIndex, err error) {
	panic("not implemented")
}
