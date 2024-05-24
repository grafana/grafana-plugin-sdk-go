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

func (a *admissionSDKAdapter) ValidateAdmission(ctx context.Context, req *pluginv2.AdmissionRequest) (*pluginv2.ValidationResponse, error) {
	ctx = propagateTenantIDIfPresent(ctx)
	parsedReq := FromProto().AdmissionRequest(req)
	resp, err := a.handler.ValidateAdmission(ctx, parsedReq)
	if err != nil {
		return nil, err
	}
	return ToProto().ValidationResponse(resp), nil
}

func (a *admissionSDKAdapter) MutateAdmission(ctx context.Context, req *pluginv2.AdmissionRequest) (*pluginv2.MutationResponse, error) {
	ctx = propagateTenantIDIfPresent(ctx)
	parsedReq := FromProto().AdmissionRequest(req)
	resp, err := a.handler.MutateAdmission(ctx, parsedReq)
	if err != nil {
		return nil, err
	}
	return ToProto().MutationResponse(resp), nil
}

func (a *admissionSDKAdapter) ConvertObject(ctx context.Context, req *pluginv2.ConversionRequest) (*pluginv2.ConversionResponse, error) {
	ctx = propagateTenantIDIfPresent(ctx)
	parsedReq := FromProto().ConversionRequest(req)
	resp, err := a.handler.ConvertObject(ctx, parsedReq)
	if err != nil {
		return nil, err
	}
	return ToProto().ConversionResponse(resp), nil
}
