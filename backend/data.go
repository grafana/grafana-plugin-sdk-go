package backend

import (
	"context"
	"encoding/json"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// QueryDataHandler handles data queries.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifer).
// The QueryDataResponse contains a map of RefID to the response for each query.
type QueryDataHandler interface {
	QueryData(ctx context.Context, req *QueryDataRequest) (*QueryDataResponse, error)
}

// QueryDataRequest contains a single request which contains multiple queries.
// It is the input type for a QueryData call.
type QueryDataRequest struct {
	PluginConfig PluginConfig
	Headers      map[string]string
	Queries      []DataQuery
	User         *User // User is information about the grafana-server user that made the request.
}

// DataQuery represents a single query as sent from the frontend.
// A slice of DataQuery makes up the Queries property of a QueryDataRequest.
type DataQuery struct {
	RefID         string          // RefID is the unique identifer of the query, set by the frontend call.
	MaxDataPoints int64           // MaxDataPoints is the maximum number of datapoints that should be returned from a time series query.
	Interval      time.Duration   // Interval is the suggested duration between time points in a time series query.
	TimeRange     TimeRange       // TimeRange is the Start and End of the query as sent by the frontend.
	JSON          json.RawMessage // JSON is the raw JSON query and includes the above properties as well as custom properties.
}

// QueryDataResponse contains the results from a QueryDataRequest.
// It is the return type of a QueryData call.
type QueryDataResponse struct {
	Responses map[string]*DataResponse // Responses is a map of RefIDs (Unique Query ID) to *DataResponse.
	Metadata  map[string]string
}

// DataResponse contains the results from a DataQuery.
// A map of RefIDs (unique query identifers) to this type makes up the Responses property of a QueryDataResponse.
// The Error property is used to allow for partial success responses from the containing QueryDataResponse.
type DataResponse struct {
	Frames []*data.Frame   // The data returned from the Query. Each Frame repeats the RefID.
	Meta   json.RawMessage // Meta contains a custom JSON object for custom response metadata that is passed to the frontend.
	Error  error           // Error is a property to be set if the the corresponding DataQuery has an error.
}

// TimeRange represents a time range for a query and is a property of DataQuery.
type TimeRange struct {
	From time.Time // From is the start time of the query.
	To   time.Time // To is the end time of the query.
}

// Duration returns a time.Duration representing the ammount of time between From and To.
func (tr TimeRange) Duration() time.Duration {
	return tr.To.Sub(tr.From)
}
