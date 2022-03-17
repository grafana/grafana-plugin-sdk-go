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

func (wf WideFrameSeries) GetMetricRefs() ([]TimeSeriesMetricRef, []FrameFieldIndex, error) {
	return validateAndGetRefsWide(wf, false, true)
}

func (wf *WideFrameSeries) SetMetricMD(metricName string, l data.Labels, fc data.FieldConfig) {
	panic("not implemented")
}

func (wf WideFrameSeries) Validate(validateData bool) (ignoredFields []FrameFieldIndex, err error) {
	_, ignoredFields, err = validateAndGetRefsWide(wf, validateData, false)
	return ignoredFields, err
}

func validateAndGetRefsWide(wf WideFrameSeries, validateData, getRefs bool) ([]TimeSeriesMetricRef, []FrameFieldIndex, error) {
	var refs []TimeSeriesMetricRef
	var ignoredFields []FrameFieldIndex
	metricIndex := make(map[[2]string]struct{})

	if wf.Frame == nil {
		return nil, nil, fmt.Errorf("frame is nil which is invalid")
	}

	if len(wf.Fields) == 0 { // TODO: Error differently if nil and not zero length?
		return refs, nil, nil // empty response
	}

	if _, err := wf.RowLen(); err != nil {
		return nil, nil, fmt.Errorf("frame has mismatched field lengths: %w", err)
	}

	for fieldIdx, field := range wf.Fields { // TODO: frame.TypeIndices should do this
		if field == nil {
			return nil, nil, fmt.Errorf("frame has a nil field at %v", fieldIdx)
		}
	}

	timeFields := wf.TypeIndices(data.FieldTypeTime)
	valueFieldIndicies := wf.TypeIndices(ValidValueFields()...)

	if len(timeFields) == 0 {
		return nil, nil, fmt.Errorf("frame is missing a []time.Time field")
	}

	if len(valueFieldIndicies) == 0 {
		return nil, nil, fmt.Errorf("frame is missing a numeric value field")
	}

	timeField := wf.Fields[timeFields[0]]
	// Validate time Field is sorted in ascending (oldest to newest) order
	if validateData {
		sorted, err := timeIsSorted(timeField)
		if err != nil {
			return nil, nil, fmt.Errorf("frame has an malformed time field")
		}
		if !sorted {
			return nil, nil, fmt.Errorf("frame has an unsorted time field")
		}
	}

	if len(timeFields) > 1 {
		for _, fieldIdx := range timeFields[1:] {
			ignoredFields = append(ignoredFields, FrameFieldIndex{0, fieldIdx, "additional time field"})
		}
	}

	// TODO this is fragile if new types are added
	otherFields := wf.TypeIndices(data.FieldTypeNullableTime, data.FieldTypeString, data.FieldTypeNullableString)
	for _, fieldIdx := range otherFields {
		ignoredFields = append(ignoredFields, FrameFieldIndex{0, fieldIdx, fmt.Sprintf("unsupported field type %v", wf.Fields[fieldIdx].Type())})
	}

	for _, vFieldIdx := range valueFieldIndicies {
		vField := wf.Fields[vFieldIdx]
		if validateData {
			metricKey := [2]string{vField.Name, vField.Labels.String()}
			if _, ok := metricIndex[metricKey]; ok && validateData {
				return nil, nil, fmt.Errorf("duplicate metrics found for metric name %q and labels %q", vField.Name, vField.Labels)
			}
			metricIndex[metricKey] = struct{}{}
		}
		if getRefs {
			refs = append(refs, TimeSeriesMetricRef{
				TimeField:  timeField,
				ValueField: vField,
			})
		}
	}

	sortTimeSeriesMetricRef(refs)
	return refs, ignoredFields, nil
}
