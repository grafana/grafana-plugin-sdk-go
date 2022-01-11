package backend

import (
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// ConvertToProtobuf has a collection of methods for converting the autogenerated
// protobuf go code to our SDK objects. This object exists to attach a collection
// of conversion methods to.
//
// This is used internally by the SDK and inside Grafana-server, plugin authors should not
// need this functionality.
type ConvertToProtobuf struct{}

// ToProto returns a new ConvertToProtobuf.
func ToProto() ConvertToProtobuf {
	return ConvertToProtobuf{}
}

// User converts the SDK version of a User to the protobuf version.
func (t ConvertToProtobuf) User(user *User) *pluginv2.User {
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

// AppInstanceSettings converts the SDK version of an AppInstanceSettings to the protobuf version.
func (t ConvertToProtobuf) AppInstanceSettings(s *AppInstanceSettings) *pluginv2.AppInstanceSettings {
	if s == nil {
		return nil
	}

	return &pluginv2.AppInstanceSettings{
		JsonData:                s.JSONData,
		DecryptedSecureJsonData: s.DecryptedSecureJSONData,
		LastUpdatedMS:           s.Updated.UnixNano() / int64(time.Millisecond),
	}
}

// DataSourceInstanceSettings converts the SDK version of a DataSourceInstanceSettings to the protobuf version.
func (t ConvertToProtobuf) DataSourceInstanceSettings(s *DataSourceInstanceSettings) *pluginv2.DataSourceInstanceSettings {
	if s == nil {
		return nil
	}

	return &pluginv2.DataSourceInstanceSettings{
		Id:                      s.ID,
		Uid:                     s.UID,
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

// PluginContext converts the SDK version of a PluginContext to the protobuf version.
func (t ConvertToProtobuf) PluginContext(pluginCtx PluginContext) *pluginv2.PluginContext {
	return &pluginv2.PluginContext{
		OrgId:                      pluginCtx.OrgID,
		PluginId:                   pluginCtx.PluginID,
		User:                       t.User(pluginCtx.User),
		AppInstanceSettings:        t.AppInstanceSettings(pluginCtx.AppInstanceSettings),
		DataSourceInstanceSettings: t.DataSourceInstanceSettings(pluginCtx.DataSourceInstanceSettings),
	}
}

// TimeRange converts the SDK version of a TimeRange to the protobuf version.
func (t ConvertToProtobuf) TimeRange(tr TimeRange) *pluginv2.TimeRange {
	return &pluginv2.TimeRange{
		FromEpochMS: tr.From.UnixNano() / int64(time.Millisecond),
		ToEpochMS:   tr.To.UnixNano() / int64(time.Millisecond),
	}
}

// HealthStatus converts the SDK version of a HealthStatus to the protobuf version.
func (t ConvertToProtobuf) HealthStatus(status HealthStatus) pluginv2.CheckHealthResponse_HealthStatus {
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

// CollectUsageStatsResponse converts the SDK version of a CollectUsageStatsResponse to the protobuf version.
func (t ConvertToProtobuf) CollectUsageStatsResponse(res *CollectUsageStatsResponse) *pluginv2.CollectUsageStatsResponse {
	return &pluginv2.CollectUsageStatsResponse{
		Stats: res.Stats,
	}
}

// CheckHealthResponse converts the SDK version of a CheckHealthResponse to the protobuf version.
func (t ConvertToProtobuf) CheckHealthResponse(res *CheckHealthResult) *pluginv2.CheckHealthResponse {
	return &pluginv2.CheckHealthResponse{
		Status:      t.HealthStatus(res.Status),
		Message:     res.Message,
		JsonDetails: res.JSONDetails,
	}
}

// DataQuery converts the SDK version of a DataQuery to the protobuf version.
func (t ConvertToProtobuf) DataQuery(q DataQuery) *pluginv2.DataQuery {
	return &pluginv2.DataQuery{
		RefId:         q.RefID,
		QueryType:     q.QueryType,
		MaxDataPoints: q.MaxDataPoints,
		IntervalMS:    q.Interval.Milliseconds(),
		TimeRange:     t.TimeRange(q.TimeRange),
		Json:          q.JSON,
	}
}

// QueryDataRequest converts the SDK version of a QueryDataRequest to the protobuf version.
func (t ConvertToProtobuf) QueryDataRequest(req *QueryDataRequest) *pluginv2.QueryDataRequest {
	queries := make([]*pluginv2.DataQuery, len(req.Queries))
	for i, q := range req.Queries {
		queries[i] = t.DataQuery(q)
	}
	return &pluginv2.QueryDataRequest{
		PluginContext: t.PluginContext(req.PluginContext),
		Headers:       req.Headers,
		Queries:       queries,
	}
}

// QueryDataResponse converts the SDK version of a QueryDataResponse to the protobuf version.
// It will set the RefID on the frames to the RefID key in Responses if a Frame's
// RefId property is an empty string.
func (t ConvertToProtobuf) QueryDataResponse(res *QueryDataResponse) (*pluginv2.QueryDataResponse, error) {
	pQDR := &pluginv2.QueryDataResponse{
		Responses: make(map[string]*pluginv2.DataResponse, len(res.Responses)),
	}
	for refID, dr := range res.Responses {
		for _, f := range dr.Frames {
			if f.RefID == "" {
				f.RefID = refID
			}
		}
		encodedFrames, err := dr.Frames.MarshalArrow()
		if err != nil {
			return nil, err
		}
		pDR := pluginv2.DataResponse{
			Frames: encodedFrames,
		}
		if dr.Error != nil {
			pDR.Error = dr.Error.Error()
		}
		pQDR.Responses[refID] = &pDR
	}

	return pQDR, nil
}

// CallResourceResponse converts the SDK version of a CallResourceResponse to the protobuf version.
func (t ConvertToProtobuf) CallResourceResponse(resp *CallResourceResponse) *pluginv2.CallResourceResponse {
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

// CallResourceRequest converts the SDK version of a CallResourceRequest to the protobuf version.
func (t ConvertToProtobuf) CallResourceRequest(req *CallResourceRequest) *pluginv2.CallResourceRequest {
	protoReq := &pluginv2.CallResourceRequest{
		PluginContext: t.PluginContext(req.PluginContext),
		Path:          req.Path,
		Method:        req.Method,
		Url:           req.URL,
		Body:          req.Body,
	}
	if req.Headers == nil {
		return protoReq
	}
	protoReq.Headers = make(map[string]*pluginv2.StringList, len(protoReq.Headers))
	for k, values := range req.Headers {
		protoReq.Headers[k] = &pluginv2.StringList{Values: values}
	}
	return protoReq
}

// RunStreamRequest ...
func (t ConvertToProtobuf) RunStreamRequest(req *RunStreamRequest) *pluginv2.RunStreamRequest {
	protoReq := &pluginv2.RunStreamRequest{
		PluginContext: t.PluginContext(req.PluginContext),
		Path:          req.Path,
		Data:          req.Data,
	}
	return protoReq
}

// SubscribeStreamRequest ...
func (t ConvertToProtobuf) SubscribeStreamRequest(req *SubscribeStreamRequest) *pluginv2.SubscribeStreamRequest {
	return &pluginv2.SubscribeStreamRequest{
		PluginContext: t.PluginContext(req.PluginContext),
		Path:          req.Path,
		Data:          req.Data,
	}
}

// SubscribeStreamResponse ...
func (t ConvertToProtobuf) SubscribeStreamResponse(req *SubscribeStreamResponse) *pluginv2.SubscribeStreamResponse {
	resp := &pluginv2.SubscribeStreamResponse{
		Status: pluginv2.SubscribeStreamResponse_Status(req.Status),
	}
	if req.InitialData != nil {
		resp.Data = req.InitialData.data
	}
	return resp
}

// PublishStreamRequest ...
func (t ConvertToProtobuf) PublishStreamRequest(req *PublishStreamRequest) *pluginv2.PublishStreamRequest {
	return &pluginv2.PublishStreamRequest{
		PluginContext: t.PluginContext(req.PluginContext),
		Path:          req.Path,
		Data:          req.Data,
	}
}

// PublishStreamResponse ...
func (t ConvertToProtobuf) PublishStreamResponse(req *PublishStreamResponse) *pluginv2.PublishStreamResponse {
	return &pluginv2.PublishStreamResponse{
		Status: pluginv2.PublishStreamResponse_Status(req.Status),
		Data:   req.Data,
	}
}

// StreamPacket ...
func (t ConvertToProtobuf) StreamPacket(p *StreamPacket) *pluginv2.StreamPacket {
	protoReq := &pluginv2.StreamPacket{
		Data: p.Data,
	}
	return protoReq
}
