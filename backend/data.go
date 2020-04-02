package backend

import (
	"context"
	"encoding/json"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// QueryDataRequest contains a single request which contains multiple queries.
type QueryDataRequest struct {
	PluginConfig PluginConfig
	Headers      map[string]string
	Queries      []DataQuery
	User         *User
}

// DataQuery represents a single query as sent from the frontend.
type DataQuery struct {
	RefID         string
	MaxDataPoints int64
	Interval      time.Duration
	TimeRange     TimeRange
	JSON          json.RawMessage
}

// QueryDataResponse contains the results from a QueryDataRequest.
type QueryDataResponse struct {
	Responses map[string]*DataResponse
	Metadata  map[string]string
}

// DataResponse contains the results from a DataQuery.
type DataResponse struct {
	Frames []*data.Frame
	Meta   json.RawMessage
	Error  error
}

// TimeRange represents a time range for a query.
type TimeRange struct {
	From time.Time
	To   time.Time
}

// QueryDataHandler handles data queries.
type QueryDataHandler interface {
	QueryData(ctx context.Context, req *QueryDataRequest) (*QueryDataResponse, error)
}
