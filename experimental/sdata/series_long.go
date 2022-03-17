package sdata

import (
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
	panic("not implemented")
}

func (ls LongSeries) GetMetricRefs() ([]TimeSeriesMetricRef, []FrameFieldIndex, error) {
	panic("needs updating to collapse in with validate like multi")
	/*
		if ls.Frame == nil || ls.Fields == nil {
			return nil, nil // TODO I think I added some meaning for nil vs empty in another... func
		}

		var ignoredFields []FrameFieldIndex
		ignoreAllFields := func(reason string) {
			for fieldIdx := range ls.Fields {
				ignoredFields = append(ignoredFields, FrameFieldIndex{0, fieldIdx, reason})
			}
		}

		if ls.Frame.Meta == nil || ls.Frame.Meta.Type != data.FrameTypeTimeSeriesLong {
			ignoreAllFields("TODO")
			return nil, ignoredFields
		}

		// metricName/labels -> SeriesRef
		mm := make(map[string]map[string]TimeSeriesMetricRef)

		timeFields := ls.TypeIndices(data.FieldTypeTime)
		valueFieldIndicies := ls.TypeIndices(ValidValueFields()...) // TODO switch on bool type option

		if len(timeFields) == 0 || len(valueFieldIndicies) == 0 {
			ignoreAllFields("TODO")
			return []TimeSeriesMetricRef{}, ignoredFields
		}

		timeField := ls.Fields[timeFields[0]]

		if len(timeFields) > 1 {
			for _, fieldIdx := range timeFields[1:] {
				ignoredFields = append(ignoredFields, FrameFieldIndex{0, fieldIdx, "TODO"})
			}
		}

		factorFieldIndicies := ls.TypeIndices(data.FieldTypeString, data.FieldTypeNullableString)

		refs := []TimeSeriesMetricRef{}
		appendToMetric := func(metricName string, l data.Labels, t time.Time, value interface{}) {
			if mm[metricName] == nil {
				mm[metricName] = make(map[string]TimeSeriesMetricRef)
			}

			lbStr := l.String()
			if ref, ok := mm[metricName][lbStr]; !ok {
				// TODO could carry time field name
				ref.TimeField = data.NewField("time", nil, []time.Time{t})

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

		for rowIdx := 0; rowIdx < ls.Rows(); rowIdx++ {
			l := data.Labels{}
			for _, strFieldIdx := range factorFieldIndicies {
				cv, _ := ls.ConcreteAt(strFieldIdx, rowIdx)
				l[ls.Fields[strFieldIdx].Name] = cv.(string)
			}
			for _, vFieldIdx := range valueFieldIndicies {
				valueField := ls.Fields[vFieldIdx]
				appendToMetric(valueField.Name, l, timeField.At(rowIdx).(time.Time), valueField.At(rowIdx))
			}
		}
		sortTimeSeriesMetricRef(refs)
		return refs, ignoredFields
	*/
}
