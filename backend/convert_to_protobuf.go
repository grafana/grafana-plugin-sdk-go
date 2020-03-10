package backend

import (
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/dataframe"
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

func (t convertToProtobuf) DataSourceConfig(config *DataSourceConfig) *pluginv2.DataSourceConfig {
	if config == nil {
		return nil
	}

	return &pluginv2.DataSourceConfig{
		Id:               config.ID,
		Name:             config.Name,
		Url:              config.URL,
		User:             config.User,
		Database:         config.Database,
		BasicAuthEnabled: config.BasicAuthEnabled,
		BasicAuthUser:    config.BasicAuthUser,
	}
}

func (t convertToProtobuf) PluginConfig(config PluginConfig) *pluginv2.PluginConfig {
	return &pluginv2.PluginConfig{
		OrgId:                   config.OrgID,
		PluginId:                config.PluginID,
		JsonData:                config.JSONData,
		DecryptedSecureJsonData: config.DecryptedSecureJSONData,
		LastUpdatedMS:           config.Updated.UnixNano() / int64(time.Millisecond),
		DatasourceConfig:        t.DataSourceConfig(config.DataSourceConfig),
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
		Config:  t.PluginConfig(req.PluginConfig),
		Headers: req.Headers,
		Queries: queries,
	}
}

func (t convertToProtobuf) QueryDataResponse(res *QueryDataResponse) (*pluginv2.QueryDataResponse, error) {
	encodedFrames := make([][]byte, len(res.Frames))
	var err error
	for i, frame := range res.Frames {
		encodedFrames[i], err = dataframe.MarshalArrow(frame)
		if err != nil {
			return nil, err
		}
	}

	return &pluginv2.QueryDataResponse{
		Frames:   encodedFrames,
		Metadata: res.Metadata,
	}, nil
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
