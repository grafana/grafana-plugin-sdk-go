package backend

import (
	"context"

	bproto "github.com/grafana/grafana-plugin-sdk-go/genproto/go/backend_plugin"
)

// type PlatformGRPCClient struct {
// 	broker *plugin.GRPCBroker
// 	client bproto.GrafanaPlatformClient
// }

// type PlatformGRPCServer struct {
// 	broker *plugin.GRPCBroker
// 	Impl   platformWrapper
// }

// func (g *PlatformGRPCServer) DataQuery(ctx context.Context, req *bproto.DataQueryRequest) (*bproto.DataQueryResponse, error) {
// 	return nil, nil
// }

// func (g *PlatformGRPCServer) Resource(ctx context.Context, req *bproto.ResourceRequest) (*bproto.ResourceResponse, error) {
// 	return nil, nil
// }

type PlatformGrpcApiClient struct {
	client bproto.GrafanaPlatformClient
}

func (g *PlatformGrpcApiClient) DataQuery(ctx context.Context, req *bproto.DataQueryRequest) (*bproto.DataQueryResponse, error) {
	return g.client.DataQuery(ctx, req)
}

func (g *PlatformGrpcApiClient) Resource(ctx context.Context, req *bproto.ResourceRequest) (*bproto.ResourceResponse, error) {
	return g.client.Resource(ctx, req)
}

type PlatformGrpcApiServer struct {
	Impl PlatformAPI
}

func (g *PlatformGrpcApiServer) DataQuery(ctx context.Context, req *bproto.DataQueryRequest) (*bproto.DataQueryResponse, error) {
	return g.Impl.DataQuery(ctx, req)
}

func (g *PlatformGrpcApiServer) Resource(ctx context.Context, req *bproto.ResourceRequest) (*bproto.ResourceResponse, error) {
	return g.Impl.Resource(ctx, req)
}
