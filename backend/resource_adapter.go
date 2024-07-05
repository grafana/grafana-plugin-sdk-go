package backend

import (
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// resourceSDKAdapter adapter between low level plugin protocol and SDK interfaces.
type resourceSDKAdapter struct {
	callResourceHandler CallResourceHandler
}

func newResourceSDKAdapter(handler CallResourceHandler) *resourceSDKAdapter {
	return &resourceSDKAdapter{
		callResourceHandler: handler,
	}
}

func (a *resourceSDKAdapter) CallResource(protoReq *pluginv2.CallResourceRequest, protoSrv pluginv2.Resource_CallResourceServer) error {
	if a.callResourceHandler == nil {
		return protoSrv.Send(&pluginv2.CallResourceResponse{
			Code: http.StatusNotImplemented,
		})
	}

	fn := CallResourceResponseSenderFunc(func(resp *CallResourceResponse) error {
		return protoSrv.Send(ToProto().CallResourceResponse(resp))
	})

	ctx := protoSrv.Context()
	ctx = WithEndpoint(ctx, EndpointCallResource)
	ctx = propagateTenantIDIfPresent(ctx)
	ctx = WithGrafanaConfig(ctx, NewGrafanaCfg(protoReq.PluginContext.GrafanaConfig))
	parsedReq := FromProto().CallResourceRequest(protoReq)
	ctx = WithPluginContext(ctx, parsedReq.PluginContext)
	ctx = WithUser(ctx, parsedReq.PluginContext.User)
	ctx = withHeaderMiddleware(ctx, parsedReq.GetHTTPHeaders())
	ctx = withContextualLogAttributes(ctx, parsedReq.PluginContext)
	ctx = WithUserAgent(ctx, parsedReq.PluginContext.UserAgent)
	return a.callResourceHandler.CallResource(ctx, parsedReq, fn)
}
