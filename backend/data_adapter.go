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
	resp, err := a.queryDataHandler.QueryData(ctx, FromProto().QueryDataRequest(req))
	if err != nil {
		return nil, err
	}

	accept := ""

	if req.Headers != nil {
		if val, exists := req.Headers["accept"]; exists {
			accept = val
		}
	}

	return ToProto().QueryDataResponse(resp, accept)
}
