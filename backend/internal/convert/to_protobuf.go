package convert

import (
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/dataframe"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

type ToProtobuf struct {
}

func ToProto() ToProtobuf {
	return ToProtobuf{}
}

func (t ToProtobuf) PluginConfig(pc backend.PluginConfig) *pluginv2.PluginConfig {
	return &pluginv2.PluginConfig{
		Id:       pc.ID,
		OrgId:    pc.OrgID,
		Name:     pc.Name,
		Type:     pc.Type,
		Url:      pc.URL,
		JsonData: string(pc.JSONData),
	}
}

func (t ToProtobuf) TimeRange(tr backend.TimeRange) *pluginv2.TimeRange {
	return &pluginv2.TimeRange{
		FromEpochMS: tr.From.UnixNano() / int64(time.Millisecond),
		ToEpochMS:   tr.To.UnixNano() / int64(time.Millisecond),
	}
}

func (t ToProtobuf) HealthStatus(status backend.HealthStatus) pluginv2.CheckHealth_Response_HealthStatus {
	switch status {
	case backend.HealthStatusUnknown:
		return pluginv2.CheckHealth_Response_UNKNOWN
	case backend.HealthStatusOk:
		return pluginv2.CheckHealth_Response_OK
	case backend.HealthStatusError:
		return pluginv2.CheckHealth_Response_ERROR
	}
	panic("unsupported protobuf health status type in sdk")
}

func (t ToProtobuf) CheckHealthResponse(res *backend.CheckHealthResult) *pluginv2.CheckHealth_Response {
	return &pluginv2.CheckHealth_Response{
		Status: t.HealthStatus(res.Status),
		Info:   res.Info,
	}
}

func (t ToProtobuf) DataQuery(q backend.DataQuery) *pluginv2.DataQuery {
	return &pluginv2.DataQuery{
		RefId:         q.RefID,
		MaxDataPoints: q.MaxDataPoints,
		IntervalMS:    q.Interval.Milliseconds(),
		TimeRange:     t.TimeRange(q.TimeRange),
		Json:          q.JSON,
	}
}

func (t ToProtobuf) DataQueryRequest(req *backend.DataQueryRequest) *pluginv2.DataQueryRequest {
	queries := make([]*pluginv2.DataQuery, len(req.Queries))
	for i, q := range req.Queries {
		queries[i] = t.DataQuery(q)
	}
	return &pluginv2.DataQueryRequest{
		Config:  t.PluginConfig(req.PluginConfig),
		Headers: req.Headers,
		Queries: queries,
	}
}

func (t ToProtobuf) DataQueryResponse(res *backend.DataQueryResponse) (*pluginv2.DataQueryResponse, error) {
	encodedFrames := make([][]byte, len(res.Frames))
	var err error
	for i, frame := range res.Frames {
		encodedFrames[i], err = dataframe.MarshalArrow(frame)
		if err != nil {
			return nil, err
		}
	}

	return &pluginv2.DataQueryResponse{
		Frames:   encodedFrames,
		Metadata: res.Metadata,
	}, nil
}

func (t ToProtobuf) RouteMethod(rm backend.RouteMethod) pluginv2.Resource_Route_Method {
	switch rm {
	case backend.RouteMethodAny:
		return pluginv2.Resource_Route_ANY
	case backend.RouteMethodGet:
		return pluginv2.Resource_Route_GET
	case backend.RouteMethodPut:
		return pluginv2.Resource_Route_PUT
	case backend.RouteMethodPost:
		return pluginv2.Resource_Route_POST
	case backend.RouteMethodDelete:
		return pluginv2.Resource_Route_DELETE
	case backend.RouteMethodPatch:
		return pluginv2.Resource_Route_PATCH
	}
	panic("unsupported protobuf resource route method type in sdk")
}

func (t ToProtobuf) Route(r *backend.Route) *pluginv2.Resource_Route {
	return &pluginv2.Resource_Route{
		Path:   r.Path,
		Method: t.RouteMethod(r.Method),
	}
}

func (t ToProtobuf) Resource(r *backend.Resource) *pluginv2.Resource {
	res := &pluginv2.Resource{
		Path:   r.Path,
		Routes: []*pluginv2.Resource_Route{},
	}

	for _, route := range r.Routes {
		res.Routes = append(res.Routes, t.Route(route))
	}

	return res
}

func (t ToProtobuf) ResourceMap(rm backend.ResourceMap) map[string]*pluginv2.Resource {
	res := map[string]*pluginv2.Resource{}
	for name, resource := range rm {
		res[name] = t.Resource(resource)
	}

	return res
}
