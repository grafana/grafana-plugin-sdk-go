package sdata

import (
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// Less sure about this interface.
// Also, probably would need to make a type with methods that wrap the interface so methods can be added
// without breaking changes - if we even want an interface like this
// But for now helps illustrate at least
type TimeSeriesCollection interface {
	AddMetric(metricName string, l data.Labels, t []time.Time, values interface{}) error
	SetMetricMD(metricName string, l data.Labels, fc data.FieldConfig)
	AppendMetricValue(metricName string, l data.Labels, t time.Time, value interface{}) error
	InsertMetricValue(metricName string, l data.Labels, t time.Time, value interface{}) error
	Validate() (isEmpty bool, errors []error)
	AsWideFrameSeries() *WideFrameSeries
	AsMultiFrameSeries() *MultiFrameSeries
}

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
}

// Appends to metric
// Error is if value is not same number type
// Error if t is before previous value since time must be sorted
func (mfs *MultiFrameSeries) AppendMetricValue(metricName string, l data.Labels, t time.Time, value interface{}) error {
	return nil
}

// Like append but inserts in sorted time order
func (mfs *MultiFrameSeries) InsertMetricValue(metricName string, l data.Labels, t time.Time, value interface{}) error {
	return nil
}

// Validates data conforms to schema, don't think it will be called normally in the course of running a plugin, but needs to exist.
// Currently this is strict in the sense that consumers must support all valid instances. Consumers may support invalid instances
// depending on the circumstances.
func (mfs *MultiFrameSeries) Validate() (isEmpty bool, errors []error) {
	if mfs == nil || len(*mfs) == 0 {
		// Unless we create a container (and expose it in our responses) that can hold the type(s) for the frames it contains,
		// anything empty probably needs be considered "valid" for the type. Else we have a requirement to create at least one frame (eww).
		return true, nil
	}

	metricIndex := make(map[[2]string]struct{})

	for fIdx, frame := range *mfs {
		if frame.Meta == nil || frame.Meta.Type != data.FrameTypeTimeSeriesMany {
			errors = append(errors, fmt.Errorf("frame %v is missing type indicator in frame metadata", fIdx))
		}

		if len(frame.Fields) == 0 {
			// an individual frame with no fields is an empty series is valid.
			continue
		}

		if _, err := frame.RowLen(); err != nil {
			errors = append(errors, fmt.Errorf("frame %v has mismatched field lengths: %w", fIdx, err))
		}

		timeFields := frame.TypeIndices(data.FieldTypeTime)

		// Must have []time.Time field (no nullable time)
		if len(timeFields) != 1 {
			errors = append(errors, fmt.Errorf("frame %v must have exactly 1 time field but has %v", fIdx, len(timeFields)))
		} else {
			// Validate time Field is sorted in ascending (oldest to newest) order
			timeField := frame.Fields[timeFields[0]]
			if timeField.Len() > 1 {
				for tIdx := 1; tIdx < timeField.Len(); tIdx++ {
					prevTime := timeField.At(tIdx - 1).(time.Time)
					curTime := timeField.At(tIdx).(time.Time)
					if curTime.Before(prevTime) {
						errors = append(errors, fmt.Errorf("frame %v has an unsorted time field", fIdx))
						break
					}
				}
			}

			// numeric or bool can be a "value"
			numericFields := frame.TypeIndices(data.NumericFieldTypes()...)
			boolFields := frame.TypeIndices(data.FieldTypeBool, data.FieldTypeNullableBool)

			valueFieldCount := len(numericFields) + len(boolFields)
			if valueFieldCount != 1 {
				errors = append(errors, fmt.Errorf("frame %v must have exactly 1 value field but has %v", fIdx, valueFieldCount))
			} else {
				vFieldIdx := 0

				if len(numericFields) == 1 {
					vFieldIdx = numericFields[0]
				} else {
					vFieldIdx = boolFields[0]
				}

				vField := frame.Fields[vFieldIdx]

				metricKey := [2]string{vField.Name, vField.Labels.String()}
				if _, ok := metricIndex[metricKey]; ok {
					errors = append(errors, fmt.Errorf("duplicate metrics found for metric name %q and labels %q", vField.Name, vField.Labels))
				} else {
					metricIndex[metricKey] = struct{}{}
				}
			}
		}
	}

	return false, errors
}

// to fullfill interface, returns itself
func (mfs *MultiFrameSeries) AsMultiFrameSeries() *MultiFrameSeries {
	return nil
}

// Converts to wide frame, will manipulate data. Generally not to be used with data sources.
func (mfs *MultiFrameSeries) AsWideFrameSeries() *WideFrameSeries {
	return nil
}

// need to think about pointers here and elsewhere
type WideFrameSeries data.Frame

func (wf *WideFrameSeries) AddMetric(metricName string, l data.Labels, t []time.Time, values interface{}) error {
	return nil
}

func (wf *WideFrameSeries) SetMetricMD(metricName string, l data.Labels, fc data.FieldConfig) {

}

func (wf *WideFrameSeries) AppendMetricValue(metricName string, l data.Labels, t time.Time, value interface{}) error {
	return nil
}

func (wf *WideFrameSeries) InsertMetricValue(metricName string, l data.Labels, t time.Time, value interface{}) error {
	return nil
}

func (wf *WideFrameSeries) Validate() (isEmpty bool, err []error) {
	return false, nil
}

// Converts to multi-frame, generally not to be used by data sources.
func (wf *WideFrameSeries) AsMultiFrameSeries() *MultiFrameSeries {
	return nil
}

// to fullfill interface, returns itself
func (wf *WideFrameSeries) AsWideFrameSeries() *WideFrameSeries {
	return nil
}
