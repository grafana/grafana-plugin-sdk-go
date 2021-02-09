package backend

import (
	"context"
	"encoding/json"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// QueryDataHandler handles data queries.
type QueryDataHandler interface {
	// QueryData handles multiple queries and returns multiple responses.
	// req contains the queries []DataQuery (where each query contains RefID as a unique identifier).
	// The QueryDataResponse contains a map of RefID to the response for each query, and each response
	// contains Frames ([]*Frame).
	//
	// The Frames' RefID property, when it is an empty string, will be automatically set to
	// the RefID in QueryDataResponse.Responses map. This is done before the QueryDataResponse is
	// sent to Grafana. Therefore one does not need to be set that property on frames when using this method.
	QueryData(ctx context.Context, req *QueryDataRequest) (*QueryDataResponse, error)
}

// QueryDataHandlerFunc is an adapter to allow the use of
// ordinary functions as backend.QueryDataHandler. If f is a function
// with the appropriate signature, QueryDataHandlerFunc(f) is a
// Handler that calls f.
type QueryDataHandlerFunc func(ctx context.Context, req *QueryDataRequest) (*QueryDataResponse, error)

// QueryData calls fn(ctx, req).
func (fn QueryDataHandlerFunc) QueryData(ctx context.Context, req *QueryDataRequest) (*QueryDataResponse, error) {
	return fn(ctx, req)
}

// QueryDataRequest contains a single request which contains multiple queries.
// It is the input type for a QueryData call.
type QueryDataRequest struct {
	PluginContext PluginContext     `json:"plugin_context,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	Queries       []DataQuery       `json:"queries,omitempty"`
}

// DataQuery represents a single query as sent from the frontend.
// A slice of DataQuery makes up the Queries property of a QueryDataRequest.
type DataQuery struct {
	// RefID is the unique identifier of the query, set by the frontend call.
	RefID string `json:"ref_id,omitempty"`

	// QueryType is an optional identifier for the type of query.
	// It can be used to distinguish different types of queries.
	QueryType string `json:"query_type,omitempty"`

	// MaxDataPoints is the maximum number of datapoints that should be returned from a time series query.
	MaxDataPoints int64 `json:"max_data_points,omitempty"`

	// Interval is the suggested duration between time points in a time series query.
	Interval time.Duration `json:"interval,omitempty"`

	// TimeRange is the Start and End of the query as sent by the frontend.
	TimeRange TimeRange `json:"time_range,omitempty"`

	// JSON is the raw JSON query and includes the above properties as well as custom properties.
	JSON json.RawMessage `json:"json,omitempty"`
}

// QueryDataResponse contains the results from a QueryDataRequest.
// It is the return type of a QueryData call.
type QueryDataResponse struct {
	// Responses is a map of RefIDs (Unique Query ID) to *DataResponse.
	Responses Responses `json:"responses,omitempty"`
}

// NewQueryDataResponse returns a QueryDataResponse with the Responses property initialized.
func NewQueryDataResponse() *QueryDataResponse {
	return &QueryDataResponse{
		Responses: make(Responses),
	}
}

// Responses is a map of RefIDs (Unique Query ID) to DataResponses.
// The QueryData method the QueryDataHandler method will set the RefId
// property on the DataRespones' frames based on these RefIDs.
type Responses map[string]DataResponse

// DataResponse contains the results from a DataQuery.
// A map of RefIDs (unique query identifers) to this type makes up the Responses property of a QueryDataResponse.
// The Error property is used to allow for partial success responses from the containing QueryDataResponse.
type DataResponse struct {
	// The data returned from the Query. Each Frame repeats the RefID.
	Frames data.Frames `json:"frames,omitempty"`

	// Error is a property to be set if the the corresponding DataQuery has an error.
	Error error `json:"error,omitempty"`
}

// TimeRange represents a time range for a query and is a property of DataQuery.
type TimeRange struct {
	// From is the start time of the query.
	From time.Time `json:"from,omitempty"`

	// To is the end time of the query.
	To time.Time `json:"to,omitempty"`
}

// Duration returns a time.Duration representing the amount of time between From and To.
func (tr TimeRange) Duration() time.Duration {
	return tr.To.Sub(tr.From)
}
