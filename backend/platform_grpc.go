package backend

import (
	"context"

	bproto "github.com/grafana/grafana-plugin-sdk-go/genproto/go/backend_plugin"
	plugin "github.com/hashicorp/go-plugin"
)

type PlatformGRPCClient struct {
	broker *plugin.GRPCBroker
	client bproto.GrafanaPlatformClient
}

type PlatformGRPCServer struct {
	broker *plugin.GRPCBroker
	Impl   platformWrapper
}

func (g *PlatformGRPCServer) DataQuery(ctx context.Context, req *bproto.DataQueryRequest) (*bproto.DataQueryResponse, error) {
	return nil, nil
}

func (g *PlatformGRPCServer) Resource(ctx context.Context, req *bproto.ResourceRequest) (*bproto.ResourceResponse, error) {
	return nil, nil
}
