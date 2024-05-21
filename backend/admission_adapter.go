package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// admissionSDKAdapter adapter between low level plugin protocol and SDK interfaces.
type admissionSDKAdapter struct {
	handler AdmissionHandler
}

func newAdmissionSDKAdapter(handler AdmissionHandler) *admissionSDKAdapter {
	return &admissionSDKAdapter{
		handler: handler,
	}
}

func (a *admissionSDKAdapter) ValidateAdmission(ctx context.Context, req *pluginv2.AdmissionRequest) (*pluginv2.AdmissionResponse, error) {
	ctx = propagateTenantIDIfPresent(ctx)
	parsedReq := FromProto().AdmissionRequest(req)
	resp, err := a.handler.ValidateAdmission(ctx, parsedReq)
	if err != nil {
		return nil, err
	}
	return ToProto().AdmissionResponse(resp), nil
}

func (a *admissionSDKAdapter) MutateAdmission(ctx context.Context, req *pluginv2.AdmissionRequest) (*pluginv2.AdmissionResponse, error) {
	ctx = propagateTenantIDIfPresent(ctx)
	parsedReq := FromProto().AdmissionRequest(req)
	resp, err := a.handler.MutateAdmission(ctx, parsedReq)
	if err != nil {
		return nil, err
	}
	return ToProto().AdmissionResponse(resp), nil
}

func (a *admissionSDKAdapter) ConvertObject(ctx context.Context, req *pluginv2.ConversionRequest) (*pluginv2.AdmissionResponse, error) {
	ctx = propagateTenantIDIfPresent(ctx)
	parsedReq := FromProto().ConversionRequest(req)
	resp, err := a.handler.ConvertObject(ctx, parsedReq)
	if err != nil {
		return nil, err
	}
	return ToProto().AdmissionResponse(resp), nil
}
