package backend

import (
	"context"
	"encoding/json"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/dataframe"
	bproto "github.com/grafana/grafana-plugin-sdk-go/genproto/go/backend_plugin"
)

// DataQueryHandler handles data source queries.
type DataQueryHandler interface {
	DataQuery(ctx context.Context, pc PluginConfig, headers map[string]string, queries []DataQuery) (DataQueryResponse, error)
}

// DataQuery represents the query as sent from the frontend.
type DataQuery struct {
	RefID         string
	MaxDataPoints int64
	Interval      time.Duration
	TimeRange     TimeRange
	JSON          json.RawMessage
}

func dataQueryFromProtobuf(q *bproto.DataQuery) *DataQuery {
	return &DataQuery{
		RefID:         q.RefId,
		MaxDataPoints: q.MaxDataPoints,
		TimeRange:     timeRangeFromProtobuf(q.TimeRange),
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
func timeRangeFromProtobuf(tr *bproto.TimeRange) TimeRange {
	return TimeRange{
		From: time.Unix(0, tr.FromEpochMS*int64(time.Millisecond)),
		To:   time.Unix(0, tr.ToEpochMS*int64(time.Millisecond)),
	}
}

func (p *coreWrapper) DataQuery(ctx context.Context, req *bproto.DataQueryRequest) (*bproto.DataQueryResponse, error) {

	pc := pluginConfigFromProto(req.Config)

	queries := make([]DataQuery, len(req.Queries))
	for i, q := range req.Queries {
		queries[i] = *dataQueryFromProtobuf(q)
	}

	resp, err := p.handlers.DataQuery(ctx, pc, req.Headers, queries)
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
		Frames:   encodedFrames,
		Metadata: resp.Metadata,
	}, nil
}
