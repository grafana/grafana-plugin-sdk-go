package backend

import (
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
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

type callResourceResponseSenderFunc func(resp *CallResourceResponse) error

func (fn callResourceResponseSenderFunc) Send(resp *CallResourceResponse) error {
	return fn(resp)
}

func (a *resourceSDKAdapter) CallResource(protoReq *pluginv2.CallResourceRequest, protoSrv pluginv2.Resource_CallResourceServer) error {
	if a.callResourceHandler == nil {
		return protoSrv.Send(&pluginv2.CallResourceResponse{
			Code: http.StatusNotImplemented,
		})
	}

	fn := callResourceResponseSenderFunc(func(resp *CallResourceResponse) error {
		return protoSrv.Send(ToProto().CallResourceResponse(resp))
	})

	req := FromProto().CallResourceRequest(protoReq)
	headers := stringMapListToStringMap(req.Headers)
	ctx := httpclient.WithContextualMiddleware(protoSrv.Context(),
		forwardedOAuthIdentityMiddleware(headers),
		forwardedCookiesMiddleware(headers))

	return a.callResourceHandler.CallResource(ctx, req, fn)
}

func stringMapListToStringMap(m map[string][]string) map[string]string {
	result := map[string]string{}
	for k, v := range m {
		if len(v) > 0 {
			result[k] = v[0]
		}
	}

	return result
}
