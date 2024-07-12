package backend

import (
	"context"
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// dataSDKAdapter adapter between low level plugin protocol and SDK interfaces.
type dataSDKAdapter struct {
	queryDataHandler      QueryDataHandler
	queryMigrationHandler QueryMigrationHandler
}

func newDataSDKAdapter(handler QueryDataHandler) *dataSDKAdapter {
	return &dataSDKAdapter{
		queryDataHandler: handler,
	}
}

func newDataSDKAdapterWithQueryMigration(handler QueryDataHandler, queryMigrationHandler QueryMigrationHandler) *dataSDKAdapter {
	return &dataSDKAdapter{
		queryDataHandler:      handler,
		queryMigrationHandler: queryMigrationHandler,
	}
}

func withHeaderMiddleware(ctx context.Context, headers http.Header) context.Context {
	if len(headers) > 0 {
		ctx = httpclient.WithContextualMiddleware(ctx,
			httpclient.MiddlewareFunc(func(opts httpclient.Options, next http.RoundTripper) http.RoundTripper {
				if !opts.ForwardHTTPHeaders {
					return next
				}

				return httpclient.RoundTripperFunc(func(qreq *http.Request) (*http.Response, error) {
					// Only set a header if it is not already set.
					for k, v := range headers {
						if qreq.Header.Get(k) == "" {
							for _, vv := range v {
								qreq.Header.Add(k, vv)
							}
						}
					}
					return next.RoundTrip(qreq)
				})
			}))
	}
	return ctx
}

func (a *dataSDKAdapter) QueryData(ctx context.Context, req *pluginv2.QueryDataRequest) (*pluginv2.QueryDataResponse, error) {
	ctx = WithEndpoint(ctx, EndpointQueryData)
	ctx = propagateTenantIDIfPresent(ctx)
	grafanaCfg := NewGrafanaCfg(req.PluginContext.GrafanaConfig)
	ctx = WithGrafanaConfig(ctx, grafanaCfg)
	parsedReq := FromProto().QueryDataRequest(req)
	ctx = WithPluginContext(ctx, parsedReq.PluginContext)
	ctx = WithUser(ctx, parsedReq.PluginContext.User)
	ctx = withHeaderMiddleware(ctx, parsedReq.GetHTTPHeaders())
	ctx = withContextualLogAttributes(ctx, parsedReq.PluginContext)
	ctx = WithUserAgent(ctx, parsedReq.PluginContext.UserAgent)
	if a.queryMigrationHandler != nil && grafanaCfg.FeatureToggles().IsEnabled("queryMigrations") {
		resp, err := a.queryMigrationHandler.MigrateQuery(ctx, &QueryMigrationRequest{
			PluginContext: parsedReq.PluginContext,
			Queries:       parsedReq.Queries,
		})
		if err != nil {
			return nil, err
		}
		parsedReq.Queries = resp.Queries
	}
	resp, err := a.queryDataHandler.QueryData(ctx, parsedReq)
	if err != nil {
		return nil, err
	}

	return ToProto().QueryDataResponse(resp)
}
