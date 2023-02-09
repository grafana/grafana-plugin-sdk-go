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

func withOAuthMiddleware(ctx context.Context, authorization, xIdToken string) context.Context {
	if authorization != "" {
		ctx = httpclient.WithContextualMiddleware(ctx,
			httpclient.MiddlewareFunc(func(opts httpclient.Options, next http.RoundTripper) http.RoundTripper {
				return httpclient.RoundTripperFunc(func(qreq *http.Request) (*http.Response, error) {
					// Only set the Authorization header if it is not already set.
					if qreq.Header.Get("Authorization") == "" {
						qreq.Header.Set("Authorization", authorization)
					}
					// Only set the X-ID-Token header if it is not already set.
					if xIdToken != "" && qreq.Header.Get("X-ID-Token") == "" {
						qreq.Header.Set("X-ID-Token", xIdToken)
					}
					return next.RoundTrip(qreq)
				})
			}))
	}
	return ctx
}

func (a *dataSDKAdapter) QueryData(ctx context.Context, req *pluginv2.QueryDataRequest) (*pluginv2.QueryDataResponse, error) {
	ctx = withOAuthMiddleware(ctx, req.Headers["Authorization"], req.Headers["X-ID-Token"])
	resp, err := a.queryDataHandler.QueryData(ctx, FromProto().QueryDataRequest(req))
	if err != nil {
		return nil, err
	}

	return ToProto().QueryDataResponse(resp)
}
