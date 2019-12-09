package backend

import (
	"context"
	"encoding/json"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/dataframe"
	bproto "github.com/grafana/grafana-plugin-sdk-go/genproto/go/grafana_plugin"
)

// TimeRange represents a time range for a query.
type TimeRange struct {
	From time.Time
	To   time.Time
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
	Frames []*dataframe.Frame
}

// DatasourceQueryResult holds the results for a given query.
type DatasourceQueryResult struct {
	Error      string
	RefID      string
	MetaJSON   string
	DataFrames []*dataframe.Frame
}

// DataQueryHandler handles data source queries.
type DataQueryHandler interface {
	DataQuery(ctx context.Context, pc PluginConfig, headers map[string]string, queries []DataQuery) (DataQueryResponse, error)
}

func (p *backendPluginWrapper) DataQuery(ctx context.Context, req *bproto.DataQueryRequest) (*bproto.DataQueryResponse, error) {

	pc := pluginConfigFromProto(req.Config)

	var queries []DataQuery
	for _, q := range req.Queries {
		tr := TimeRange{
			From: time.Unix(0, q.TimeRange.FromEpochMs*int64(time.Millisecond)),
			To:   time.Unix(0, q.TimeRange.ToEpochMs*int64(time.Millisecond)),
		}
		queries = append(queries, DataQuery{
			RefID:         q.RefId,
			MaxDataPoints: q.MaxDataPoints,
			TimeRange:     tr,
			Interval:      time.Duration(q.IntervalMs) * time.Millisecond,
			JSON:          []byte(q.Json),
		})
	}

	resp, err := p.dataHandler.DataQuery(ctx, pc, req.Headers, queries)
	if err != nil {
		return nil, err
	}

	encodedFrames := make([][]byte, len(resp.Frames))
	for i, frame := range resp.Frames {
		encodedFrames[i], err = dataframe.MarshalArrow(frame)
		if err != nil {
			return nil, err
		}
	}

	return &bproto.DataQueryResponse{
		Frames: encodedFrames,
	}, nil
}
