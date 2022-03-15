package sdata

import (
	"sort"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// Notes on Defining Empty Responses and Empty Metrics
//   - Typed "No Metrics" (Empty Response) == At least one Frame with type indicator with no fields
//   - Typed "Empty Metric" == Fields Present of proper types, Zero Length fields
// Need to think of how these map in Go Code -> Data frames (passing over the wire), and JS
// For this Go code:
//   - Make sure GetMetricRefs can capture "No Metrics" correctly and
//       that is the default (e.g. add NewX funcs)
//   - Make sure that information is kept over wire

type TimeSeriesCollectionReader interface {
	// TODO: Split Validate into ValidateSchema and ValidateData (or something like this).
	// The difference being if reads the contents of fields for validity. Which is mainly
	// "is time sorted?". Although perhaps duplicate labels as well (they exist on field header),
	// but are data and not metadata.
	// ValidateSchema will check
	Validate() (ignoredFieldIndices []FrameFieldIndex, err error)
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
	FrameIdx int
	FieldIdx int    // -1 means no fields (Frame is nil or Fields are nil)
	Reason   string // only meant for human consumption
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
