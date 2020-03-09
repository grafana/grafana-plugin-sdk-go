package backend

import (
	"context"
	"encoding/json"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/dataframe"
)

// User represents the Grafana user.
type User struct {
	Login string
	Name  string
	Email string
	Role  string
}

// DataSourceConfig configuration for a datasource plugin.
type DataSourceConfig struct {
	ID               int64
	Name             string
	URL              string
	User             string
	Database         string
	BasicAuthEnabled bool
	BasicAuthUser    string
}

// PluginConfig configuration for a plugin.
type PluginConfig struct {
	OrgID                   int64
	PluginID                string
	PluginType              string
	JSONData                json.RawMessage
	DecryptedSecureJSONData map[string]string
	Updated                 time.Time
	DataSourceConfig        *DataSourceConfig
}

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
}

// QueryDataResponse holds the results for a given query.
type QueryDataResponse struct {
	Frames   []*dataframe.Frame
	Metadata map[string]string
}

// TimeRange represents a time range for a query.
type TimeRange struct {
	From time.Time
	To   time.Time
}

type CallResourceRequest struct {
	PluginConfig PluginConfig
	Path         string
	Method       string
	URL          string
	Headers      map[string][]string
	Body         []byte
	User         *User
}

type CallResourceResponse struct {
	Status  int
	Headers map[string][]string
	Body    []byte
}

// CallResourceResponseSender used for sending resource call responses.
type CallResourceResponseSender interface {
	Send(*CallResourceResponse) error
}

// CallResourceHandler handles resource calls.
type CallResourceHandler interface {
	CallResource(ctx context.Context, req *CallResourceRequest, sender CallResourceResponseSender) error
}

// QueryDataHandler handles data queries.
type QueryDataHandler interface {
	QueryData(ctx context.Context, req *QueryDataRequest) (*QueryDataResponse, error)
}
