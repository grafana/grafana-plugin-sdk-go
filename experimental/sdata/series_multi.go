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

func (mfs *MultiFrameSeries) GetMetricRefs() ([]TimeSeriesMetricRef, []FrameFieldIndex, error) {
	return validateAndGetRefsMulti(mfs, false, true)
}

/*
Generally, when the type indicator in present on a frame, we become stricter on what the shape of the frame can be.
However, there are still degrees of freedom: - extra frames without the indicator, or extra fields when the indicator is present.


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
	_, ignoredFields, err = validateAndGetRefsMulti(mfs, validateData, false)
	return ignoredFields, err
}

// wrap validation and metric ref fetching together for consistency in validation
// not sure if "getRefs" should be there or not, or if should just always be built and ignored.
func validateAndGetRefsMulti(mfs *MultiFrameSeries, validateData, getRefs bool) (refs []TimeSeriesMetricRef, ignoredFields []FrameFieldIndex, err error) {
	if mfs == nil || len(*mfs) == 0 {
		return nil, nil, fmt.Errorf("must have at least one frame to be valid")
	}

	if len(*mfs) == 1 { // empty typed response (single frame, with type indicator, and no fields)
		f := (*mfs)[0]
		if f == nil {
			return nil, nil, fmt.Errorf("frame 0 is nil which is invalid")
		}
		if !frameHasType(f, data.FrameTypeTimeSeriesMany) {
			return nil, nil, fmt.Errorf("single frame response is missing a type indicator")
		}
		if len(f.Fields) == 0 {
			return []TimeSeriesMetricRef{}, nil, nil
		}
	}

	metricIndex := make(map[[2]string]struct{})

	for frameIdx, frame := range *mfs {
		if frame == nil {
			return nil, nil, fmt.Errorf("frame %v is nil which is not valid", frameIdx)
		}

		ignoreAllFields := func(reason string) {
			for fieldIdx := range frame.Fields {
				ignoredFields = append(ignoredFields, FrameFieldIndex{frameIdx, fieldIdx, reason})
			}
		}

		if !frameHasType(frame, data.FrameTypeTimeSeriesMany) {
			if frameIdx == 0 {
				return nil, nil, fmt.Errorf("first frame must have the many/multi type indicator in frame metadata")
			}
			ignoreAllFields("no type indicator in frame or metadata is not type many/multi")
			continue
		}

		if len(frame.Fields) == 0 { // note: single frame with no fields is acceptable, but is returned before this
			return nil, nil, fmt.Errorf("frame %v has zero or null fields which is invalid when more than one frame", frameIdx)
		}

		if err := malformedFrameCheck(frameIdx, frame); err != nil {
			return nil, nil, err
		}

		timeField, ignoredTimedFields, err := seriesCheckSelectTime(frameIdx, frame)
		if err != nil {
			return nil, nil, err
		}
		if ignoredTimedFields != nil {
			ignoredFields = append(ignoredFields, ignoredTimedFields...)
		}

		valueFields := frame.TypeIndices(ValidValueFields()...)
		if len(valueFields) == 0 {
			return nil, nil, fmt.Errorf("frame %v must have at least one value field but has %v", frameIdx, len(valueFields))
		} else {
			if len(valueFields) > 1 {
				for _, fieldIdx := range valueFields[1:] {
					ignoredFields = append(ignoredFields, FrameFieldIndex{frameIdx, fieldIdx, "additional numeric value field"})
				}
			}

			vField := frame.Fields[valueFields[0]]
			metricKey := [2]string{vField.Name, vField.Labels.String()}

			if _, ok := metricIndex[metricKey]; ok && validateData {
				return nil, nil, fmt.Errorf("duplicate metrics found for metric name %q and labels %q", vField.Name, vField.Labels)
			}
			metricIndex[metricKey] = struct{}{}
		}

		if validateData {
			sorted, err := timeIsSorted(timeField)
			if err != nil {
				return nil, nil, fmt.Errorf("frame %v has an malformed time field", 0)
			}
			if !sorted {
				return nil, nil, fmt.Errorf("frame %v has an unsorted time field", 0)
			}
		}

		if getRefs {
			refs = append(refs, TimeSeriesMetricRef{
				TimeField:  timeField,
				ValueField: frame.Fields[valueFields[0]],
			})
		}

		// TODO this is fragile if new types are added
		otherFields := frame.TypeIndices(data.FieldTypeNullableTime, data.FieldTypeString, data.FieldTypeNullableString)
		for _, fieldIdx := range otherFields {
			ignoredFields = append(ignoredFields, FrameFieldIndex{frameIdx, fieldIdx, fmt.Sprintf("unsupported field type %v", frame.Fields[fieldIdx].Type())})
		}
	}

	if len(metricIndex) == 0 {
		return nil, nil, fmt.Errorf("no metrics in response and not an empty response")
	}

	sort.Sort(FrameFieldIndices(ignoredFields))

	return refs, ignoredFields, nil
}
