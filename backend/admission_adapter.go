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
	ctx = setupContext(ctx, EndpointValidateAdmission)
	parsedReq := FromProto().AdmissionRequest(req)

	var resp *ValidationResponse
	err := wrapHandler(ctx, parsedReq.PluginContext, func(ctx context.Context) (RequestStatus, error) {
		var innerErr error
		resp, innerErr = a.handler.ValidateAdmission(ctx, parsedReq)
		return RequestStatusFromError(innerErr), innerErr
	})
	if err != nil {
		return nil, err
	}

	return ToProto().ValidationResponse(resp), nil
}

func (a *admissionSDKAdapter) MutateAdmission(ctx context.Context, req *pluginv2.AdmissionRequest) (*pluginv2.MutationResponse, error) {
	ctx = setupContext(ctx, EndpointMutateAdmission)
	parsedReq := FromProto().AdmissionRequest(req)

	var resp *MutationResponse
	err := wrapHandler(ctx, parsedReq.PluginContext, func(ctx context.Context) (RequestStatus, error) {
		var innerErr error
		resp, innerErr = a.handler.MutateAdmission(ctx, parsedReq)
		return RequestStatusFromError(innerErr), innerErr
	})
	if err != nil {
		return nil, err
	}

	return ToProto().MutationResponse(resp), nil
}

func (a *admissionSDKAdapter) ConvertObject(ctx context.Context, req *pluginv2.ConversionRequest) (*pluginv2.ConversionResponse, error) {
	ctx = setupContext(ctx, EndpointConvertObject)
	parsedReq := FromProto().ConversionRequest(req)

	var resp *ConversionResponse
	err := wrapHandler(ctx, parsedReq.PluginContext, func(ctx context.Context) (RequestStatus, error) {
		var innerErr error
		resp, innerErr = a.handler.ConvertObject(ctx, parsedReq)
		return RequestStatusFromError(innerErr), innerErr
	})
	if err != nil {
		return nil, err
	}

	return ToProto().ConversionResponse(resp), nil
}
