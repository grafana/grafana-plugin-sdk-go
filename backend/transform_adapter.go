package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend/plugin"
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

func (a *transformSDKAdapter) TransformData(ctx context.Context, req *pluginv2.QueryDataRequest, callBack plugin.TransformDataCallBack) (*pluginv2.QueryDataResponse, error) {
	resp, err := a.transformDataHandler.TransformData(ctx, fromProto().QueryDataRequest(req), &transformDataCallBackWrapper{callBack})
	if err != nil {
		return nil, err
	}

	return toProto().QueryDataResponse(resp)
}

type transformDataCallBackWrapper struct {
	callBack plugin.TransformDataCallBack
}

func (tw *transformDataCallBackWrapper) QueryData(ctx context.Context, req *QueryDataRequest) (*QueryDataResponse, error) {
	protoRes, err := tw.callBack.QueryData(ctx, toProto().QueryDataRequest(req))
	if err != nil {
		return nil, err
	}

	return fromProto().QueryDataResponse(protoRes)
}
