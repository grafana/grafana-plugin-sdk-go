package backend

import (
	"errors"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/useragent"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	"google.golang.org/protobuf/proto"
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
		Login:   user.Login,
		Name:    user.Name,
		Email:   user.Email,
		Role:    user.Role,
		IdToken: user.IDToken,
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
		ApiVersion:              s.APIVersion,
	}
}

func AppInstanceSettingsToProtoBytes(s *AppInstanceSettings) ([]byte, error) {
	if s == nil {
		return nil, nil
	}
	return proto.Marshal(ConvertToProtobuf{}.AppInstanceSettings(s))
}

func AppInstanceSettingsFromProto(body []byte) (*AppInstanceSettings, error) {
	if len(body) == 0 {
		return nil, nil
	}
	tmp := &pluginv2.AppInstanceSettings{}
	err := proto.Unmarshal(body, tmp)
	if err != nil {
		return nil, err
	}
	return ConvertFromProtobuf{}.AppInstanceSettings(tmp), nil
}

func DataSourceInstanceSettingsToProtoBytes(s *DataSourceInstanceSettings) ([]byte, error) {
	if s == nil {
		return nil, nil
	}
	return proto.Marshal(ConvertToProtobuf{}.DataSourceInstanceSettings(s))
}

func DataSourceInstanceSettingsFromProto(body []byte, pluginID string) (*DataSourceInstanceSettings, error) {
	if len(body) == 0 {
		return nil, nil
	}
	tmp := &pluginv2.DataSourceInstanceSettings{}
	err := proto.Unmarshal(body, tmp)
	if err != nil {
		return nil, err
	}
	return ConvertFromProtobuf{}.DataSourceInstanceSettings(tmp, pluginID), nil
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
		ApiVersion:              s.APIVersion,
	}
}

// UserAgent converts the SDK version of a useragent.UserAgent to the protobuf version.
func (t ConvertToProtobuf) UserAgent(ua *useragent.UserAgent) string {
	if ua == nil {
		return ""
	}

	return ua.String()
}

// PluginContext converts the SDK version of a PluginContext to the protobuf version.
func (t ConvertToProtobuf) PluginContext(pluginCtx PluginContext) *pluginv2.PluginContext {
	return &pluginv2.PluginContext{
		OrgId:                      pluginCtx.OrgID,
		PluginId:                   pluginCtx.PluginID,
		PluginVersion:              pluginCtx.PluginVersion,
		ApiVersion:                 pluginCtx.APIVersion,
		User:                       t.User(pluginCtx.User),
		AppInstanceSettings:        t.AppInstanceSettings(pluginCtx.AppInstanceSettings),
		DataSourceInstanceSettings: t.DataSourceInstanceSettings(pluginCtx.DataSourceInstanceSettings),
		GrafanaConfig:              t.GrafanaConfig(pluginCtx.GrafanaConfig),
		UserAgent:                  t.UserAgent(pluginCtx.UserAgent),
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
			if f == nil {
				return nil, errors.New("frame can not be nil")
			}
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
		status := dr.Status
		if dr.Error != nil {
			pDR.Error = dr.Error.Error()
			if !status.IsValid() {
				status = statusFromError(dr.Error)
			}
		}
		if status.IsValid() {
			pDR.Status = int32(status)
		} else if status == 0 {
			pDR.Status = int32(StatusOK)
		}
		pDR.ErrorSource = string(dr.ErrorSource)

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

// CollectMetricsRequest converts the SDK version of a CollectMetricsRequest to the protobuf version.
func (t ConvertToProtobuf) CollectMetricsRequest(req *CollectMetricsRequest) *pluginv2.CollectMetricsRequest {
	return &pluginv2.CollectMetricsRequest{
		PluginContext: t.PluginContext(req.PluginContext),
	}
}

// CollectMetricsResult converts the SDK version of a CollectMetricsResult to the protobuf version.
func (t ConvertToProtobuf) CollectMetricsResult(res *CollectMetricsResult) *pluginv2.CollectMetricsResponse {
	return &pluginv2.CollectMetricsResponse{
		Metrics: &pluginv2.CollectMetricsResponse_Payload{
			Prometheus: res.PrometheusMetrics,
		},
	}
}

// StatusResult converts the SDK version of a StatusResult to the protobuf version.
func (t ConvertToProtobuf) StatusResult(s *StatusResult) *pluginv2.StatusResult {
	if s == nil {
		return nil
	}
	return &pluginv2.StatusResult{
		Status:  s.Status,
		Message: s.Message,
		Reason:  s.Reason,
		Code:    s.Code,
	}
}

// GroupVersionKind converts the SDK version of a GroupVersionKind to the protobuf version.
func (t ConvertToProtobuf) GroupVersionKind(req *GroupVersionKind) *pluginv2.GroupVersionKind {
	return &pluginv2.GroupVersionKind{
		Group:   req.Group,
		Version: req.Version,
		Kind:    req.Kind,
	}
}

// GroupVersion converts the SDK version of a GroupVersion to the protobuf version.
func (t ConvertToProtobuf) GroupVersion(req *GroupVersion) *pluginv2.GroupVersion {
	return &pluginv2.GroupVersion{
		Group:   req.Group,
		Version: req.Version,
	}
}

// RawObject converts the SDK version of a RawObject to the protobuf version.
func (t ConvertToProtobuf) RawObject(req RawObject) *pluginv2.RawObject {
	return &pluginv2.RawObject{
		Raw:         req.Raw,
		ContentType: req.ContentType,
	}
}

// RawObjects converts the SDK version of a RawObject to the protobuf version.
func (t ConvertToProtobuf) RawObjects(req []RawObject) []*pluginv2.RawObject {
	objects := make([]*pluginv2.RawObject, len(req))
	for i, o := range req {
		objects[i] = t.RawObject(o)
	}
	return objects
}

// AdmissionRequest converts the SDK version of a AdmissionRequest to the protobuf version.
func (t ConvertToProtobuf) AdmissionRequest(req *AdmissionRequest) *pluginv2.AdmissionRequest {
	return &pluginv2.AdmissionRequest{
		PluginContext:  t.PluginContext(req.PluginContext),
		Operation:      pluginv2.AdmissionRequest_Operation(req.Operation),
		Kind:           t.GroupVersionKind(&req.Kind),
		ObjectBytes:    req.ObjectBytes,
		OldObjectBytes: req.OldObjectBytes,
	}
}

// ConversionRequest converts the SDK version of a ConversionRequest to the protobuf version.
func (t ConvertToProtobuf) ConversionRequest(req *ConversionRequest) *pluginv2.ConversionRequest {
	return &pluginv2.ConversionRequest{
		PluginContext: t.PluginContext(req.PluginContext),
		Uid:           req.UID,
		TargetVersion: t.GroupVersion(&req.TargetVersion),
		Objects:       t.RawObjects(req.Objects),
	}
}

// MutationResponse converts the SDK version of a MutationResponse to the protobuf version.
func (t ConvertToProtobuf) MutationResponse(rsp *MutationResponse) *pluginv2.MutationResponse {
	return &pluginv2.MutationResponse{
		Allowed:     rsp.Allowed,
		Result:      t.StatusResult(rsp.Result),
		Warnings:    rsp.Warnings,
		ObjectBytes: rsp.ObjectBytes,
	}
}

// ValidationResponse converts the SDK version of a ValidationResponse to the protobuf version.
func (t ConvertToProtobuf) ValidationResponse(rsp *ValidationResponse) *pluginv2.ValidationResponse {
	return &pluginv2.ValidationResponse{
		Allowed:  rsp.Allowed,
		Result:   t.StatusResult(rsp.Result),
		Warnings: rsp.Warnings,
	}
}

// ConversionResponse converts the SDK version of a ConversionResponse to the protobuf version.
func (t ConvertToProtobuf) ConversionResponse(rsp *ConversionResponse) *pluginv2.ConversionResponse {
	return &pluginv2.ConversionResponse{
		Uid:     rsp.UID,
		Result:  t.StatusResult(rsp.Result),
		Objects: t.RawObjects(rsp.Objects),
	}
}

// GrafanaConfig converts the SDK version of a GrafanaCfg to the protobuf version.
func (t ConvertToProtobuf) GrafanaConfig(cfg *GrafanaCfg) map[string]string {
	if cfg == nil {
		return map[string]string{}
	}
	return cfg.config
}
