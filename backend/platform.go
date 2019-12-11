package backend

import (
	"context"

	bproto "github.com/grafana/grafana-plugin-sdk-go/genproto/go/backend_plugin"
	plugin "github.com/hashicorp/go-plugin"
)

type PlatformAPIClient struct {
	bproto.GrafanaPlatformClient
}

type platformWrapper struct {
	plugin.NetRPCUnsupportedPlugin

	Handlers PlatformHandlers
}

// func (p *platformWrapper) PlatformPluginQuery(ctx context.Context, req *bproto.DataQueryRequest, api PlatformAPI) (*bproto.DataQueryResponse, error) {
// 	queries := make([]DataQuery, len(req.Queries))
// 	wrappedAPI := &platformAPIWrapper{api: api}
// 	_, _ = p.Handlers.PlatformDataQuery(ctx, queries, wrappedAPI)
// 	return nil, nil
// }

type PlatformHandlers struct {
	PlatformDataQueryHandler
}

type PlatformResourceHandler interface {
}

type PlatformDataQueryHandler interface {
	PlatformDataQuery(ctx context.Context, queries []DataQuery, api PlatformHandler) (DataQueryResponse, error)
}

type PlatformHandler interface {
}

type platformAPIWrapper struct {
	api PlatformAPI
}

type PlatformAPI interface {
	DataQuery(ctx context.Context, req *bproto.DataQueryRequest) (*bproto.DataQueryResponse, error)
	Resource(ctx context.Context, req *bproto.ResourceRequest) (*bproto.ResourceResponse, error)
}
