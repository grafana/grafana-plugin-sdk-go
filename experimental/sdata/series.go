package sdata

import (
	"sort"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type TimeSeriesCollectionReader interface {
	// Validate will error if the data is invalid according to its type rules.
	// validateData will check for duplicate metrics and sorting and costs more resources.
	// If the data is valid, then any data that was not part of the time series data
	// is also returned as ignoredFieldIndices.
	Validate(validateData bool) (ignoredFieldIndices []FrameFieldIndex, err error)

	// GetMetricRefs runs validate without validateData. If the data is valid, then
	// []TimeSeriesMetricRef is returned from reading as well as any ignored data. If invalid,
	// then an error is returned, and not refs or ignoredFieldIndices are returned.
	GetMetricRefs() (refs []TimeSeriesMetricRef, ignoredFieldIndices []FrameFieldIndex, err error)
}

func ValidValueFields() []data.FieldType {
	return append(data.NumericFieldTypes(), []data.FieldType{data.FieldTypeBool, data.FieldTypeNullableBool}...)
}

// TimeSeriesMetricRef is for reading and contains the data for an individual
// time series. In the cases of the Multi and Wide formats, the Fields are pointers
// to the data in the original frame. In the case of Long, new fields are constructed.
type TimeSeriesMetricRef struct {
	TimeField  *data.Field
	ValueField *data.Field
	// TODO: RefID string
	// TODO: Pointer to frame meta?
}

// FrameFieldIndex is for referencing data that is not considered part of the metric data
// when the data is valid. Reason states why the field was not part of the metric data.
type FrameFieldIndex struct {
	FrameIdx int
	FieldIdx int
	Reason   string // only meant for human consumption
}


func (m TimeSeriesMetricRef) GetMetricName() string {
	if m.ValueField != nil {
		return m.ValueField.Name
	}
	return ""
}

// TODO GetFQMetric (or something, Names + Labels)

func (m TimeSeriesMetricRef) GetLabels() data.Labels {
	if m.ValueField != nil {
		return m.ValueField.Labels
	}
	return nil
}

type FrameFieldIndices []FrameFieldIndex

func (f FrameFieldIndices) Len() int {
	return len(f)
}

func (f FrameFieldIndices) Less(i, j int) bool {
	if f[i].FrameIdx == f[j].FrameIdx {
		return f[i].FieldIdx < f[j].FieldIdx
	}
	return f[i].FrameIdx < f[j].FrameIdx
}

func (f FrameFieldIndices) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

func sortTimeSeriesMetricRef(refs []TimeSeriesMetricRef) {
	sort.SliceStable(refs, func(i, j int) bool {
		iRef := refs[i]
		jRef := refs[j]

		if iRef.GetMetricName() < jRef.GetMetricName() {
			return true
		}
		if iRef.GetMetricName() > jRef.GetMetricName() {
			return false
		}

		// If here Names are equal, next sort based on if there are labels.

		if iRef.GetLabels() == nil && jRef.GetLabels() == nil {
			return true // no labels first
		}
		if iRef.GetLabels() == nil && jRef.GetLabels() != nil {
			return true
		}
		if iRef.GetLabels() != nil && jRef.GetLabels() == nil {
			return false
		}

		return iRef.GetLabels().String() < jRef.GetLabels().String()
	})
}
