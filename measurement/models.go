package measurement

import "github.com/grafana/grafana-plugin-sdk-go/data"

// Measurement is a single measurement value.
type Measurement struct {
	// Name of the measurement.
	Name string `json:"name,omitempty"`

	// Time is the measurement time. Units are usually ms, but depends on the channel
	Time int64 `json:"time,omitempty"`

	// Values is the measurement's values. The value type is typically number or string.
	Values map[string]interface{} `json:"values,omitempty"`

	// Config is an optional list of field configs.
	Config map[string]data.FieldConfig `json:"config,omitempty"`

	// Labels are applied to all values.
	Labels map[string]string `json:"labels,omitempty"`
}

// Action defines what should happen when you send a list of measurements.
type Action string

const (
	// ActionAppend means new values should be added to a client buffer.  This is the default action
	ActionAppend Action = "append"

	// ActionReplace means new values should replace any existing values.
	ActionReplace Action = "replace"

	// ActionClear means all existing values should be remoed before adding.
	ActionClear Action = "clear"
)

// Batch is a collection of measurements all sent at once.
type Batch struct {
	// Action is the action in question, the default is append.
	Action Action `json:"action,omitempty"`

	// Measurements is the array of measurements.
	Measurements []Measurement `json:"measurements,omitempty"`

	// Capacity is the suggested size of the client buffer
	Capacity int64 `json:"capacity,omitempty"`
}
