package sdata

import (
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type LongSeries []*data.Frame

func NewLongSeries() *LongSeries { // possible TODO: argument BoolAsMetric
	return &LongSeries{emptyFrameWithTypeMD(data.FrameTypeTimeSeriesLong)}
}

func (ls *LongSeries) Validate(validateData bool) (ignoredFields []FrameFieldIndex, err error) {
	_, ignored, err := validateAndGetRefsLong(ls, validateData, false)
	if err != nil {
		return nil, err
	}
	return ignored, nil
}

func (ls *LongSeries) GetMetricRefs() ([]TimeSeriesMetricRef, []FrameFieldIndex, error) {
	return validateAndGetRefsLong(ls, false, true)
}

func validateAndGetRefsLong(ls *LongSeries, validateData, getRefs bool) ([]TimeSeriesMetricRef, []FrameFieldIndex, error) {
	switch {
	case ls == nil:
		return nil, nil, fmt.Errorf("frames may not be nil")
	case len(*ls) == 0:
		return nil, nil, fmt.Errorf("missing frame, must be at least one frame")
	}

	frame := (*ls)[0]

	if frame == nil {
		return nil, nil, fmt.Errorf("frame 0 must not be nil")
	}

	if !frameHasType(frame, data.FrameTypeTimeSeriesLong) {
		return nil, nil, fmt.Errorf("frame 0 is missing long type indicator")
	}

	var ignoredFields []FrameFieldIndex
	if len(frame.Fields) == 0 { // empty response
		if err := ignoreAdditionalFrames("additional frame on empty response", *ls, &ignoredFields); err != nil {
			return nil, nil, err
		}
		return []TimeSeriesMetricRef{}, ignoredFields, nil
	}

	if err := malformedFrameCheck(0, frame); err != nil {
		return nil, nil, err
	}

	// metricName/labels -> SeriesRef
	mm := make(map[string]map[string]TimeSeriesMetricRef)

	timeField, ignoredTimedFields, err := seriesCheckSelectTime(0, frame)
	if err != nil {
		return nil, nil, err
	}
	if ignoredTimedFields != nil {
		ignoredFields = append(ignoredFields, ignoredTimedFields...)
	}

	valueFieldIndices := frame.TypeIndices(ValidValueFields()...) // TODO switch on bool type option
	if len(valueFieldIndices) == 0 {
		return nil, nil, fmt.Errorf("frame is missing a numeric value field")
	}

	factorFieldIndices := frame.TypeIndices(data.FieldTypeString, data.FieldTypeNullableString)

	var refs []TimeSeriesMetricRef
	appendToMetric := func(metricName string, l data.Labels, t time.Time, value interface{}) {
		if mm[metricName] == nil {
			mm[metricName] = make(map[string]TimeSeriesMetricRef)
		}

		lbStr := l.String()
		if ref, ok := mm[metricName][lbStr]; !ok {
			ref.TimeField = data.NewField(timeField.Name, nil, []time.Time{t})

			vt := data.FieldTypeFor(value)
			ref.ValueField = data.NewFieldFromFieldType(vt, 1)
			ref.ValueField.Set(0, value)
			ref.ValueField.Name = metricName
			ref.ValueField.Labels = l

			mm[metricName][lbStr] = ref
			refs = append(refs, ref)
		} else {
			ref.TimeField.Append(t)
			ref.ValueField.Append(value)
		}
	}

	if getRefs {
		for rowIdx := 0; rowIdx < frame.Rows(); rowIdx++ {
			l := data.Labels{}
			for _, strFieldIdx := range factorFieldIndices {
				cv, _ := frame.ConcreteAt(strFieldIdx, rowIdx)
				l[frame.Fields[strFieldIdx].Name] = cv.(string)
			}
			for _, vFieldIdx := range valueFieldIndices {
				valueField := frame.Fields[vFieldIdx]
				appendToMetric(valueField.Name, l, timeField.At(rowIdx).(time.Time), valueField.At(rowIdx))
			}
		}
		sortTimeSeriesMetricRef(refs)
	}

	// TODO this is fragile if new types are added
	otherFields := frame.TypeIndices(data.FieldTypeNullableTime)
	for _, fieldIdx := range otherFields {
		ignoredFields = append(ignoredFields, FrameFieldIndex{0, fieldIdx, fmt.Sprintf("unsupported field type %v", frame.Fields[fieldIdx].Type())})
	}

	if len(*ls) > 1 {
		for frameIdx, f := range *ls {
			if f == nil {
				return nil, nil, fmt.Errorf("nil frame at %v which is invalid", frameIdx)
			}
			if len(f.Fields) == 0 {
				ignoredFields = append(ignoredFields, FrameFieldIndex{frameIdx, -1, "extra frame"})
			}
			for fieldIdx := range *ls {
				ignoredFields = append(ignoredFields, FrameFieldIndex{frameIdx, fieldIdx, "extra frame"})
			}
		}
	}

	if err := ignoreAdditionalFrames("additional frame", *ls, &ignoredFields); err != nil {
		return nil, nil, err
	}

	return refs, ignoredFields, nil
}

func (ls *LongSeries) Frames() []*data.Frame {
	if ls == nil {
		return nil
	}
	return []*data.Frame(*ls)
}
