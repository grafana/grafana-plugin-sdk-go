package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// settingsSDKAdapter adapter between low level plugin protocol and SDK interfaces.
type settingsSDKAdapter struct {
	admissionHandler InstanceSettingsHandler
}

func newInstanceSettingsSDKAdapter(handler InstanceSettingsHandler) *settingsSDKAdapter {
	return &settingsSDKAdapter{
		admissionHandler: handler,
	}
}

func (a *settingsSDKAdapter) ProcessInstanceSettings(ctx context.Context, req *pluginv2.ProcessInstanceSettingsRequest) (*pluginv2.ProcessInstanceSettingsResponse, error) {
	ctx = propagateTenantIDIfPresent(ctx)
	ctx = WithGrafanaConfig(ctx, NewGrafanaCfg(req.PluginContext.GrafanaConfig))
	parsedReq := FromProto().ProcessInstanceSettingsRequest(req)
	ctx = withContextualLogAttributes(ctx, parsedReq.PluginContext, endpointQueryData)
	ctx = WithUserAgent(ctx, parsedReq.PluginContext.UserAgent)
	resp, err := a.admissionHandler.ProcessInstanceSettings(ctx, parsedReq)
	if err != nil {
		return nil, err
	}
	return ToProto().ProcessInstanceSettingsResponse(resp), nil
}
