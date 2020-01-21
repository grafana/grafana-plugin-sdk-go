package convert

import (
	"encoding/json"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/dataframe"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

type FromProtobuf struct {
}

func FromProto() FromProtobuf {
	return FromProtobuf{}
}

func (f FromProtobuf) PluginConfig(proto *pluginv2.PluginConfig) backend.PluginConfig {
	return backend.PluginConfig{
		ID:       proto.Id,
		OrgID:    proto.OrgId,
		Name:     proto.Name,
		Type:     proto.Type,
		URL:      proto.Url,
		JSONData: json.RawMessage([]byte(proto.JsonData)),
	}
}

func (f FromProtobuf) TimeRange(proto *pluginv2.TimeRange) backend.TimeRange {
	return backend.TimeRange{
		From: time.Unix(0, proto.FromEpochMS*int64(time.Millisecond)),
		To:   time.Unix(0, proto.ToEpochMS*int64(time.Millisecond)),
	}
}

func (f FromProtobuf) DataQuery(proto *pluginv2.DataQuery) *backend.DataQuery {
	return &backend.DataQuery{
		RefID:         proto.RefId,
		MaxDataPoints: proto.MaxDataPoints,
		TimeRange:     f.TimeRange(proto.TimeRange),
		Interval:      time.Duration(proto.IntervalMS) * time.Millisecond,
		JSON:          []byte(proto.Json),
	}
}

func (f FromProtobuf) DataQueryRequest(protoReq *pluginv2.DataQueryRequest) *backend.DataQueryRequest {
	queries := make([]backend.DataQuery, len(protoReq.Queries))
	for i, q := range protoReq.Queries {
		queries[i] = *f.DataQuery(q)
	}
	return &backend.DataQueryRequest{
		PluginConfig: f.PluginConfig(protoReq.Config),
		Headers:      protoReq.Headers,
		Queries:      queries,
	}
}

func (f FromProtobuf) DataQueryResponse(protoRes *pluginv2.DataQueryResponse) (*backend.DataQueryResponse, error) {
	frames := make([]*dataframe.Frame, len(protoRes.Frames))
	var err error
	for i, encodedFrame := range protoRes.Frames {
		frames[i], err = dataframe.UnmarshalArrow(encodedFrame)
		if err != nil {
			return nil, err
		}
	}
	return &backend.DataQueryResponse{Metadata: protoRes.Metadata, Frames: frames}, nil
}

func (f FromProtobuf) CallResourceRequest(protoReq *pluginv2.CallResource_Request) *backend.ResourceRequestContext {
	return backend.NewResourceRequestContext(f.PluginConfig(protoReq.Config), protoReq.Params)
}
