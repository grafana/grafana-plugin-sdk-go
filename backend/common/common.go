package common

import (
	"encoding/json"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/dataframe"
	bproto "github.com/grafana/grafana-plugin-sdk-go/genproto/go/grafana_plugin"
)

// PluginConfig holds configuration for the queried plugin.
type PluginConfig struct {
	ID       int64
	OrgID    int64
	Name     string
	Type     string
	URL      string
	JSONData json.RawMessage
}

// PluginConfigFromProto converts the generated protobuf PluginConfig to this
// package's PluginConfig.
func PluginConfigFromProto(pc *bproto.PluginConfig) PluginConfig {
	return PluginConfig{
		ID:       pc.Id,
		OrgID:    pc.OrgId,
		Name:     pc.Name,
		Type:     pc.Type,
		URL:      pc.Url,
		JSONData: json.RawMessage(pc.JsonData),
	}
}

// DataQuery represents the query as sent from the frontend.
type DataQuery struct {
	RefID         string
	MaxDataPoints int64
	Interval      time.Duration
	TimeRange     TimeRange
	JSON          json.RawMessage
}

func DataQueryFromProtobuf(q *bproto.DataQuery) *DataQuery {
	return &DataQuery{
		RefID:         q.RefId,
		MaxDataPoints: q.MaxDataPoints,
		TimeRange:     TimeRangeFromProtobuf(q.TimeRange),
		Interval:      time.Duration(q.IntervalMS) * time.Millisecond,
		JSON:          []byte(q.Json),
	}
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

// TimeRangeFromProtobuf converts the generated protobuf TimeRange to this
// package's FetchInfo.
func TimeRangeFromProtobuf(tr *bproto.TimeRange) TimeRange {
	return TimeRange{
		From: time.Unix(0, tr.FromEpochMS*int64(time.Millisecond)),
		To:   time.Unix(0, tr.ToEpochMS*int64(time.Millisecond)),
	}
}
