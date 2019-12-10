package backend

import (
	"context"

	bproto "github.com/grafana/grafana-plugin-sdk-go/genproto/go/grafana_plugin"
	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

// PluginImpl implements the plugin interface from github.com/hashicorp/go-plugin.
type PluginImpl struct {
	plugin.NetRPCUnsupportedPlugin

	Wrap backendPluginWrapper
}

func (p *PluginImpl) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	bproto.RegisterBackendPluginServer(s, &grpcServer{
		Impl: p.Wrap,
	})
	return nil
}

func (p *PluginImpl) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClient{client: bproto.NewBackendPluginClient(c)}, nil
}
