package timeseries

import (
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/sdata"
)

// LongFrame is a time series format where all series live in one frame.
// This time series format should be used with Table-like sources (e.g. SQL) that
// do not have a native concept of Labels.
type LongFrame []*data.Frame

func NewLongFrame() *LongFrame { // possible TODO: argument BoolAsMetric
	return &LongFrame{emptyFrameWithTypeMD(data.FrameTypeTimeSeriesLong)}
}

func (ls *LongFrame) GetMetricRefs(validateData bool) ([]MetricRef, []sdata.FrameFieldIndex, error) {
	return validateAndGetRefsLong(ls, validateData, true)
}

func validateAndGetRefsLong(ls *LongFrame, validateData, getRefs bool) ([]MetricRef, []sdata.FrameFieldIndex, error) {
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

	var ignoredFields []sdata.FrameFieldIndex
	if len(frame.Fields) == 0 { // empty response
		if err := ignoreAdditionalFrames("additional frame on empty response", *ls, &ignoredFields); err != nil {
			return nil, nil, err
		}
		return []MetricRef{}, ignoredFields, nil
	}

	if err := malformedFrameCheck(0, frame); err != nil {
		return nil, nil, err
	}

	// metricName/labels -> SeriesRef
	mm := make(map[string]map[string]MetricRef)

	timeField, ignoredTimedFields, err := seriesCheckSelectTime(0, frame)
	if err != nil {
		return nil, nil, err
	}
	if ignoredTimedFields != nil {
		ignoredFields = append(ignoredFields, ignoredTimedFields...)
	}

	valueFieldIndices := frame.TypeIndices(sdata.ValidValueFields()...) // TODO switch on bool type option
	if len(valueFieldIndices) == 0 {
		return nil, nil, fmt.Errorf("frame is missing a numeric value field")
	}

	factorFieldIndices := frame.TypeIndices(data.FieldTypeString, data.FieldTypeNullableString)

	var refs []MetricRef
	appendToMetric := func(metricName string, l data.Labels, t time.Time, value interface{}) error {
		if mm[metricName] == nil {
			mm[metricName] = make(map[string]MetricRef)
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
			if validateData && ref.TimeField.Len() > 1 {
				prevTime := ref.TimeField.At(ref.TimeField.Len() - 1).(time.Time)
				if prevTime.After(t) {
					return fmt.Errorf("unsorted time field")
				}
				if prevTime.Equal(t) {
					return fmt.Errorf("duplicate data points in metric %v %v", metricName, lbStr)
				}
			}
			ref.TimeField.Append(t)
			ref.ValueField.Append(value)
		}
		return nil
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
				if err := appendToMetric(valueField.Name, l, timeField.At(rowIdx).(time.Time), valueField.At(rowIdx)); err != nil {
					return nil, nil, err
				}
			}
		}
		sortTimeSeriesMetricRef(refs)
	}

	// TODO this is fragile if new types are added
	otherFields := frame.TypeIndices(data.FieldTypeNullableTime)
	for _, fieldIdx := range otherFields {
		ignoredFields = append(ignoredFields, sdata.FrameFieldIndex{
			FrameIdx: 0, FieldIdx: fieldIdx,
			Reason: fmt.Sprintf("unsupported field type %v", frame.Fields[fieldIdx].Type())},
		)
	}

	if err := ignoreAdditionalFrames("additional frame", *ls, &ignoredFields); err != nil {
		return nil, nil, err
	}

	return refs, ignoredFields, nil
}

func (ls *LongFrame) Frames() []*data.Frame {
	if ls == nil {
		return nil
	}
	return []*data.Frame(*ls)
}
