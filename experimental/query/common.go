package query

import "embed"

type CommonQueryProperties struct {
	// RefID is the unique identifier of the query, set by the frontend call.
	RefID string `json:"refId,omitempty"`

	// TimeRange represents the query range
	// NOTE: unlike generic /ds/query, we can now send explicit time values in each query
	TimeRange *TimeRange `json:"timeRange,omitempty"`

	// The datasource
	Datasource *DataSourceRef `json:"datasource,omitempty"`

	// Deprecated -- use datasource ref instead
	DatasourceId int64 `json:"datasourceId,omitempty"`

	// QueryType is an optional identifier for the type of query.
	// It can be used to distinguish different types of queries.
	QueryType string `json:"queryType,omitempty"`

	// MaxDataPoints is the maximum number of data points that should be returned from a time series query.
	MaxDataPoints int64 `json:"maxDataPoints,omitempty"`

	// Interval is the suggested duration between time points in a time series query.
	IntervalMS float64 `json:"intervalMs,omitempty"`

	// true if query is disabled (ie should not be returned to the dashboard)
	// Note this does not always imply that the query should not be executed since
	// the results from a hidden query may be used as the input to other queries (SSE etc)
	Hide bool `json:"hide,omitempty"`
}

type DataSourceRef struct {
	// The datasource plugin type
	Type string `json:"type"`

	// Datasource UID
	UID string `json:"uid"`

	// ?? the datasource API version
	// ApiVersion string `json:"apiVersion"`
}

// TimeRange represents a time range for a query and is a property of DataQuery.
type TimeRange struct {
	// From is the start time of the query.
	From string `json:"from"`

	// To is the end time of the query.
	To string `json:"to"`
}

// GenericDataQuery is a replacement for `dtos.MetricRequest` with more explicit typing
type GenericDataQuery struct {
	CommonQueryProperties `json:",inline"`

	// Additional Properties (that live at the root)
	Additional map[string]any `json:",inline"`
}

//go:embed common.jsonschema
var f embed.FS

// Get the cached feature list (exposed as a k8s resource)
func GetCommonJSONSchema() []byte {
	body, _ := f.ReadFile("common.jsonschema")
	return body
}
