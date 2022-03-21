package sdata

import (
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type LongSeries struct {
	*data.Frame
	BoolAsMetric bool
	// TODO, BoolAsMetricneeds be a property on Frame somewhere
	// or: we get rid of property, and the ds must turn the bool into a number, otherwise it is a dimension
}

func NewLongSeries() LongSeries { // possible TODO: argument BoolAsMetric
	return LongSeries{Frame: emptyFrameWithTypeMD(data.FrameTypeTimeSeriesLong)}
}

func (ls LongSeries) Validate(validateData bool) (ignoredFields []FrameFieldIndex, err error) {
	_, ignored, err := validateAndGetRefsLong(ls, validateData, false)
	if err != nil {
		return nil, err
	}
	return ignored, nil
}

func (ls LongSeries) GetMetricRefs() ([]TimeSeriesMetricRef, []FrameFieldIndex, error) {
	return validateAndGetRefsLong(ls, false, true)
}

func validateAndGetRefsLong(ls LongSeries, validateData, getRefs bool) ([]TimeSeriesMetricRef, []FrameFieldIndex, error) {
	if ls.Frame == nil {
		return nil, nil, fmt.Errorf("frame must not be nil")
	}

	if !frameHasType(ls.Frame, data.FrameTypeTimeSeriesLong) {
		return nil, nil, fmt.Errorf("frame is missing long type indicator")
	}

	if len(ls.Fields) == 0 { // empty response
		return []TimeSeriesMetricRef{}, nil, nil
	}

	if err := malformedFrameCheck(0, ls.Frame); err != nil {
		return nil, nil, err
	}

	var ignoredFields []FrameFieldIndex

	// metricName/labels -> SeriesRef
	mm := make(map[string]map[string]TimeSeriesMetricRef)

	timeField, ignoredTimedFields, err := seriesCheckSelectTime(0, ls.Frame)
	if err != nil {
		return nil, nil, err
	}
	if ignoredTimedFields != nil {
		ignoredFields = append(ignoredFields, ignoredTimedFields...)
	}

	valueFieldIndices := ls.TypeIndices(ValidValueFields()...) // TODO switch on bool type option
	if len(valueFieldIndices) == 0 {
		return nil, nil, fmt.Errorf("frame is missing a numeric value field")
	}

	factorFieldIndices := ls.TypeIndices(data.FieldTypeString, data.FieldTypeNullableString)

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
		for rowIdx := 0; rowIdx < ls.Rows(); rowIdx++ {
			l := data.Labels{}
			for _, strFieldIdx := range factorFieldIndices {
				cv, _ := ls.ConcreteAt(strFieldIdx, rowIdx)
				l[ls.Fields[strFieldIdx].Name] = cv.(string)
			}
			for _, vFieldIdx := range valueFieldIndices {
				valueField := ls.Fields[vFieldIdx]
				appendToMetric(valueField.Name, l, timeField.At(rowIdx).(time.Time), valueField.At(rowIdx))
			}
		}
		sortTimeSeriesMetricRef(refs)
	}

	// TODO this is fragile if new types are added
	otherFields := ls.Frame.TypeIndices(data.FieldTypeNullableTime)
	for _, fieldIdx := range otherFields {
		ignoredFields = append(ignoredFields, FrameFieldIndex{0, fieldIdx, fmt.Sprintf("unsupported field type %v", ls.Frame.Fields[fieldIdx].Type())})
	}

	return refs, ignoredFields, nil
}
