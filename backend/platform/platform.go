package platform

import bproto "github.com/grafana/grafana-plugin-sdk-go/genproto/go/grafana_plugin"
import plugin "github.com/hashicorp/go-plugin"

import "context"

import "github.com/grafana/grafana-plugin-sdk-go/backend/common"

type PlatformAPIClient struct {
	bproto.GrafanaPlatformClient
}

type PlatformPluginWrapper struct {
	plugin.NetRPCUnsupportedPlugin

	Handlers Handlers
}

func (p *PlatformPluginWrapper) PlatformPluginQuery(ctx context.Context, req *bproto.DataQueryRequest, api PlatformAPI) (*bproto.DataQueryResponse, error) {
	queries := make([]common.DataQuery, len(req.Queries))
	wrappedAPI := &platformAPIWrapper{api: api}
	_, _ = p.Handlers.PlatformDataQuery(ctx, queries, wrappedAPI)
	return nil, nil
}

type Handlers struct {
	PlatformDataQueryHandler
}

type PlatformResourceHandler interface {
}

type PlatformDataQueryHandler interface {
	PlatformDataQuery(ctx context.Context, queries []common.DataQuery, api PlatformHandler) (common.DataQueryResponse, error)
}

type PlatformHandler interface {
}

type platformAPIWrapper struct {
	api PlatformAPI
}

type PlatformAPI interface {
	PlatformDataQuery(ctx context.Context, queries []common.DataQuery)
}
