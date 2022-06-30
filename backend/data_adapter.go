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

func (a *dataSDKAdapter) QueryData(ctx context.Context, req *pluginv2.QueryDataRequest, acClient pluginv2.AccessControlClient) (*pluginv2.QueryDataResponse, error) {
	// Convert req to SDK req
	sdkReq := FromProto().QueryDataRequest(req)

	// Set AccessControlClient
	sdkReq.PluginContext.AccessControlClient = FromProto().AccessControlClient(acClient)

	resp, err := a.queryDataHandler.QueryData(ctx, sdkReq)
	if err != nil {
		return nil, err
	}

	return ToProto().QueryDataResponse(resp)
}
