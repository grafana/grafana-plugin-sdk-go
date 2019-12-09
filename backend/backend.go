package backend

import (
	"context"
	"encoding/json"
	"fmt"

	bproto "github.com/grafana/grafana-plugin-sdk-go/genproto/go/grafana_plugin"
	plugin "github.com/hashicorp/go-plugin"
)

// PluginConfig holds configuration for the queried data source.
type PluginConfig struct {
	ID       int64
	OrgID    int64
	Name     string
	Type     string
	URL      string
	JSONData json.RawMessage
}

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

type FetchInfo int

const (
	FetchInfoStatus FetchInfo = iota
	FetchInfoAPI
	FetchInfoMetrics
	FetchInfoDebug
)

func (fi FetchInfo) toProtobuf() bproto.PluginStatusRequest_FetchInfo {
	switch fi {
	case FetchInfoStatus:
		return bproto.PluginStatusRequest_STATUS
	case FetchInfoAPI:
		return bproto.PluginStatusRequest_API
	case FetchInfoMetrics:
		return bproto.PluginStatusRequest_METRICS
	case FetchInfoDebug:
		return bproto.PluginStatusRequest_DEBUG

	}
	panic("unsupported protobuf FetchInfo type in sdk")
}

func FetchInfoFromProtobuf(ptype bproto.PluginStatusRequest_FetchInfo) (FetchInfo, error) {
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

type PluginStatus int

const (
	PluginStatusUnknown PluginStatus = iota
	PluginStatusOk
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

// backendPluginWrapper converts to and from protobuf types.
type backendPluginWrapper struct {
	plugin.NetRPCUnsupportedPlugin

	dataHandler  DataQueryHandler
	checkHandler CheckHandler
}

// CheckHandler handles backend plugin checks.
type CheckHandler interface {
	Check(ctx context.Context, pc PluginConfig, headers map[string]string, fetch FetchInfo) (CheckResponse, error)
}

func (p *backendPluginWrapper) Check(ctx context.Context, req *bproto.PluginStatusRequest) (*bproto.PluginStatusResponse, error) {
	fetchType, err := FetchInfoFromProtobuf(req.Fetch)
	if err != nil {
		return nil, err
	}
	pc := pluginConfigFromProto(req.Config)
	resp, err := p.checkHandler.Check(ctx, pc, req.Headers, fetchType)
	if err != nil {
		return nil, err
	}
	pbRes := resp.toProtobuf()
	return &pbRes, nil
}
