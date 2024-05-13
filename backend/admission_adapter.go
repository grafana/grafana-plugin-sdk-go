package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// admissionSDKAdapter adapter between low level plugin protocol and SDK interfaces.
type admissionSDKAdapter struct {
	admissionHandler AdmissionHandler
}

func newAdmissionSDKAdapter(handler AdmissionHandler) *admissionSDKAdapter {
	return &admissionSDKAdapter{
		admissionHandler: handler,
	}
}

func (a *admissionSDKAdapter) ProcessInstanceSettings(ctx context.Context, req *pluginv2.ProcessInstanceSettingsRequest) (*pluginv2.ProcessInstanceSettingsResponse, error) {
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

func (a *admissionSDKAdapter) ValidateAdmission(ctx context.Context, req *pluginv2.AdmissionRequest) (*pluginv2.AdmissionResponse, error) {
	ctx = propagateTenantIDIfPresent(ctx)
	ctx = WithGrafanaConfig(ctx, NewGrafanaCfg(req.PluginContext.GrafanaConfig))
	parsedReq := FromProto().AdmissionRequest(req)
	ctx = withContextualLogAttributes(ctx, parsedReq.PluginContext, endpointQueryData)
	ctx = WithUserAgent(ctx, parsedReq.PluginContext.UserAgent)
	resp, err := a.admissionHandler.ValidateAdmission(ctx, parsedReq)
	if err != nil {
		return nil, err
	}
	return ToProto().AdmissionResponse(resp), nil
}

func (a *admissionSDKAdapter) MutateAdmission(ctx context.Context, req *pluginv2.AdmissionRequest) (*pluginv2.AdmissionResponse, error) {
	ctx = propagateTenantIDIfPresent(ctx)
	ctx = WithGrafanaConfig(ctx, NewGrafanaCfg(req.PluginContext.GrafanaConfig))
	parsedReq := FromProto().AdmissionRequest(req)
	ctx = withContextualLogAttributes(ctx, parsedReq.PluginContext, endpointQueryData)
	ctx = WithUserAgent(ctx, parsedReq.PluginContext.UserAgent)
	resp, err := a.admissionHandler.MutateAdmission(ctx, parsedReq)
	if err != nil {
		return nil, err
	}
	return ToProto().AdmissionResponse(resp), nil
}
