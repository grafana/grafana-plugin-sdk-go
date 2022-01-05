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
	Validate() (bool, []error)
	AsWideFrameSeries() *WideFrameSeries
	AsMultiFrameSeries() *MultiFrameSeries
}

// or perhaps a container struct with non-exported fields (for indicies and such) and the Frames exported.
type MultiFrameSeries []*data.Frame

// values must be a numeric slice such as []int64, []float64, []*float64, etc or []bool / []*bool, else this will panic.
func (mfs *MultiFrameSeries) AddMetric(metricName string, l data.Labels, t []time.Time, values interface{}) error {
	var err error

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

// Validates data conforms to schema, don't think it will be called normally in the course of running a plugin, but needs to exist
func (mfs *MultiFrameSeries) Validate() (bool, []error) {
	return false, nil
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

func (wf *WideFrameSeries) Validate() (bool, []error) {
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
