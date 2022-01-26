package data

// FieldMeta attaches structural metadata to each frame
type FieldMeta struct {
	// StepSize is the interval between each value.
	// If no values exist in the values array, this indictes mising data for that interval
	// Only supports non-zero step sizes
	StepSize float64 `json:"stepSize,omitempty"`

	// Path is a browsable path on the datasource.
	StepScale string `json:"stepScale,omitempty"`

	// Aggregation defines the separator pattern to decode a hiearchy. The default separator is '/'.
	Aggregation string `json:"aggregation,omitempty"`

	// MetricType is 'counter' | 'gauge' | 'summary';
	MetricType string `json:"metricType,omitempty"`
}

const (
	StepScaleLinear = "linear"
	StepScaleLog2   = "log2"
	StepScaleLog10  = "log10"

	FieldAggregationBefore = "before"
	FieldAggregationMiddle = "middle"
	FieldAggregationAfter  = "after"

	MetricTypeCounter = "counter"
	MetricTypeGauge   = "gauge"
	MetricTypeSummary = "summary"
)
