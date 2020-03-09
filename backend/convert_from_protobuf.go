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

// User converts proto version of user to SDK version
func (f convertFromProtobuf) User(user *pluginv2.User) *User {
	if user == nil {
		return nil
	}

	return &User{
		Login: user.Login,
		Name:  user.Name,
		Email: user.Email,
		Role:  user.Role,
	}
}

func (f convertFromProtobuf) DataSourceConfig(proto *pluginv2.DataSourceConfig) *DataSourceConfig {
	if proto == nil {
		return nil
	}

	return &DataSourceConfig{
		ID:               proto.Id,
		Name:             proto.Name,
		URL:              proto.Url,
		User:             proto.User,
		Database:         proto.Database,
		BasicAuthEnabled: proto.BasicAuthEnabled,
		BasicAuthUser:    proto.BasicAuthUser,
	}
}

func (f convertFromProtobuf) PluginConfig(proto *pluginv2.PluginConfig) PluginConfig {
	return PluginConfig{
		OrgID:                   proto.OrgId,
		PluginID:                proto.PluginId,
		PluginType:              proto.PluginType,
		JSONData:                proto.JsonData,
		DecryptedSecureJSONData: proto.DecryptedSecureJsonData,
		Updated:                 time.Unix(0, proto.UpdatedMS*int64(time.Millisecond)),
		DataSourceConfig:        f.DataSourceConfig(proto.DatasourceConfig),
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

func (f convertFromProtobuf) QueryDataRequest(protoReq *pluginv2.QueryDataRequest) *DataQueryRequest {
	queries := make([]DataQuery, len(protoReq.Queries))
	for i, q := range protoReq.Queries {
		queries[i] = *f.DataQuery(q)
	}

	return &DataQueryRequest{
		PluginConfig: f.PluginConfig(protoReq.Config),
		Headers:      protoReq.Headers,
		Queries:      queries,
		User:         f.User(protoReq.User),
	}
}

func (f convertFromProtobuf) QueryDataResponse(protoRes *pluginv2.QueryDataResponse) (*DataQueryResponse, error) {
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

func (f convertFromProtobuf) CallResourceRequest(protoReq *pluginv2.CallResourceRequest) *CallResourceRequest {
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
		User:         f.User(protoReq.User),
	}
}

// HealthCheckRequest converts proto version to SDK version.
func (f convertFromProtobuf) HealthCheckRequest(protoReq *pluginv2.CheckHealthRequest) *CheckHealthRequest {
	return &CheckHealthRequest{
		PluginConfig: f.PluginConfig(protoReq.Config),
	}
}
