package backend

import (
	"errors"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// ConvertFromProtobuf has a collection of methods for converting from the autogenerated
// protobuf go code to our SDK objects. This object exists to attach a collection
// of conversion methods to.
//
// This is used internally by the SDK and inside Grafana-server, plugin authors should not
// need this functionality.
type ConvertFromProtobuf struct{}

// FromProto returns a new ConvertFromProtobuf.
func FromProto() ConvertFromProtobuf {
	return ConvertFromProtobuf{}
}

// User converts protobuf version of a User to the SDK version.
func (f ConvertFromProtobuf) User(user *pluginv2.User) *User {
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

// AppInstanceSettings converts protobuf version of an AppInstanceSettings to the SDK version.
func (f ConvertFromProtobuf) AppInstanceSettings(proto *pluginv2.AppInstanceSettings) *AppInstanceSettings {
	if proto == nil {
		return nil
	}

	return &AppInstanceSettings{
		JSONData:                proto.JsonData,
		DecryptedSecureJSONData: proto.DecryptedSecureJsonData,
		Updated:                 time.Unix(0, proto.LastUpdatedMS*int64(time.Millisecond)),
	}
}

// DataSourceInstanceSettings converts protobuf version of a DataSourceInstanceSettings to the SDK version.
func (f ConvertFromProtobuf) DataSourceInstanceSettings(proto *pluginv2.DataSourceInstanceSettings, pluginID string) *DataSourceInstanceSettings {
	if proto == nil {
		return nil
	}

	return &DataSourceInstanceSettings{
		ID:                      proto.Id,
		UID:                     proto.Uid,
		Type:                    pluginID,
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

// PluginContext converts protobuf version of a PluginContext to the SDK version.
func (f ConvertFromProtobuf) PluginContext(proto *pluginv2.PluginContext) PluginContext {
	return PluginContext{
		OrgID:                      proto.OrgId,
		PluginID:                   proto.PluginId,
		User:                       f.User(proto.User),
		AppInstanceSettings:        f.AppInstanceSettings(proto.AppInstanceSettings),
		DataSourceInstanceSettings: f.DataSourceInstanceSettings(proto.DataSourceInstanceSettings, proto.PluginId),
	}
}

// TimeRange converts protobuf version of a TimeRange to the SDK version.
func (f ConvertFromProtobuf) TimeRange(proto *pluginv2.TimeRange) TimeRange {
	return TimeRange{
		From: time.Unix(0, proto.FromEpochMS*int64(time.Millisecond)),
		To:   time.Unix(0, proto.ToEpochMS*int64(time.Millisecond)),
	}
}

// DataQuery converts protobuf version of a DataQuery to the SDK version.
func (f ConvertFromProtobuf) DataQuery(proto *pluginv2.DataQuery) *DataQuery {
	if proto == nil {
		return nil
	}
	return &DataQuery{
		RefID:         proto.RefId,
		QueryType:     proto.QueryType,
		MaxDataPoints: proto.MaxDataPoints,
		TimeRange:     f.TimeRange(proto.TimeRange),
		Interval:      time.Duration(proto.IntervalMS) * time.Millisecond,
		JSON:          proto.Json,
	}
}

// QueryDataRequest converts protobuf version of a QueryDataRequest to the SDK version.
func (f ConvertFromProtobuf) QueryDataRequest(protoReq *pluginv2.QueryDataRequest) *QueryDataRequest {
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

// QueryDataResponse converts protobuf version of a QueryDataResponse to the SDK version.
func (f ConvertFromProtobuf) QueryDataResponse(protoRes *pluginv2.QueryDataResponse) (*QueryDataResponse, error) {
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
		}
		if res.Error != "" {
			dr.Error = errors.New(res.Error)
		}
		if res.ErrorDetails != nil {
			dr.ErrorDetails = &ErrorDetails{
				Status: ErrorStatus(res.ErrorDetails.Status),
			}
		}
		qdr.Responses[refID] = dr
	}
	return qdr, nil
}

// CallResourceRequest converts protobuf version of a CallResourceRequest to the SDK version.
func (f ConvertFromProtobuf) CallResourceRequest(protoReq *pluginv2.CallResourceRequest) *CallResourceRequest {
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

// CallResourceResponse converts protobuf version of a CallResourceResponse to the SDK version.
func (f ConvertFromProtobuf) CallResourceResponse(protoResp *pluginv2.CallResourceResponse) *CallResourceResponse {
	headers := map[string][]string{}
	for k, values := range protoResp.Headers {
		headers[k] = values.Values
	}

	return &CallResourceResponse{
		Status:  int(protoResp.Code),
		Body:    protoResp.Body,
		Headers: headers,
	}
}

// CheckHealthRequest converts protobuf version of a CheckHealthRequest to the SDK version.
func (f ConvertFromProtobuf) CheckHealthRequest(protoReq *pluginv2.CheckHealthRequest) *CheckHealthRequest {
	if protoReq.Headers == nil {
		protoReq.Headers = map[string]string{}
	}

	return &CheckHealthRequest{
		PluginContext: f.PluginContext(protoReq.PluginContext),
		Headers:       protoReq.Headers,
	}
}

// CheckHealthResponse converts protobuf version of a HealthCheckResponse to the SDK version.
func (f ConvertFromProtobuf) CheckHealthResponse(protoResp *pluginv2.CheckHealthResponse) *CheckHealthResult {
	status := HealthStatusUnknown
	switch protoResp.Status {
	case pluginv2.CheckHealthResponse_ERROR:
		status = HealthStatusError
	case pluginv2.CheckHealthResponse_OK:
		status = HealthStatusOk
	}

	return &CheckHealthResult{
		Status:      status,
		Message:     protoResp.Message,
		JSONDetails: protoResp.JsonDetails,
	}
}

// CollectMetricsRequest converts protobuf version of a CollectMetricsRequest to the SDK version.
func (f ConvertFromProtobuf) CollectMetricsRequest(protoReq *pluginv2.CollectMetricsRequest) *CollectMetricsRequest {
	return &CollectMetricsRequest{
		PluginContext: f.PluginContext(protoReq.PluginContext),
	}
}

// CollectMetricsResponse converts protobuf version of a CollectMetricsResponse to the SDK version.
func (f ConvertFromProtobuf) CollectMetricsResponse(protoResp *pluginv2.CollectMetricsResponse) *CollectMetricsResult {
	var prometheusMetrics []byte

	if protoResp.Metrics != nil {
		prometheusMetrics = protoResp.Metrics.Prometheus
	}

	return &CollectMetricsResult{
		PrometheusMetrics: prometheusMetrics,
	}
}

// SubscribeStreamRequest ...
func (f ConvertFromProtobuf) SubscribeStreamRequest(protoReq *pluginv2.SubscribeStreamRequest) *SubscribeStreamRequest {
	return &SubscribeStreamRequest{
		PluginContext: f.PluginContext(protoReq.PluginContext),
		Path:          protoReq.GetPath(),
		Data:          protoReq.GetData(),
	}
}

// SubscribeStreamResponse ...
func (f ConvertFromProtobuf) SubscribeStreamResponse(protoReq *pluginv2.SubscribeStreamResponse) *SubscribeStreamResponse {
	return &SubscribeStreamResponse{
		Status: SubscribeStreamStatus(protoReq.GetStatus()),
		InitialData: &InitialData{
			data: protoReq.Data,
		},
	}
}

// PublishStreamRequest ...
func (f ConvertFromProtobuf) PublishStreamRequest(protoReq *pluginv2.PublishStreamRequest) *PublishStreamRequest {
	return &PublishStreamRequest{
		PluginContext: f.PluginContext(protoReq.PluginContext),
		Path:          protoReq.GetPath(),
	}
}

// PublishStreamResponse ...
func (f ConvertFromProtobuf) PublishStreamResponse(protoReq *pluginv2.PublishStreamResponse) *PublishStreamResponse {
	return &PublishStreamResponse{
		Status: PublishStreamStatus(protoReq.GetStatus()),
		Data:   protoReq.GetData(),
	}
}

// RunStreamRequest ...
func (f ConvertFromProtobuf) RunStreamRequest(protoReq *pluginv2.RunStreamRequest) *RunStreamRequest {
	return &RunStreamRequest{
		PluginContext: f.PluginContext(protoReq.PluginContext),
		Path:          protoReq.GetPath(),
		Data:          protoReq.GetData(),
	}
}

// StreamPacket ...
func (f ConvertFromProtobuf) StreamPacket(protoReq *pluginv2.StreamPacket) *StreamPacket {
	return &StreamPacket{
		Data: protoReq.GetData(),
	}
}
