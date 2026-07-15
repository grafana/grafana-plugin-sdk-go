package backend

import (
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// customRouteSDKAdapter adapter between low level plugin protocol and SDK interfaces.
type customRouteSDKAdapter struct {
	customRouteHandler CustomRouteHandler
}

func newCustomRouteSDKAdapter(handler CustomRouteHandler) *customRouteSDKAdapter {
	return &customRouteSDKAdapter{
		customRouteHandler: handler,
	}
}

func (a *customRouteSDKAdapter) CallCustomRoute(protoReq *pluginv2.CallCustomRouteRequest, protoSrv pluginv2.CustomRoute_CallCustomRouteServer) error {
	if a.customRouteHandler == nil {
		return protoSrv.Send(&pluginv2.CallCustomRouteResponse{
			Code: http.StatusNotImplemented,
		})
	}

	fn := CallCustomRouteResponseSenderFunc(func(resp *CallCustomRouteResponse) error {
		return protoSrv.Send(ToProto().CallCustomRouteResponse(resp))
	})

	ctx := protoSrv.Context()
	parsedReq := FromProto().CallCustomRouteRequest(protoReq)
	return a.customRouteHandler.CallCustomRoute(ctx, parsedReq, fn)
}
