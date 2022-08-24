package grpcplugin

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

// DataServer represents a data server for the SDK Adapter layer
type DataServer interface {
	// pluginv2.DataServer
	QueryData(context.Context, *pluginv2.QueryDataRequest, pluginv2.AccessControlClient) (*pluginv2.QueryDataResponse, error)
}

// DataServer represents a data client for the SDK Adapter layer
type DataClient interface {
	// pluginv2.DataClient
	// TODO check if we want to abstract this
	QueryData(ctx context.Context, req *pluginv2.QueryDataRequest, acSrv pluginv2.AccessControlServer, opts ...grpc.CallOption) (*pluginv2.QueryDataResponse, error)
}

// DataGRPCPlugin implements the GRPCPlugin interface from github.com/hashicorp/go-plugin.
type DataGRPCPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	plugin.GRPCPlugin
	DataServer DataServer
}

// GRPCServer registers p as a data gRPC server.
func (p *DataGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	pluginv2.RegisterDataServer(s, &dataGRPCServer{
		server: p.DataServer,
		broker: broker,
	})
	return nil
}

// GRPCClient returns c as a data gRPC client.
func (p *DataGRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &dataGRPCClient{
		client: pluginv2.NewDataClient(c),
		broker: broker,
	}, nil
}

type dataGRPCServer struct {
	server DataServer
	broker *plugin.GRPCBroker
}

// QueryData queries s for data.
func (s *dataGRPCServer) QueryData(ctx context.Context, req *pluginv2.QueryDataRequest) (*pluginv2.QueryDataResponse, error) {
	accessControlClient := newAccessControlClient(s.broker, req.PluginContext.CallbackServerID)
	return s.server.QueryData(ctx, req, accessControlClient)
}

type dataGRPCClient struct {
	client pluginv2.DataClient
	broker *plugin.GRPCBroker
}

// QueryData start a grpc accesscontrol server then queries m for data.
func (m *dataGRPCClient) QueryData(ctx context.Context, req *pluginv2.QueryDataRequest, acSrv pluginv2.AccessControlServer, opts ...grpc.CallOption) (*pluginv2.QueryDataResponse, error) {
	req.PluginContext.CallbackServerID = newAccessControlServer(ctx, m.broker, acSrv)
	return m.client.QueryData(ctx, req, opts...)
}

var _ pluginv2.DataServer = &dataGRPCServer{}
var _ DataClient = &dataGRPCClient{}
