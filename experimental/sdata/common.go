package sdata

import (
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

func ValidValueFields() []data.FieldType {
	// TODO: not sure about bool (factor or value)
	return append(data.NumericFieldTypes(), []data.FieldType{data.FieldTypeBool, data.FieldTypeNullableBool}...)
}

// FrameFieldIndex is for referencing data that is not considered part of the metric data
// when the data is valid. Reason states why the field was not part of the metric data.
type FrameFieldIndex struct {
	FrameIdx int
	FieldIdx int    // -1 means no fields
	Reason   string // only meant for human consumption
}

type FrameFieldIndices []FrameFieldIndex

func (f FrameFieldIndices) Len() int {
	return len(f)
}

func (f FrameFieldIndices) Less(i, j int) bool {
	return f[i].FrameIdx < f[j].FrameIdx
}

func (f FrameFieldIndices) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}
