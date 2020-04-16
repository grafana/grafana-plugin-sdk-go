package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// dataSDKAdapter adapter between low level plugin protocol and SDK interfaces.
type dataSDKAdapter struct {
	queryDataHandler QueryDataHandler
}

func newDataSDKAdapter(handler QueryDataHandler) *dataSDKAdapter {
	return &dataSDKAdapter{
		queryDataHandler: handler,
	}
}

func (a *dataSDKAdapter) QueryData(ctx context.Context, req *pluginv2.QueryDataRequest) (*pluginv2.QueryDataResponse, error) {
	pCtx := fromProto().PluginContext(ctx, req.Context)
	resp, err := a.queryDataHandler.QueryData(pCtx, fromProto().QueryDataRequest(req))
	if err != nil {
		return nil, err
	}

	return toProto().QueryDataResponse(resp)
}
