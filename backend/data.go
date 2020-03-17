package backend

import (
	"context"
	"encoding/json"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type QueryDataRequest struct {
	PluginConfig PluginConfig
	Headers      map[string]string
	Queries      []DataQuery
	User         *User
}

// DataQuery represents the query as sent from the frontend.
type DataQuery struct {
	RefID         string
	MaxDataPoints int64
	Interval      time.Duration
	TimeRange     TimeRange
	JSON          json.RawMessage
	Model         interface{} // The parsed model (same data as in JSON)
}

// QueryDataResponse holds the results for a given query.
type QueryDataResponse struct {
	Frames   []*data.Frame
	Metadata map[string]string
}

// TimeRange represents a time range for a query.
type TimeRange struct {
	From time.Time
	To   time.Time
}

// QueryDataHandler handles data queries.
type QueryDataHandler interface {
	ParseDataSourceConfigModel(jsonBytes json.RawMessage, secureJSONData map[string]string) (interface{}, error)
	ParseQueryModel(jsonBytes json.RawMessage) (interface{}, error)
	QueryData(ctx context.Context, req *QueryDataRequest) (*QueryDataResponse, error)
}
