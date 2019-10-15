package grafana

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/datasource"
	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

// DatasourcePlugin implements the plugin interface from github.com/hashicorp/go-plugin.
type DatasourcePlugin struct {
	plugin.NetRPCUnsupportedPlugin

	Impl datasourcePluginWrapper
}

func (p *DatasourcePlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	datasource.RegisterDatasourcePluginServer(s, &grpcServer{Impl: p.Impl})
	return nil
}

func (p *DatasourcePlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &grpcClient{client: datasource.NewDatasourcePluginClient(c)}, nil
}
