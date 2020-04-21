package backend

import (
	"errors"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
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

func (f convertFromProtobuf) AppInstanceSettings(proto *pluginv2.AppInstanceSettings) *AppInstanceSettings {
	if proto == nil {
		return nil
	}

	return &AppInstanceSettings{
		JSONData:                proto.JsonData,
		DecryptedSecureJSONData: proto.DecryptedSecureJsonData,
		Updated:                 time.Unix(0, proto.LastUpdatedMS*int64(time.Millisecond)),
	}
}

func (f convertFromProtobuf) DataSourceInstanceSettings(proto *pluginv2.DataSourceInstanceSettings) *DataSourceInstanceSettings {
	if proto == nil {
		return nil
	}

	return &DataSourceInstanceSettings{
		ID:                      proto.Id,
		Name:                    proto.Name,
		URL:                     proto.Url,
		User:                    proto.User,
		Database:                proto.Database,
		BasicAuthEnabled:        proto.BasicAuthEnabled,
		BasicAuthUser:           proto.BasicAuthUser,
		JSONData:                proto.JsonData,
		DecryptedSecureJSONData: proto.DecryptedSecureJsonData,
		Updated:                 time.Unix(0, proto.LastUpdatedMS*int64(time.Millisecond)),
	}
}

func (f convertFromProtobuf) PluginContext(proto *pluginv2.PluginContext) PluginContext {
	return PluginContext{
		OrgID:                      proto.OrgId,
		PluginID:                   proto.PluginId,
		User:                       f.User(proto.User),
		AppInstanceSettings:        f.AppInstanceSettings(proto.AppInstanceSettings),
		DataSourceInstanceSettings: f.DataSourceInstanceSettings(proto.DataSourceInstanceSettings),
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

func (f convertFromProtobuf) QueryDataRequest(protoReq *pluginv2.QueryDataRequest) *QueryDataRequest {
	queries := make([]DataQuery, len(protoReq.Queries))
	for i, q := range protoReq.Queries {
		queries[i] = *f.DataQuery(q)
	}

	return &QueryDataRequest{
		PluginContext: f.PluginContext(protoReq.PluginContext),
		Headers:       protoReq.Headers,
		Queries:       queries,
	}
}

func (f convertFromProtobuf) QueryDataResponse(protoRes *pluginv2.QueryDataResponse) (*QueryDataResponse, error) {
	qdr := &QueryDataResponse{
		Responses: make(Responses, len(protoRes.Responses)),
	}
	for refID, res := range protoRes.Responses {
		frames, err := data.UnmarshalArrowFrames(res.Frames)
		if err != nil {
			return nil, err
		}
		dr := DataResponse{
			Frames: frames,
			Meta:   res.JsonMeta,
		}
		if res.Error != "" {
			dr.Error = errors.New(res.Error)
		}
		qdr.Responses[refID] = dr
	}
	return qdr, nil
}

func (f convertFromProtobuf) CallResourceRequest(protoReq *pluginv2.CallResourceRequest) *CallResourceRequest {
	headers := map[string][]string{}
	for k, values := range protoReq.Headers {
		headers[k] = values.Values
	}

	return &CallResourceRequest{
		PluginContext: f.PluginContext(protoReq.PluginContext),
		Path:          protoReq.Path,
		Method:        protoReq.Method,
		URL:           protoReq.Url,
		Headers:       headers,
		Body:          protoReq.Body,
	}
}

// HealthCheckRequest converts proto version to SDK version.
func (f convertFromProtobuf) HealthCheckRequest(protoReq *pluginv2.CheckHealthRequest) *CheckHealthRequest {
	return &CheckHealthRequest{
		PluginContext: f.PluginContext(protoReq.PluginContext),
	}
}
