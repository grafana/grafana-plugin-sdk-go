package resource

import (
	"embed"
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

// QueryTypeDefinition is a kubernetes shaped object that represents a single query definition
type QueryTypeDefinition struct {
	ObjectMeta ObjectMeta `json:"metadata,omitempty"`

	Spec QueryTypeDefinitionSpec `json:"spec,omitempty"`
}

// QueryTypeDefinitionList is a kubernetes shaped object that represents a list of query types
// For simple data sources, there may be only a single query type, however when multiple types
// exist they must be clearly specified with distinct discriminator field+value pairs
type QueryTypeDefinitionList struct {
	Kind       string `json:"kind"`       // "QueryTypeDefinitionList",
	APIVersion string `json:"apiVersion"` // "query.grafana.app/v0alpha1",

	ObjectMeta `json:"metadata,omitempty"`

	Items []QueryTypeDefinition `json:"items"`
}

type QueryTypeDefinitionSpec struct {
	// Multiple schemas can be defined using discriminators
	Discriminators []DiscriminatorFieldValue `json:"discriminators,omitempty"`

	// Describe whe the query type is for
	Description string `json:"description,omitempty"`

	// The query schema represents the properties that can be sent to the API
	// In many cases, this may be the same properties that are saved in a dashboard
	// In the case where the save model is different, we must also specify a save model
	QuerySchema any `json:"querySchema"`

	// The save model defines properties that can be saved into dashboard or similar
	// These values are processed by frontend components and then sent to the api
	// When specified, this schema will be used to validate saved objects rather than
	// the query schema
	SaveModel any `json:"saveModel,omitempty"`

	// Examples (include a wrapper) ideally a template!
	Examples []QueryExample `json:"examples,omitempty"`

	// Changelog defines the changed from the previous version
	// All changes in the same version *must* be backwards compatible
	// Only notable changes will be shown here, for the full version history see git!
	Changelog []string `json:"changelog,omitempty"`
}

type QueryExample struct {
	// Version identifier or empty if only one exists
	Name string `json:"name,omitempty"`

	// An example value saved that can be saved in a dashboard
	SaveModel any `json:"saveModel,omitempty"`
}

type CommonQueryProperties struct {
	// RefID is the unique identifier of the query, set by the frontend call.
	RefID string `json:"refId,omitempty"`

	// Optionally define expected query result behavior
	ResultAssertions *ResultAssertions `json:"resultAssertions,omitempty"`

	// TimeRange represents the query range
	// NOTE: unlike generic /ds/query, we can now send explicit time values in each query
	// NOTE: the values for timeRange are not saved in a dashboard, they are constructed on the fly
	TimeRange *TimeRange `json:"timeRange,omitempty"`

	// The datasource
	Datasource *DataSourceRef `json:"datasource,omitempty"`

	// Deprecated -- use datasource ref instead
	DatasourceID int64 `json:"datasourceId,omitempty"`

	// QueryType is an optional identifier for the type of query.
	// It can be used to distinguish different types of queries.
	QueryType string `json:"queryType,omitempty"`

	// MaxDataPoints is the maximum number of data points that should be returned from a time series query.
	// NOTE: the values for maxDataPoints is not saved in the query model.  It is typically calculated
	// from the number of pixels visible in a visualization
	MaxDataPoints int64 `json:"maxDataPoints,omitempty"`

	// Interval is the suggested duration between time points in a time series query.
	// NOTE: the values for intervalMs is not saved in the query model.  It is typically calculated
	// from the interval required to fill a pixels in the visualization
	IntervalMS float64 `json:"intervalMs,omitempty"`

	// true if query is disabled (ie should not be returned to the dashboard)
	// NOTE: this does not always imply that the query should not be executed since
	// the results from a hidden query may be used as the input to other queries (SSE etc)
	Hide bool `json:"hide,omitempty"`
}

type DataSourceRef struct {
	// The datasource plugin type
	Type string `json:"type"`

	// Datasource UID
	UID string `json:"uid"`

	// ?? the datasource API version?  (just version, not the group? type | apiVersion?)
}

// TimeRange represents a time range for a query and is a property of DataQuery.
type TimeRange struct {
	// From is the start time of the query.
	From string `json:"from" jsonschema:"example=now-1h"`

	// To is the end time of the query.
	To string `json:"to" jsonschema:"example=now"`
}

// ResultAssertions define the expected response shape and query behavior.  This is useful to
// enforce behavior over time.  The assertions are passed to the query engine and can be used
// to fail queries *before* returning them to a client (select * from bigquery!)
type ResultAssertions struct {
	// Type asserts that the frame matches a known type structure.
	Type data.FrameType `json:"type,omitempty" jsonschema:"example=timeseries-wide,example=timeseries-long"`

	// TypeVersion is the version of the Type property. Versions greater than 0.0 correspond to the dataplane
	// contract documentation https://grafana.github.io/dataplane/contract/.
	TypeVersion data.FrameTypeVersion `json:"typeVersion"`

	// Maximum bytes that can be read -- if the query planning expects more then this, the query may fail fast
	MaxBytes int64 `json:"maxBytes,omitempty"`

	// Maximum frame count
	MaxFrames int64 `json:"maxFrames,omitempty"`
}

type DiscriminatorFieldValue struct {
	// DiscriminatorField is the field used to link behavior to this specific
	// query type.  It is typically "queryType", but can be another field if necessary
	Field string `json:"field"`

	// The discriminator value
	Value string `json:"value"`
}

// using any since this will often be enumerations
func NewDiscriminators(keyvals ...any) []DiscriminatorFieldValue {
	if len(keyvals)%2 != 0 {
		panic("values must be even")
	}
	dis := []DiscriminatorFieldValue{}
	for i := 0; i < len(keyvals); i += 2 {
		dis = append(dis, DiscriminatorFieldValue{
			Field: fmt.Sprintf("%v", keyvals[i]),
			Value: fmt.Sprintf("%v", keyvals[i+1]),
		})
	}
	return dis
}

//go:embed query.schema.json
var f embed.FS

// Get the cached feature list (exposed as a k8s resource)
func CommonQueryPropertiesSchema() (*spec.Schema, error) {
	body, err := f.ReadFile("query.schema.json")
	if err != nil {
		return nil, err
	}
	s := &spec.Schema{}
	err = s.UnmarshalJSON(body)
	return s, err
}
