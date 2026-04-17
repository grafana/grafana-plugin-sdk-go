package example

import "github.com/grafana/grafana-plugin-sdk-go/data"

// +enum
type QueryType string

const (
	// Math query type
	QueryTypeMath QueryType = "math"

	// Reduce query type
	QueryTypeReduce QueryType = "reduce"

	// Reduce query type
	QueryTypeResample QueryType = "resample"
)

type MathQuery struct {
	// General math expression
	Expression string `json:"expression" jsonschema:"minLength=1,example=$A + 1,example=$A/$B"`
}

type ReduceQuery struct {
	// Reference to other query results
	Expression string `json:"expression"`

	// The reducer
	Reducer ReducerID `json:"reducer"`

	// Reducer Options
	Settings ReduceSettings `json:"settings"`
}

type ReduceSettings struct {
	// Non-number reduce behavior
	Mode ReduceMode `json:"mode"`

	// Only valid when mode is replace
	ReplaceWithValue *float64 `json:"replaceWithValue,omitempty"`
}

// The reducer function
// +enum
type ReducerID string

const (
	// The sum
	ReducerSum ReducerID = "sum"
	// The mean
	ReducerMean  ReducerID = "mean"
	ReducerMin   ReducerID = "min"
	ReducerMax   ReducerID = "max"
	ReducerCount ReducerID = "count"
	ReducerLast  ReducerID = "last"
)

// Non-Number behavior mode
// +enum
type ReduceMode string

// Dummy value makes sure the enum extraction logic is valid
const DummyValueA = "dummyA"

const (
	// Drop non-numbers
	ReduceModeDrop ReduceMode = "dropNN"

	// Replace non-numbers
	ReduceModeReplace ReduceMode = "replaceNN"
)

// Dummy value makes sure the enum extraction logic is valid
const DummyValueB = "dummyB"

// QueryType = resample
type ResampleQuery struct {
	// The math expression
	Expression string `json:"expression"`

	// A time duration string
	Window string `json:"window"`

	// The reducer
	Downsampler string `json:"downsampler"`

	// The reducer
	Upsampler string `json:"upsampler"`

	LoadedDimensions *data.Frame `json:"loadedDimensions"`
}
