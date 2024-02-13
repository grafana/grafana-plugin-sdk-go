package expr

import "github.com/grafana/grafana-plugin-sdk-go/data"

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

func (*ResampleQuery) ExpressionQueryType() QueryType {
	return QueryTypeReduce
}

func (q *ResampleQuery) Variables() []string {
	return []string{q.Expression}
}
