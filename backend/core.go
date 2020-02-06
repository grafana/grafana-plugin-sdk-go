package backend

import (
	"context"
	"encoding/json"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/dataframe"
)

// InstanceSettings plugin instance settings.
type InstanceSettings struct {
	JSONData                json.RawMessage
	DecryptedSecureJSONData map[string]string
	Updated                 time.Time
}

// AppInstanceSettings app plugin instance settings.
type AppInstanceSettings struct {
	*InstanceSettings
}

// DataSourceInstanceSettings data source plugin instance settings.
type DataSourceInstanceSettings struct {
	*InstanceSettings
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
	OrgID              int64
	PluginID           string
	PluginType         string
	AppSettings        *AppInstanceSettings
	DataSourceSettings *DataSourceInstanceSettings
}

type DataQueryRequest struct {
	PluginConfig PluginConfig
	Headers      map[string]string
	Queries      []DataQuery
}

// DataQuery represents the query as sent from the frontend.
type DataQuery struct {
	RefID         string
	MaxDataPoints int64
	Interval      time.Duration
	TimeRange     TimeRange
	JSON          json.RawMessage
}

// DataQueryResponse holds the results for a given query.
type DataQueryResponse struct {
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
}

type CallResourceResponse struct {
	Status  int
	Headers map[string][]string
	Body    []byte
}

// CallResourceHandler handles resource calls.
type CallResourceHandler interface {
	CallResource(ctx context.Context, req *CallResourceRequest) (*CallResourceResponse, error)
}

// DataQueryHandler handles data source queries.
type DataQueryHandler interface {
	DataQuery(ctx context.Context, req *DataQueryRequest) (*DataQueryResponse, error)
}

// PluginHandlers is the collection of handlers that corresponds to the
// grpc "service BackendPlugin".
type PluginHandlers interface {
	DataQueryHandler
}
