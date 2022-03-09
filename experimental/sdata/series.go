package sdata

import (
	"sort"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type TimeSeriesCollectionReader interface {
	Validate() (isEmpty bool, ignoredFieldIndices []FrameFieldIndex, errors []error)
	GetMetricRefs() ([]TimeSeriesMetricRef, []FrameFieldIndex)
}

func ValidValueFields() []data.FieldType {
	return append(data.NumericFieldTypes(), []data.FieldType{data.FieldTypeBool, data.FieldTypeNullableBool}...)
}

// I am not sure about this but want to get the idea down
type TimeSeriesMetricRef struct {
	TimeField  *data.Field
	ValueField *data.Field
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

type FrameFieldIndex struct {
	FrameIdx int // -1 means nil Frame
	FieldIdx int // -1 means no fields
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
