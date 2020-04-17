package backend

import (
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

type convertToProtobuf struct {
}

func toProto() convertToProtobuf {
	return convertToProtobuf{}
}

// User converts SDK version of user to proto version
func (t convertToProtobuf) User(user *User) *pluginv2.User {
	if user == nil {
		return nil
	}

	return &pluginv2.User{
		Login: user.Login,
		Name:  user.Name,
		Email: user.Email,
		Role:  user.Role,
	}
}

func (t convertToProtobuf) AppInstanceSettings(s *AppInstanceSettings) *pluginv2.AppInstanceSettings {
	if s == nil {
		return nil
	}

	return &pluginv2.AppInstanceSettings{
		JsonData:                s.JSONData,
		DecryptedSecureJsonData: s.DecryptedSecureJSONData,
		LastUpdatedMS:           s.Updated.UnixNano() / int64(time.Millisecond),
	}
}

func (t convertToProtobuf) DataSourceInstanceSettings(s *DataSourceInstanceSettings) *pluginv2.DataSourceInstanceSettings {
	if s == nil {
		return nil
	}

	return &pluginv2.DataSourceInstanceSettings{
		Id:                      s.ID,
		Name:                    s.Name,
		Url:                     s.URL,
		User:                    s.User,
		Database:                s.Database,
		BasicAuthEnabled:        s.BasicAuthEnabled,
		BasicAuthUser:           s.BasicAuthUser,
		JsonData:                s.JSONData,
		DecryptedSecureJsonData: s.DecryptedSecureJSONData,
		LastUpdatedMS:           s.Updated.UnixNano() / int64(time.Millisecond),
	}
}

func (t convertToProtobuf) PluginContext(pCtx PluginContext) *pluginv2.PluginContext {
	return &pluginv2.PluginContext{
		OrgId:                      pCtx.OrgID,
		PluginId:                   pCtx.PluginID,
		User:                       t.User(pCtx.User),
		AppInstanceSettings:        t.AppInstanceSettings(pCtx.AppInstanceSettings),
		DataSourceInstanceSettings: t.DataSourceInstanceSettings(pCtx.DataSourceInstanceSettings),
	}
}

func (t convertToProtobuf) TimeRange(tr TimeRange) *pluginv2.TimeRange {
	return &pluginv2.TimeRange{
		FromEpochMS: tr.From.UnixNano() / int64(time.Millisecond),
		ToEpochMS:   tr.To.UnixNano() / int64(time.Millisecond),
	}
}

func (t convertToProtobuf) HealthStatus(status HealthStatus) pluginv2.CheckHealthResponse_HealthStatus {
	switch status {
	case HealthStatusUnknown:
		return pluginv2.CheckHealthResponse_UNKNOWN
	case HealthStatusOk:
		return pluginv2.CheckHealthResponse_OK
	case HealthStatusError:
		return pluginv2.CheckHealthResponse_ERROR
	}
	panic("unsupported protobuf health status type in sdk")
}

func (t convertToProtobuf) CheckHealthResponse(res *CheckHealthResult) *pluginv2.CheckHealthResponse {
	return &pluginv2.CheckHealthResponse{
		Status:      t.HealthStatus(res.Status),
		Message:     res.Message,
		JsonDetails: res.JSONDetails,
	}
}

func (t convertToProtobuf) DataQuery(q DataQuery) *pluginv2.DataQuery {
	return &pluginv2.DataQuery{
		RefId:         q.RefID,
		MaxDataPoints: q.MaxDataPoints,
		IntervalMS:    q.Interval.Milliseconds(),
		TimeRange:     t.TimeRange(q.TimeRange),
		Json:          q.JSON,
	}
}

func (t convertToProtobuf) QueryDataRequest(req *QueryDataRequest) *pluginv2.QueryDataRequest {
	queries := make([]*pluginv2.DataQuery, len(req.Queries))
	for i, q := range req.Queries {
		queries[i] = t.DataQuery(q)
	}
	return &pluginv2.QueryDataRequest{
		Context: t.PluginContext(req.PluginContext),
		Headers: req.Headers,
		Queries: queries,
	}
}

func (t convertToProtobuf) QueryDataResponse(res *QueryDataResponse) (*pluginv2.QueryDataResponse, error) {
	pQDR := &pluginv2.QueryDataResponse{
		Responses: make(map[string]*pluginv2.DataResponse, len(res.Responses)),
	}
	for refID, dr := range res.Responses {
		encodedFrames, err := dr.Frames.MarshalArrow()
		if err != nil {
			return nil, err
		}
		pDR := pluginv2.DataResponse{
			Frames:   encodedFrames,
			JsonMeta: dr.Meta,
		}
		if dr.Error != nil {
			pDR.Error = dr.Error.Error()
		}
		pQDR.Responses[refID] = &pDR
	}

	return pQDR, nil
}

func (t convertToProtobuf) CallResourceResponse(resp *CallResourceResponse) *pluginv2.CallResourceResponse {
	headers := map[string]*pluginv2.StringList{}

	for key, values := range resp.Headers {
		headers[key] = &pluginv2.StringList{Values: values}
	}

	return &pluginv2.CallResourceResponse{
		Headers: headers,
		Code:    int32(resp.Status),
		Body:    resp.Body,
	}
}
