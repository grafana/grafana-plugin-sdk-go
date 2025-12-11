package grpcplugin

import (
	"context"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// InformationServer represents a stream server.
type InformationServer interface {
	pluginv2.InformationServer
}

// InformationClient represents a stream client.
type InformationClient interface {
	pluginv2.InformationClient
}

// InformationGRPCPlugin implements the GRPCPlugin interface from github.com/hashicorp/go-plugin.
type InformationGRPCPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	plugin.GRPCPlugin
	InformationServer InformationServer
}

// GRPCServer registers p as a resource gRPC server.
func (p *InformationGRPCPlugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	pluginv2.RegisterInformationServer(s, &informationGRPCServer{
		server: p.InformationServer,
	})
	return nil
}

// GRPCClient returns c as a resource gRPC client.
func (p *InformationGRPCPlugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &informationGRPCClient{client: pluginv2.NewInformationClient(c)}, nil
}

type informationGRPCServer struct {
	server InformationServer
}

func (s informationGRPCServer) Schema(ctx context.Context, request *pluginv2.SchemaRequest) (*pluginv2.SchemaResponse, error) {
	return s.server.Schema(ctx, request)
}

type informationGRPCClient struct {
	client InformationClient
}

func (s informationGRPCClient) Schema(ctx context.Context, in *pluginv2.SchemaRequest, opts ...grpc.CallOption) (*pluginv2.SchemaResponse, error) {
	return s.client.Schema(ctx, in, opts...)
}

var _ InformationServer = &informationGRPCServer{}
var _ InformationClient = &informationGRPCClient{}
