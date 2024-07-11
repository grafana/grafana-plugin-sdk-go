package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// queryMigrationSDKAdapter adapter between low level plugin protocol and SDK interfaces.
type queryMigrationSDKAdapter struct {
	handler QueryMigrationHandler
}

func newQueryMigrationSDKAdapter(handler QueryMigrationHandler) *queryMigrationSDKAdapter {
	return &queryMigrationSDKAdapter{
		handler: handler,
	}
}

func (a *queryMigrationSDKAdapter) MigrateQuery(ctx context.Context, req *pluginv2.QueryMigrationRequest) (*pluginv2.QueryMigrationResponse, error) {
	ctx = WithEndpoint(ctx, EndpointQueryMigration)
	ctx = propagateTenantIDIfPresent(ctx)
	ctx = WithGrafanaConfig(ctx, NewGrafanaCfg(req.PluginContext.GrafanaConfig))
	parsedReq := FromProto().QueryMigrationRequest(req)
	ctx = WithPluginContext(ctx, parsedReq.PluginContext)
	ctx = WithUser(ctx, parsedReq.PluginContext.User)
	ctx = withContextualLogAttributes(ctx, parsedReq.PluginContext)
	ctx = WithUserAgent(ctx, parsedReq.PluginContext.UserAgent)
	resp, err := a.handler.MigrateQuery(ctx, parsedReq)
	if err != nil {
		return nil, err
	}
	return ToProto().QueryMigrationResponse(resp), nil
}
