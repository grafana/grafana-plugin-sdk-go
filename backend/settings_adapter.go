package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// settingsSDKAdapter adapter between low level plugin protocol and SDK interfaces.
type settingsSDKAdapter struct {
	handler InstanceSettingsHandler
}

func newInstanceSettingsSDKAdapter(handler InstanceSettingsHandler) *settingsSDKAdapter {
	return &settingsSDKAdapter{
		handler: handler,
	}
}

func (a *settingsSDKAdapter) CreateInstanceSettings(ctx context.Context, req *pluginv2.CreateInstanceSettingsRequest) (*pluginv2.InstanceSettingsResponse, error) {
	ctx = propagateTenantIDIfPresent(ctx)
	parsedReq := FromProto().CreateInstanceSettingsRequest(req)
	resp, err := a.handler.CreateInstanceSettings(ctx, parsedReq)
	if err != nil {
		return nil, err
	}
	return ToProto().InstanceSettingsResponse(resp), nil
}

func (a *settingsSDKAdapter) UpdateInstanceSettings(ctx context.Context, req *pluginv2.UpdateInstanceSettingsRequest) (*pluginv2.InstanceSettingsResponse, error) {
	ctx = propagateTenantIDIfPresent(ctx)
	parsedReq := FromProto().UpdateInstanceSettingsRequest(req)
	ctx = WithUserAgent(ctx, parsedReq.PluginContext.UserAgent)
	resp, err := a.handler.UpdateInstanceSettings(ctx, parsedReq)
	if err != nil {
		return nil, err
	}
	return ToProto().InstanceSettingsResponse(resp), nil
}
