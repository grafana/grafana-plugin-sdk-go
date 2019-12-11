package backend

import (
	"context"
	"encoding/json"
	"fmt"

	bproto "github.com/grafana/grafana-plugin-sdk-go/genproto/go/backend_plugin"
	plugin "github.com/hashicorp/go-plugin"
)

// PluginConfig holds configuration for the queried plugin.
type PluginConfig struct {
	ID       int64
	OrgID    int64
	Name     string
	Type     string
	URL      string
	JSONData json.RawMessage
}

// PluginConfigFromProto converts the generated protobuf PluginConfig to this
// package's PluginConfig.
func pluginConfigFromProto(pc *bproto.PluginConfig) PluginConfig {
	return PluginConfig{
		ID:       pc.Id,
		OrgID:    pc.OrgId,
		Name:     pc.Name,
		Type:     pc.Type,
		URL:      pc.Url,
		JSONData: json.RawMessage(pc.JsonData),
	}
}

// FetchInfo is type information requested from the Check endpoint.
type FetchInfo int

const (
	// FetchInfoStatus is a request for plugin's status only (no info).
	FetchInfoStatus FetchInfo = iota
	// FetchInfoAPI is a request for an OpenAPI description of the plugin.
	FetchInfoAPI
	// FetchInfoMetrics is a request for Promotheus style metrics on the endpoint.
	FetchInfoMetrics
	// FetchInfoDebug is a request for JSON debug info (admin+ view).
	FetchInfoDebug
)

// fetchInfoFromProtobuf converts the generated protobuf PluginStatusRequest_FetchInfo to this
// package's FetchInfo.
func fetchInfoFromProtobuf(ptype bproto.PluginStatusRequest_FetchInfo) (FetchInfo, error) {
	switch ptype {
	case bproto.PluginStatusRequest_STATUS:
		return FetchInfoStatus, nil
	case bproto.PluginStatusRequest_API:
		return FetchInfoAPI, nil
	case bproto.PluginStatusRequest_METRICS:
		return FetchInfoMetrics, nil
	case bproto.PluginStatusRequest_DEBUG:
		return FetchInfoDebug, nil

	}
	return FetchInfoStatus, fmt.Errorf("unsupported protobuf FetchInfo type in sdk: %v", ptype)
}

// PluginStatus is the status of the Plugin and should be returned
// with any Check request.
type PluginStatus int

const (
	// PluginStatusUnknown means the status of the plugin is unknown.
	PluginStatusUnknown PluginStatus = iota
	// PluginStatusOk means the status of the plugin is good.
	PluginStatusOk
	// PluginStatusError means the plugin is in an error state.
	PluginStatusError
)

func (ps PluginStatus) toProtobuf() bproto.PluginStatusResponse_PluginStatus {
	switch ps {
	case PluginStatusUnknown:
		return bproto.PluginStatusResponse_UNKNOWN
	case PluginStatusOk:
		return bproto.PluginStatusResponse_OK
	case PluginStatusError:
		return bproto.PluginStatusResponse_ERROR
	}
	panic("unsupported protobuf FetchInfo type in sdk")
}

// CheckResponse is the return type from a Check Request.
type CheckResponse struct {
	Status PluginStatus
	Info   string
}

func (cr CheckResponse) toProtobuf() bproto.PluginStatusResponse {
	return bproto.PluginStatusResponse{
		Status: cr.Status.toProtobuf(),
		Info:   cr.Info,
	}
}

// coreWrapper converts to and from protobuf types.
type coreWrapper struct {
	plugin.NetRPCUnsupportedPlugin

	handlers PluginHandlers
}

// PluginHandlers is the collection of handlers that corresponds to the
// grpc "service BackendPlugin".
type PluginHandlers struct {
	DataQueryHandler
	CheckHandler
	ResourceHandler
}

// CheckHandler handles backend plugin checks.
type CheckHandler interface {
	Check(ctx context.Context, pc PluginConfig, headers map[string]string, fetch FetchInfo) (CheckResponse, error)
}

func (p *coreWrapper) Check(ctx context.Context, req *bproto.PluginStatusRequest) (*bproto.PluginStatusResponse, error) {
	fetchType, err := fetchInfoFromProtobuf(req.Fetch)
	if err != nil {
		return nil, err
	}
	pc := pluginConfigFromProto(req.Config)
	resp, err := p.handlers.Check(ctx, pc, req.Headers, fetchType)
	if err != nil {
		return nil, err
	}
	pbRes := resp.toProtobuf()
	return &pbRes, nil
}
