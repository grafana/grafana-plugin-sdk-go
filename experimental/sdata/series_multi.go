package sdata

import (
	"fmt"
	"sort"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// or perhaps a container struct with non-exported fields (for indices and such) and the Frames exported.
type MultiFrameSeries []*data.Frame

func NewMultiFrameSeries() *MultiFrameSeries {
	return &MultiFrameSeries{
		emptyFrameWithTypeMD(data.FrameTypeTimeSeriesMany),
	}
}

// values must be a numeric slice such as []int64, []float64, []*float64, etc or []bool / []*bool.
func (mfs *MultiFrameSeries) AddMetric(metricName string, l data.Labels, t []time.Time, values interface{}) error {
	var err error

	if mfs == nil || len(*mfs) == 0 {
		return fmt.Errorf("zero frames when calling AddMetric must call NewMultiFrameSeries first") // panic? maybe?
	}

	if !data.ValidFieldType(values) {
		return fmt.Errorf("type %T is not a valid data frame field type", values)
	}

	valueField := data.NewField(metricName, l, values)
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

	if len(*mfs) == 1 && len((*mfs)[0].Fields) == 0 { // update empty response placeholder frame
		(*mfs)[0].Fields = append((*mfs)[0].Fields, timeField, valueField)
	} else {
		frame := emptyFrameWithTypeMD(data.FrameTypeTimeSeriesMany)
		frame.Fields = append(frame.Fields, timeField, valueField)
		*mfs = append(*mfs, frame)
	}
	return err
}

func (mfs *MultiFrameSeries) SetMetricMD(metricName string, l data.Labels, fc data.FieldConfig) {
	panic("not implemented")
}

func (mfs *MultiFrameSeries) GetMetricRefs() ([]TimeSeriesMetricRef, []FrameFieldIndex) {
	if mfs == nil || len(*mfs) == 0 {
		return nil, nil // nil / nil == invalid
	}

	var ignoredFields []FrameFieldIndex
	var refs []TimeSeriesMetricRef

	if len(*mfs) == 1 {
		f := (*mfs)[0]
		if frameHasMetaType(f, data.FrameTypeTimeSeriesMany) && len(f.Fields) == 0 {
			return []TimeSeriesMetricRef{}, nil // non-nil empty slice / nil == valid "empty response"
		}
	}

	for frameIdx, frame := range *mfs {
		if frame == nil { // nil frames not valid
			ignoredFields = append(ignoredFields, FrameFieldIndex{frameIdx, -1})
			continue
		}

		ignoreAllFields := func() {
			if len(frame.Fields) == 0 {
				ignoredFields = append(ignoredFields, FrameFieldIndex{0, -1})
			}
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
			ignoreAllFields()
			continue
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
func (mfs *MultiFrameSeries) Validate() (ignoredFieldIndices []FrameFieldIndex, err error) {
	if mfs == nil || len(*mfs) == 0 {
		return nil, fmt.Errorf("must have at least one frame to be valid")
	}

	if len(*mfs) == 1 {
		f := (*mfs)[0]
		if frameHasMetaType(f, data.FrameTypeTimeSeriesMany) && len(f.Fields) == 0 {
			return nil, nil
		}
	}

	metricIndex := make(map[[2]string]struct{})

	for frameIdx, frame := range *mfs {
		if frame.Meta == nil || frame.Meta.Type != data.FrameTypeTimeSeriesMany {
			return nil, fmt.Errorf("frame %v is missing type indicator in frame metadata", frameIdx)
		}

		if len(frame.Fields) == 0 {
			ignoredFieldIndices = append(ignoredFieldIndices, FrameFieldIndex{frameIdx, -1})
			continue
		}

		if _, err := frame.RowLen(); err != nil {
			return nil, fmt.Errorf("frame %v has mismatched field lengths: %w", frameIdx, err)
		}

		timeFields := frame.TypeIndices(data.FieldTypeTime)

		// Must have []time.Time field (no nullable time)
		if len(timeFields) == 0 {
			return nil, fmt.Errorf("frame %v must have at least one time field but has %v", frameIdx, len(timeFields))
		}

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
					return nil, fmt.Errorf("frame %v has an unsorted time field", frameIdx)
				}
			}
		}

		valueFields := frame.TypeIndices(ValidValueFields()...)

		if len(valueFields) == 0 {
			return nil, fmt.Errorf("frame %v must have at least one value field but has %v", frameIdx, len(valueFields))
		} else {
			if len(valueFields) > 1 {
				for _, fieldIdx := range valueFields[1:] {
					ignoredFieldIndices = append(ignoredFieldIndices, FrameFieldIndex{frameIdx, fieldIdx})
				}
			}

			vField := frame.Fields[valueFields[0]]
			metricKey := [2]string{vField.Name, vField.Labels.String()}

			if _, ok := metricIndex[metricKey]; ok {
				return nil, fmt.Errorf("duplicate metrics found for metric name %q and labels %q", vField.Name, vField.Labels)
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

	sort.Sort(FrameFieldIndices(ignoredFieldIndices))

	return ignoredFieldIndices, nil
}
