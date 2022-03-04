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

func (wf *WideFrameSeries) AddMetric(metricName string, l data.Labels, t []time.Time, values interface{}) error {
	if !data.ValidFieldType(values) {
		return fmt.Errorf("type %T is not a valid data frame field type", values)
	}

	tFieldIndex := -1
	var timeIndicies []int

	if wf.Frame != nil {
		timeIndicies = wf.Frame.TypeIndices(data.FieldTypeTime)
	}

	if len(timeIndicies) != 0 {
		tFieldIndex = timeIndicies[0]
	}

	if t == nil && tFieldIndex == -1 {
		return fmt.Errorf("must provide time field when adding first metric")
	}

	if t != nil && tFieldIndex > -1 {
		return fmt.Errorf("time field must only be provided once")
	}

	valueField := data.NewField(metricName, l, values)
	var timeField *data.Field
	if t != nil {
		timeField = data.NewField("time", nil, t)
	} else {
		timeField = wf.Frame.Fields[tFieldIndex]
	}

	if valueField.Len() != timeField.Len() {
		return fmt.Errorf("value field length must match time field length, but gots length %v for time and %v for values",
			timeField.Len(), valueField.Len())
	}

	if t != nil {
		wf.Frame = data.NewFrame("", timeField, valueField)
		wf.Frame.SetMeta(&data.FrameMeta{Type: data.FrameTypeTimeSeriesWide})
	} else {
		wf.Fields = append(wf.Fields, valueField)
	}

	return nil
}

func (wf *WideFrameSeries) GetMetricRefs() ([]TimeSeriesMetricRef, []FrameFieldIndex) {
	refs := []TimeSeriesMetricRef{}
	var ignoredFields []FrameFieldIndex

	if wf == nil || wf.Frame == nil {
		return nil, nil
	}

	ignoreAllFields := func() {
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

func (wf *WideFrameSeries) Validate() (isEmpty bool, err []error) {
	panic("not implemented")
}
