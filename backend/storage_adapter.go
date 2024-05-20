package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// storageSDKAdapter adapter between low level plugin protocol and SDK interfaces.
type storageSDKAdapter struct {
	handler StorageHandler
}

func newStorageSDKAdapter(handler StorageHandler) *storageSDKAdapter {
	return &storageSDKAdapter{
		handler: handler,
	}
}

func (a *storageSDKAdapter) MutateInstanceSettings(ctx context.Context, req *pluginv2.InstanceSettingsAdmissionRequest) (*pluginv2.InstanceSettingsResponse, error) {
	ctx = propagateTenantIDIfPresent(ctx)
	parsedReq := FromProto().InstanceSettingsAdmissionRequest(req)
	resp, err := a.handler.MutateInstanceSettings(ctx, parsedReq)
	if err != nil {
		return nil, err
	}
	return ToProto().InstanceSettingsResponse(resp), nil
}

func (a *storageSDKAdapter) ValidateAdmission(ctx context.Context, req *pluginv2.AdmissionRequest) (*pluginv2.StorageResponse, error) {
	ctx = propagateTenantIDIfPresent(ctx)
	parsedReq := FromProto().AdmissionRequest(req)
	resp, err := a.handler.ValidateAdmission(ctx, parsedReq)
	if err != nil {
		return nil, err
	}
	return ToProto().StorageResponse(resp), nil
}

func (a *storageSDKAdapter) ConvertObject(ctx context.Context, req *pluginv2.ConversionRequest) (*pluginv2.StorageResponse, error) {
	ctx = propagateTenantIDIfPresent(ctx)
	parsedReq := FromProto().ConversionRequest(req)
	resp, err := a.handler.ConvertObject(ctx, parsedReq)
	if err != nil {
		return nil, err
	}
	return ToProto().StorageResponse(resp), nil
}

func (a *storageSDKAdapter) MutateAdmission(ctx context.Context, req *pluginv2.AdmissionRequest) (*pluginv2.StorageResponse, error) {
	ctx = propagateTenantIDIfPresent(ctx)
	parsedReq := FromProto().AdmissionRequest(req)
	resp, err := a.handler.MutateAdmission(ctx, parsedReq)
	if err != nil {
		return nil, err
	}
	return ToProto().StorageResponse(resp), nil
}
