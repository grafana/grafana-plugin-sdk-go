package backend

import (
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/dataframe"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

type convertFromProtobuf struct {
}

func fromProto() convertFromProtobuf {
	return convertFromProtobuf{}
}

func (f convertFromProtobuf) PluginConfig(proto *pluginv2.PluginConfig) PluginConfig {
	return PluginConfig{
		OrgID:              proto.OrgId,
		PluginID:           proto.PluginId,
		PluginType:         proto.PluginType,
		AppSettings:        f.AppInstanceSettings(proto.GetApp()),
		DataSourceSettings: f.DataSourceInstanceSettings(proto.GetDataSource()),
	}
}

func (f convertFromProtobuf) AppInstanceSettings(proto *pluginv2.PluginConfig_AppInstanceSettings) *AppInstanceSettings {
	if proto == nil {
		return nil
	}

	return &AppInstanceSettings{
		InstanceSettings: &InstanceSettings{
			JSONData:                proto.JsonData,
			DecryptedSecureJSONData: proto.DecryptedSecureJsonData,
			Updated:                 time.Unix(0, proto.UpdatedMS*int64(time.Millisecond)),
		},
	}
}

func (f convertFromProtobuf) DataSourceInstanceSettings(proto *pluginv2.PluginConfig_DataSourceInstanceSettings) *DataSourceInstanceSettings {
	if proto == nil {
		return nil
	}

	return &DataSourceInstanceSettings{
		InstanceSettings: &InstanceSettings{
			JSONData:                proto.JsonData,
			DecryptedSecureJSONData: proto.DecryptedSecureJsonData,
			Updated:                 time.Unix(0, proto.UpdatedMS*int64(time.Millisecond)),
		},
		ID:               proto.Id,
		Name:             proto.Name,
		URL:              proto.Url,
		User:             proto.User,
		Database:         proto.Database,
		BasicAuthEnabled: proto.BasicAuthEnabled,
		BasicAuthUser:    proto.BasicAuthUser,
	}
}

func (f convertFromProtobuf) TimeRange(proto *pluginv2.TimeRange) TimeRange {
	return TimeRange{
		From: time.Unix(0, proto.FromEpochMS*int64(time.Millisecond)),
		To:   time.Unix(0, proto.ToEpochMS*int64(time.Millisecond)),
	}
}

func (f convertFromProtobuf) DataQuery(proto *pluginv2.DataQuery) *DataQuery {
	return &DataQuery{
		RefID:         proto.RefId,
		MaxDataPoints: proto.MaxDataPoints,
		TimeRange:     f.TimeRange(proto.TimeRange),
		Interval:      time.Duration(proto.IntervalMS) * time.Millisecond,
		JSON:          []byte(proto.Json),
	}
}

func (f convertFromProtobuf) DataQueryRequest(protoReq *pluginv2.DataQueryRequest) *DataQueryRequest {
	queries := make([]DataQuery, len(protoReq.Queries))
	for i, q := range protoReq.Queries {
		queries[i] = *f.DataQuery(q)
	}
	return &DataQueryRequest{
		PluginConfig: f.PluginConfig(protoReq.Config),
		Headers:      protoReq.Headers,
		Queries:      queries,
	}
}

func (f convertFromProtobuf) DataQueryResponse(protoRes *pluginv2.DataQueryResponse) (*DataQueryResponse, error) {
	frames := make([]*dataframe.Frame, len(protoRes.Frames))
	var err error
	for i, encodedFrame := range protoRes.Frames {
		frames[i], err = dataframe.UnmarshalArrow(encodedFrame)
		if err != nil {
			return nil, err
		}
	}
	return &DataQueryResponse{Metadata: protoRes.Metadata, Frames: frames}, nil
}

func (f convertFromProtobuf) CallResourceRequest(protoReq *pluginv2.CallResource_Request) *CallResourceRequest {
	headers := map[string][]string{}
	for k, values := range protoReq.Headers {
		headers[k] = values.Values
	}

	return &CallResourceRequest{
		PluginConfig: f.PluginConfig(protoReq.Config),
		Path:         protoReq.Path,
		Method:       protoReq.Method,
		URL:          protoReq.Url,
		Headers:      headers,
		Body:         protoReq.Body,
	}
}
