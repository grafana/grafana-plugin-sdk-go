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

/*

Rules
- Whenever an error is returned, there are no ignored fields returned
- Must have at least one frame
- The first frame may have no fields, but then it must be the only frame (empty response case)
- The first frame must be valid or will error, additional invalid frames with the type indicator will error,
    frames without type indicator are ignored
- A valid individual Frame (in the non empty case) has:
	- The type indicator
	- a []time.Time field (not []*time.Time) sorted from oldest to newest
	- a numeric value field
- Any nil Frames or Fields will cause an error (e.g. [Frame, Frame, nil, Frame] or [nil])
- If any frame has fields within the frame of different lengths, an error will be returned
- If validateData is true, duplicate labels and sorted time fields will error, otherwise only the schema/metadata is checked.
- If all frames and their fields are ignored, and it is not the empty response case, an error is returned

When things get ignored
- Frames that don't have the type indicator as long as they are not first
- Fields when the type indicator is present and the frame is valid (e.g. has both time and value fields):
  - String, Additional Time Fields, Additional Value fields

*/
func (mfs *MultiFrameSeries) Validate(validateData bool) (ignoredFields []FrameFieldIndex, err error) {
	if mfs == nil || len(*mfs) == 0 {
		return nil, fmt.Errorf("must have at least one frame to be valid")
	}

	if len(*mfs) == 1 { // empty typed response (single frame, with type indicator, and no fields)
		f := (*mfs)[0]
		if f == nil {
			return nil, fmt.Errorf("frame %v is nil which is not valid")
		}
		if frameHasMetaType(f, data.FrameTypeTimeSeriesMany) && len(f.Fields) == 0 {
			return nil, nil
		} else {
			return nil, fmt.Errorf("single frame response is missing a type indicator")
		}
	}

	metricIndex := make(map[[2]string]struct{})

	for frameIdx, frame := range *mfs {
		if frame == nil {
			return nil, fmt.Errorf("frame %v is nil which is not valid")
		}

		ignoreAllFields := func(reason string) {
			for fieldIdx := range frame.Fields {
				ignoredFields = append(ignoredFields, FrameFieldIndex{frameIdx, fieldIdx, reason})
			}
		}

		if frame.Meta == nil || frame.Meta.Type != data.FrameTypeTimeSeriesMany {
			if frameIdx == 0 {
				return nil, fmt.Errorf("first frame must have the many/multi type indicator in frame metadata", frameIdx)
			}
			ignoreAllFields("no type indicator in frame or metadata is not type many/multi")
			continue
		}

		if len(frame.Fields) == 0 { // note: single frame with no fields is acceptable, but is returned before this
			if frameIdx == 0 {
				return nil, fmt.Errorf("first frame must have non-zero fields if not the only frame", frameIdx)
			}
			ignoredFields = append(ignoredFields, FrameFieldIndex{frameIdx, -1, "frame has no fields and is not the only frame"})
			continue
		}

		if _, err := frame.RowLen(); err != nil {
			return nil, fmt.Errorf("frame %v has mismatched field lengths: %w", frameIdx, err)
		}

		for fieldIdx, field := range frame.Fields {
			if field == nil {
				return nil, fmt.Errorf("frame %v has a nil field at %v", frameIdx, fieldIdx)
			}
		}
		timeFields := frame.TypeIndices(data.FieldTypeTime)

		// Must have []time.Time field (no nullable time)
		if len(timeFields) == 0 {
			return nil, fmt.Errorf("frame %v must have at least one time field but has 0", frameIdx)
		}

		if len(timeFields) > 1 {
			for _, fieldIdx := range timeFields[1:] {
				ignoredFields = append(ignoredFields, FrameFieldIndex{frameIdx, fieldIdx, "additional time field"})
			}
		}

		// Validate time Field is sorted in ascending (oldest to newest) order
		timeField := frame.Fields[timeFields[0]]
		if validateData {
			sorted, err := timeIsSorted(timeField)
			if err != nil {
				return nil, fmt.Errorf("frame %v has an malformed time field", frameIdx)
			}
			if !sorted {
				return nil, fmt.Errorf("frame %v has an unsorted time field", frameIdx)
			}
		}

		valueFields := frame.TypeIndices(ValidValueFields()...)
		if len(valueFields) == 0 {
			return nil, fmt.Errorf("frame %v must have at least one value field but has %v", frameIdx, len(valueFields))
		} else {
			if len(valueFields) > 1 {
				for _, fieldIdx := range valueFields[1:] {
					ignoredFields = append(ignoredFields, FrameFieldIndex{frameIdx, fieldIdx})
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
			ignoredFields = append(ignoredFields, FrameFieldIndex{frameIdx, fieldIdx})
		}
	}

	sort.Sort(FrameFieldIndices(ignoredFields))

	return ignoredFields, nil
}
