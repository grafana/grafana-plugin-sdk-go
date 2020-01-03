package dataframe

// FieldConfig represents the display properties for a field
type FieldConfig struct {
	Title      string `json:"title,omitempty"`
	Filterable *bool  `json:"filterable,omitempty"`

	// Numeric Options
	Unit     string `json:"unit,omitempty"`
	Decimals *int16 `json:"decimals,omitempty"`
	Min      *int64 `json:"min,omitempty"`
	Max      *int64 `json:"max,omitempty"`

	// Convert input values into a display string
	Mappings []*ValueMapping `json:"mappings,omitempty"`

	// Map numeric values to states
	Thresholds *ThresholdsConfig `json:"thresholds,omitempty"`

	// Map values to a display color
	Color *FieldColor `json:"color,omitempty"`

	// Used when reducing field values
	NullValueMode *NullValueMode `json:"nullValueMode,omitempty"`

	// The behavior when clicking on a result
	Links []*DataLink `json:"links,omitempty"`

	// Alternative to empty string
	NoValue string `json:"noValue,omitempty"`

	// Panel Specific Values
	Custom *map[string]interface{} `json:"custom,omitempty"`
}

// NullValueMode say how the UI should show null values
type NullValueMode string

const (
	// Null show null values
	Null NullValueMode = "null"
	// Ignore null values
	Ignore = "connected"
	// AsZero show null as zero
	AsZero = "null as zero"
)

// Null = 'null',
// Ignore = 'connected',
// AsZero = 'null as zero',

// ValueMapping convert input value to something else
type ValueMapping struct {
	// anything???
}

// ThresholdsConfig setup thresholds
type ThresholdsConfig struct {
	// anything???
}

// FieldColor configure field color
type FieldColor struct {
	// anything???
}

// DataLink define what
type DataLink struct {
	// anything???
}
