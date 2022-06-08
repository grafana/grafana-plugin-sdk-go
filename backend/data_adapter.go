package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
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

func (a *dataSDKAdapter) QueryData(ctx context.Context, protoReq *pluginv2.QueryDataRequest) (*pluginv2.QueryDataResponse, error) {
	req := FromProto().QueryDataRequest(protoReq)
	ctx = httpclient.WithContextualMiddleware(ctx,
		forwardedOAuthIdentityMiddleware(req.Headers),
		forwardedCookiesMiddleware(req.Headers))

	resp, err := a.queryDataHandler.QueryData(ctx, req)
	if err != nil {
		return nil, err
	}

	return ToProto().QueryDataResponse(resp)
}
