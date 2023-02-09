package backend

import (
	"context"
	"net/http"

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

func withHeaderMiddleware(ctx context.Context, headers map[string]string) context.Context {
	if len(headers) > 0 {
		ctx = httpclient.WithContextualMiddleware(ctx,
			httpclient.MiddlewareFunc(func(opts httpclient.Options, next http.RoundTripper) http.RoundTripper {
				return httpclient.RoundTripperFunc(func(qreq *http.Request) (*http.Response, error) {
					// Only set a header if it is not already set.
					for k, v := range headers {
						if qreq.Header.Get(k) == "" {
							qreq.Header.Set(k, v)
						}
					}
					return next.RoundTrip(qreq)
				})
			}))
	}
	return ctx
}

func (a *dataSDKAdapter) QueryData(ctx context.Context, req *pluginv2.QueryDataRequest) (*pluginv2.QueryDataResponse, error) {
	ctx = withHeaderMiddleware(ctx, req.Headers)
	resp, err := a.queryDataHandler.QueryData(ctx, FromProto().QueryDataRequest(req))
	if err != nil {
		return nil, err
	}

	return ToProto().QueryDataResponse(resp)
}
