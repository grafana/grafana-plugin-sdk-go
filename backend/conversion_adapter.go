package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

type conversionSDKAdapter struct {
	handler ConversionHandler
}

func newConversionSDKAdapter(handler ConversionHandler) *conversionSDKAdapter {
	return &conversionSDKAdapter{
		handler: handler,
	}
}

func (a *conversionSDKAdapter) ConvertObjects(ctx context.Context, req *pluginv2.ConversionRequest) (*pluginv2.ConversionResponse, error) {
	ctx = setupContext(ctx, EndpointConvertObjects)
	parsedReq := FromProto().ConversionRequest(req)

	var resp *ConversionResponse
	err := wrapHandler(ctx, parsedReq.PluginContext, func(ctx context.Context) (RequestStatus, error) {
		var innerErr error
		resp, innerErr = a.handler.ConvertObjects(ctx, parsedReq)
		return RequestStatusFromError(innerErr), innerErr
	})
	if err != nil {
		return nil, err
	}

	return ToProto().ConversionResponse(resp), nil
}
