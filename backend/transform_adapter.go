package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend/grpcplugin"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// transformSDKAdapter adapter between low level plugin protocol and SDK interfaces.
type transformSDKAdapter struct {
	transformDataHandler TransformDataHandler
}

func newTransformSDKAdapter(handler TransformDataHandler) *transformSDKAdapter {
	return &transformSDKAdapter{
		transformDataHandler: handler,
	}
}

func (a *transformSDKAdapter) TransformData(ctx context.Context, req *pluginv2.QueryDataRequest, callBack grpcplugin.TransformDataCallBack) (*pluginv2.QueryDataResponse, error) {
	pCtx := fromProto().PluginContext(ctx, req.Context)
	resp, err := a.transformDataHandler.TransformData(pCtx, fromProto().QueryDataRequest(req), &transformDataCallBackWrapper{callBack})
	if err != nil {
		return nil, err
	}

	return toProto().QueryDataResponse(resp)
}

type transformDataCallBackWrapper struct {
	callBack grpcplugin.TransformDataCallBack
}

func (tw *transformDataCallBackWrapper) QueryData(pCtx PluginContext, req *QueryDataRequest) (*QueryDataResponse, error) {
	protoRes, err := tw.callBack.QueryData(pCtx.RequestContext, toProto().QueryDataRequest(pCtx, req))
	if err != nil {
		return nil, err
	}

	return fromProto().QueryDataResponse(protoRes)
}
