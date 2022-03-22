package sdata

import (
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type WideFrameSeries []*data.Frame

func NewWideFrameSeries() *WideFrameSeries {
	f := data.NewFrame("")
	f.SetMeta(&data.FrameMeta{Type: data.FrameTypeTimeSeriesWide})
	return &WideFrameSeries{f}
}

func (wf *WideFrameSeries) SetTime(timeName string, t []time.Time) error {
	switch {
	case wf == nil:
		return fmt.Errorf("wf is nil, NewWideFrameSeries must be called first")
	case len(*wf) == 0:
		return fmt.Errorf("missing frame, NewWideFrameSeries must be called first")
	case len(*wf) > 1:
		return fmt.Errorf("may not set time after adding extra frames")
	}

	frame := (*wf)[0]

	switch {
	case t == nil:
		return fmt.Errorf("t may not be nil")
	case frame.Fields != nil:
		return fmt.Errorf("expected fields property to be nil (metrics added before calling SetTime?)")
	case frame == nil:
		return fmt.Errorf("missing is nil, NewWideFrameSeries must be called first")
	}

	frame.Fields = append(frame.Fields, data.NewField(timeName, nil, t))
	return nil
}

func (wf *WideFrameSeries) AddMetric(metricName string, l data.Labels, values interface{}) error {
	if !data.ValidFieldType(values) {
		return fmt.Errorf("type %T is not a valid data frame field type", values)
	}

	switch {
	case wf == nil:
		return fmt.Errorf("wf is nil, NewWideFrameSeries must be called first")
	case len(*wf) == 0:
		return fmt.Errorf("missing frame, NewWideFrameSeries must be called first")
	case len(*wf) > 1:
		return fmt.Errorf("may not add metrics after adding extra frames")
	}

	frame := (*wf)[0]

	if frame == nil {
		return fmt.Errorf("missing is nil, NewWideFrameSeries must be called first")
	}

	// Note: Readers are not required to make the Time field first, but using New/SetTime/AddMetric does.
	if len(frame.Fields) == 0 || frame.Fields[0].Type() != data.FieldTypeTime {
		return fmt.Errorf("frame is missing time field or time field is not first, SetTime must be called first")
	}

	valueField := data.NewField(metricName, l, values)

	if valueField.Len() != frame.Fields[0].Len() {
		return fmt.Errorf("value field length must match time field length, but got length %v for time and %v for values",
			frame.Fields[0].Len(), valueField.Len())
	}

	frame.Fields = append(frame.Fields, valueField)

	return nil
}

func (wf *WideFrameSeries) GetMetricRefs() ([]TimeSeriesMetricRef, []FrameFieldIndex, error) {
	return validateAndGetRefsWide(wf, false, true)
}

func (wf *WideFrameSeries) SetMetricMD(metricName string, l data.Labels, fc data.FieldConfig) {
	panic("not implemented")
}

func (wf *WideFrameSeries) Validate(validateData bool) (ignoredFields []FrameFieldIndex, err error) {
	_, ignoredFields, err = validateAndGetRefsWide(wf, validateData, false)
	return ignoredFields, err
}

func validateAndGetRefsWide(wf *WideFrameSeries, validateData, getRefs bool) ([]TimeSeriesMetricRef, []FrameFieldIndex, error) {
	var refs []TimeSeriesMetricRef
	var ignoredFields []FrameFieldIndex
	metricIndex := make(map[[2]string]struct{})

	switch {
	case wf == nil:
		return nil, nil, fmt.Errorf("frames may not be nil")
	case len(*wf) == 0:
		return nil, nil, fmt.Errorf("missing frame, must be at least one frame")
	}

	frame := (*wf)[0]

	if frame == nil {
		return nil, nil, fmt.Errorf("frame is nil which is invalid")
	}

	if len(frame.Fields) == 0 { // TODO: Error differently if nil and not zero length?
		if err := ignoreAdditionalFrames("additional frame on empty response", *wf, &ignoredFields); err != nil {
			return nil, nil, err
		}
		return []TimeSeriesMetricRef{}, nil, nil // empty response
	}

	if err := malformedFrameCheck(0, frame); err != nil {
		return nil, nil, err
	}

	timeField, ignoredTimedFields, err := seriesCheckSelectTime(0, frame)
	if err != nil {
		return nil, nil, err
	}
	if ignoredTimedFields != nil {
		ignoredFields = append(ignoredFields, ignoredTimedFields...)
	}

	valueFieldIndices := frame.TypeIndices(ValidValueFields()...)
	if len(valueFieldIndices) == 0 {
		return nil, nil, fmt.Errorf("frame is missing a numeric value field")
	}

	// TODO this is fragile if new types are added
	otherFields := frame.TypeIndices(data.FieldTypeNullableTime, data.FieldTypeString, data.FieldTypeNullableString)
	for _, fieldIdx := range otherFields {
		ignoredFields = append(ignoredFields, FrameFieldIndex{0, fieldIdx, fmt.Sprintf("unsupported field type %v", frame.Fields[fieldIdx].Type())})
	}

	for _, vFieldIdx := range valueFieldIndices {
		vField := frame.Fields[vFieldIdx]
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

	ignoreAdditionalFrames("additional frame", *wf, &ignoredFields)

	sortTimeSeriesMetricRef(refs)
	return refs, ignoredFields, nil
}

func (wf *WideFrameSeries) Frames() []*data.Frame {
	if wf == nil {
		return nil
	}
	return []*data.Frame(*wf)
}
