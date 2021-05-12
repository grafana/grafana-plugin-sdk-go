package data

// MappingType see https://github.com/grafana/grafana/blob/main/packages/grafana-data/src/types/valueMapping.ts
type MappingType string

const (
	ValueToText  MappingType = "value"
	RangeToText  MappingType = "range"
	SpecialValue MappingType = "special"
)

type SpecialValueMatch string

const (
	SpecialValueTrue       SpecialValueMatch = "true"
	SpecialValueFalse      SpecialValueMatch = "false"
	SpecialValueNull       SpecialValueMatch = "null"
	SpecialValueNaN        SpecialValueMatch = "nan"
	SpecialValueNullAndNaN SpecialValueMatch = "null+nan"
	SpecialValueEmpty      SpecialValueMatch = "empty"
)

// ValueMappingResult is the results from mapping a value
type ValueMappingResult struct {
	Text  string `json:"text,omitempty"`
	Color string `json:"color,omitempty"`
	Index int    `json:"index,omitempty"` // just used ofr ui ordering
}

// ValueMapping convert input value to something else
type ValueMapping struct {
	Type MappingType `json:"type"`

	// Only valid for MappingType == ValueMap
	Options interface{} `json:"options,omitempty"`
}

type SpecialValueMappingOptions struct {
	Match  SpecialValueMatch  `json:"match"`
	Result ValueMappingResult `json:"result"`
}

type RangeMapOptions struct {
	From   *float64           `json:"from"`
	To     *float64           `json:"to"`
	Result ValueMappingResult `json:"result"`
}

type ValueMapOptions = map[string]ValueMappingResult
