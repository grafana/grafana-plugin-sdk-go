package backend

import (
	"context"

	bproto "github.com/grafana/grafana-plugin-sdk-go/genproto/go/grafana_plugin"
	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

// DatasourcePluginImpl implements the plugin interface from github.com/hashicorp/go-plugin.
type DatasourcePluginImpl struct {
	plugin.NetRPCUnsupportedPlugin

	Impl backendPluginWrapper
}

func (p *DatasourcePluginImpl) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	bproto.RegisterBackendPluginServer(s, &grpcServer{
		Impl: p.Impl,
	})
	return nil
}

func (p *DatasourcePluginImpl) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClient{client: bproto.NewBackendPluginClient(c)}, nil
}
