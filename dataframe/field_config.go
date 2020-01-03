package dataframe

// FieldConfig represents the display properties for a field
type FieldConfig struct {
	Title      string `json:"title,omitempty"`
	Filterable *bool  `json:"filterable,omitempty"`

	// Numeric Options
	unit     string `json:"title,omitempty"`
	decimals *int16 `json:"title,omitempty"`
	min      *int64 `json:"title,omitempty"`
	max      *int64 `json:"title,omitempty"`

	// Convert input values into a display string
	mappings []*ValueMapping `json:"title,omitempty"`

	// Map numeric values to states
	thresholds *ThresholdsConfig `json:"title,omitempty"`

	// Map values to a display color
	color *FieldColor `json:"title,omitempty"`

	// Used when reducing field values
	nullValueMode *NullValueMode `json:"title,omitempty"`

	// The behavior when clicking on a result
	links []*DataLink `json:"title,omitempty"`

	// Alternative to empty string
	noValue *string `json:"noValue,omitempty"`

	// Panel Specific Values
	custom *map[string]interface{} `json:"custom,omitempty"`
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
