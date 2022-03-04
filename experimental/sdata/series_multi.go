package sdata

import (
	"fmt"
	"sort"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// or perhaps a container struct with non-exported fields (for indicies and such) and the Frames exported.
type MultiFrameSeries []*data.Frame

// values must be a numeric slice such as []int64, []float64, []*float64, etc or []bool / []*bool.
func (mfs *MultiFrameSeries) AddMetric(metricName string, l data.Labels, t []time.Time, values interface{}) error {
	var err error

	if !data.ValidFieldType(values) {
		return fmt.Errorf("type %T is not a valid data frame field type", values)
	}

	valueField := data.NewField(metricName, l, values) // note
	timeField := data.NewField("time", nil, t)

	if valueField.Len() != timeField.Len() {
		// return error since creating the frame will eventually fail to marshal due to the
		// arrow constraint that fields must be of the same length.
		// Alternatively we could pad, but this seems like it would be a programing error more than
		// a data error to me.
		return fmt.Errorf("invalid series, time and value must be of the same length")
	}

	valueFieldType := valueField.Type()
	if !valueFieldType.Numeric() && valueFieldType != data.FieldTypeBool && valueFieldType != data.FieldTypeNullableBool {
		err = fmt.Errorf("value type %s is not valid time series value type", valueFieldType)
	}

	frame := data.NewFrame("", timeField, valueField)
	frame.SetMeta(&data.FrameMeta{Type: data.FrameTypeTimeSeriesMany}) // I think "Multi" is better than "Many"
	*mfs = append(*mfs, frame)
	return err
}

func (mfs *MultiFrameSeries) SetMetricMD(metricName string, l data.Labels, fc data.FieldConfig) {
	panic("not implemented")
}

func (mfs *MultiFrameSeries) GetMetricRefs() ([]TimeSeriesMetricRef, []FrameFieldIndex) {
	// if no Frames we return nil (non-empty input but no series will be zero length)
	if mfs == nil || len(*mfs) == 0 {
		return nil, nil
	}

	var ignoredFields []FrameFieldIndex
	refs := []TimeSeriesMetricRef{}

	for frameIdx, frame := range *mfs {
		if frame == nil { // nil frames not valid
			ignoredFields = append(ignoredFields, FrameFieldIndex{-1, -1})
			continue
		}

		ignoreAllFields := func() {
			for fieldIdx := range frame.Fields {
				ignoredFields = append(ignoredFields, FrameFieldIndex{frameIdx, fieldIdx})
			}
		}

		if !frameHasMetaType(frame, data.FrameTypeTimeSeriesMany) { // must have type indicator
			ignoreAllFields()
			continue
		}

		m := TimeSeriesMetricRef{}

		if len(frame.Fields) == 0 {
			if frameIdx == 0 && len(*mfs) == 1 {
				// If Type indicator is there (checked earlier) then it is an empty typed response (no metrics in response)
				// There should only be one
				return refs, nil
			} else {
				ignoreAllFields()
				continue
			}
		}

		valueFields := frame.TypeIndices(ValidValueFields()...)
		timeFields := frame.TypeIndices(data.FieldTypeTime)

		if len(valueFields) == 0 || len(timeFields) == 0 {
			ignoreAllFields()
			continue
		}

		// Time Field
		if len(timeFields) == 1 {
			m.TimeField = frame.Fields[timeFields[0]]
		} else {
			m.TimeField = frame.Fields[timeFields[0]]
			for _, fieldIdx := range timeFields[1:] {
				ignoredFields = append(ignoredFields, FrameFieldIndex{frameIdx, fieldIdx})
			}
		}
		// Value Field
		if len(valueFields) == 1 {
			m.ValueField = frame.Fields[valueFields[0]]
		} else {
			m.ValueField = frame.Fields[valueFields[0]]
			for _, fieldIdx := range timeFields[1:] {
				ignoredFields = append(ignoredFields, FrameFieldIndex{frameIdx, fieldIdx})
			}
		}

		// TODO this is fragile if new types are added
		otherFields := frame.TypeIndices(data.FieldTypeNullableTime, data.FieldTypeString, data.FieldTypeNullableString)
		for _, fieldIdx := range otherFields {
			ignoredFields = append(ignoredFields, FrameFieldIndex{frameIdx, fieldIdx})
		}

		refs = append(refs, m)
	}

	sortTimeSeriesMetricRef(refs)
	return refs, ignoredFields
}

// Validates data conforms to schema, don't think it will be called normally in the course of running a plugin, but needs to exist.
// Currently this is strict in the sense that consumers must support all valid instances. Consumers may support invalid instances
// depending on the circumstances.
func (mfs *MultiFrameSeries) Validate() (isEmpty bool, ignoredFieldIndices []FrameFieldIndex, errors []error) {
	if mfs == nil || len(*mfs) == 0 {
		// Unless we create a container (and expose it in our responses) that can hold the type(s) for the frames it contains,
		// anything empty probably needs be considered "valid" for the type. Else we have a requirement to create at least one frame (eww).
		return true, nil, nil
	}

	metricIndex := make(map[[2]string]struct{})

	for frameIdx, frame := range *mfs {
		if frame.Meta == nil || frame.Meta.Type != data.FrameTypeTimeSeriesMany {
			errors = append(errors, fmt.Errorf("frame %v is missing type indicator in frame metadata", frameIdx))
		}

		if len(frame.Fields) == 0 {
			// an individual frame with no fields is an empty series is valid.
			continue
		}

		if _, err := frame.RowLen(); err != nil {
			errors = append(errors, fmt.Errorf("frame %v has mismatched field lengths: %w", frameIdx, err))
		}

		timeFields := frame.TypeIndices(data.FieldTypeTime)

		// Must have []time.Time field (no nullable time)
		if len(timeFields) == 0 {
			errors = append(errors, fmt.Errorf("frame %v must have at least one time field but has %v", frameIdx, len(timeFields)))
		} else {
			if len(timeFields) > 1 {
				for _, fieldIdx := range timeFields[1:] {
					ignoredFieldIndices = append(ignoredFieldIndices, FrameFieldIndex{frameIdx, fieldIdx})
				}
			}

			// Validate time Field is sorted in ascending (oldest to newest) order
			timeField := frame.Fields[timeFields[0]]
			if timeField.Len() > 1 {
				for tIdx := 1; tIdx < timeField.Len(); tIdx++ {
					prevTime := timeField.At(tIdx - 1).(time.Time)
					curTime := timeField.At(tIdx).(time.Time)
					if curTime.Before(prevTime) {
						errors = append(errors, fmt.Errorf("frame %v has an unsorted time field", frameIdx))
						break
					}
				}
			}

			valueFields := frame.TypeIndices(ValidValueFields()...)

			if len(valueFields) == 0 {
				errors = append(errors, fmt.Errorf("frame %v must have at least one value field but has %v", frameIdx, len(valueFields)))
			} else {
				if len(valueFields) > 1 {
					for _, fieldIdx := range valueFields[1:] {
						ignoredFieldIndices = append(ignoredFieldIndices, FrameFieldIndex{frameIdx, fieldIdx})
					}
				}
				vField := frame.Fields[valueFields[0]]
				metricKey := [2]string{vField.Name, vField.Labels.String()}
				if _, ok := metricIndex[metricKey]; ok {
					errors = append(errors, fmt.Errorf("duplicate metrics found for metric name %q and labels %q", vField.Name, vField.Labels))
				} else {
					metricIndex[metricKey] = struct{}{}
				}
			}
			// TODO this is fragile if new types are added
			otherFields := frame.TypeIndices(data.FieldTypeNullableTime, data.FieldTypeString, data.FieldTypeNullableString)
			for _, fieldIdx := range otherFields {
				ignoredFieldIndices = append(ignoredFieldIndices, FrameFieldIndex{frameIdx, fieldIdx})
			}
		}
	}

	if errors != nil {
		ignoredFieldIndices = nil
	}

	sort.Sort(FrameFieldIndices(ignoredFieldIndices))

	return false, ignoredFieldIndices, errors
}
