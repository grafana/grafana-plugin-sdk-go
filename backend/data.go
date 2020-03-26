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
}

// QueryDataResponse holds the results for a given query.
type QueryDataResponse struct {
	Responses []DataResponse
	Metadata  map[string]string
}

type DataResponse struct {
	RefID  string
	Frames []*data.Frame
	Meta   QueryResultMeta
	Error  string
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

type QueryResultMeta struct {
	Custom interface{}
}

// type QueryResultMetaNotice struct {
// 	Severity NoticeSeverity
// 	Text     string
// 	URL      string
// 	Inspect  InspectType
// }

// type NoticeSeverity int

// const (
// 	NoticeSeverityInfo NoticeSeverity = iota
// 	NoticeSeverityWarning
// 	NoticeSeverityError
// )

// type InspectType int

// const (
// 	InspectTypeMeta InspectType = iota
// 	InspectTypeError
// 	InspectTypeData
// 	InspectTypeStats
// )
