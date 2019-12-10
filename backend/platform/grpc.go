package platform

import (
	"context"

	bproto "github.com/grafana/grafana-plugin-sdk-go/genproto/go/grafana_plugin"
	plugin "github.com/hashicorp/go-plugin"
)

// GRPCClient is an implementation of TransformPluginClient that talks over RPC.
type GRPCClient struct {
	broker *plugin.GRPCBroker
	client bproto.GrafanaPlatformClient
}

// GRPCServer is the gRPC server that GRPCClient talks to.
type GRPCServer struct {
	broker *plugin.GRPCBroker
	Impl   PlatformPluginWrapper
}

// TODO DataQuery rename
func (g *GRPCServer) PlatformPluginQuery(ctx context.Context, req *bproto.DataQueryRequest) (*bproto.DataQueryResponse, error) {
	return nil, nil
}

// TODO resource rename
func (g *GRPCServer) PlatformPluginRequest(ctx context.Context, req *bproto.ResourceRequest) (*bproto.ResourceResponse, error) {
	return nil, nil
}
