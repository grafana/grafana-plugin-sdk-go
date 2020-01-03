package dataframe

// FieldConfig represents the display properties for a field
// This struct needs to match the frontend component defined in:
// https://github.com/grafana/grafana/blob/master/packages/grafana-data/src/types/dataFrame.ts#L23
type FieldConfig struct {
	Title      string `json:"title,omitempty"`
	Filterable *bool  `json:"filterable,omitempty"`

	// Numeric Options
	Unit     string `json:"unit,omitempty"`
	Decimals *int16 `json:"decimals,omitempty"`
	Min      *int64 `json:"min,omitempty"`
	Max      *int64 `json:"max,omitempty"`

	// Convert input values into a display string
	Mappings []ValueMapping `json:"mappings,omitempty"`

	// Map numeric values to states
	Thresholds *ThresholdsConfig `json:"thresholds,omitempty"`

	// Map values to a display color
	Color map[string]interface{} `json:"color,omitempty"`

	// Used when reducing field values
	NullValueMode *NullValueMode `json:"nullValueMode,omitempty"`

	// The behavior when clicking on a result
	Links []DataLink `json:"links,omitempty"`

	// Alternative to empty string
	NoValue string `json:"noValue,omitempty"`

	// Panel Specific Values
	Custom map[string]interface{} `json:"custom,omitempty"`
}

// NullValueMode say how the UI should show null values
type NullValueMode string

const (
	// Null show null values
	Null NullValueMode = "null"
	// Ignore null values
	Ignore NullValueMode = "connected"
	// AsZero show null as zero
	AsZero NullValueMode = "null as zero"
)

// MappingType value or range
type MappingType int8

const (
	// ValueToText map a value to text
	ValueToText MappingType = 1

	// RangeToText map a range to text
	RangeToText MappingType = 2
)

// ValueMapping convert input value to something else
type ValueMapping struct {
	ID       int16       `json:"id"`
	Operator string      `json:"operator"`
	Text     string      `json:"title"`
	Type     MappingType `json:"type"`

	// Only valid for MappingType == ValueMap
	Value string `json:"value,omitempty"`

	// Only valid for MappingType == RangeMap
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
}

// DataLink define what
type DataLink struct {
	Title       string `json:"title,omitempty"`
	TargetBlank bool   `json:"targetBlank,omitempty"`
	URL         string `json:"url,omitempty"`
}

// ThresholdsConfig setup thresholds
type ThresholdsConfig struct {
	mode ThresholdsMode `json:"mode"`
  
	// Must be sorted by 'value', first value is always -Infinity
	steps []Threshold  `json:"steps"`
}

type Threshold  struct {
	Value *int64 `json:"min,omitempty"` // First value is always -Infinity serialize to null
	Color string `json:"color,omitempty"`
	State string `json:"state,omitempty"`
}

type ThresholdsMode = string
const (
	Absolute ThresholdsMode = 'absolute'
	Percentage ThresholdsMode = 'percentage'
)