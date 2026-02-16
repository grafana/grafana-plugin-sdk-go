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

func (s informationGRPCServer) Tables(ctx context.Context, request *pluginv2.TableInformationRequest) (*pluginv2.TableInformationResponse, error) {
	return s.server.Tables(ctx, request)
}

type informationGRPCClient struct {
	client InformationClient
}

func (s informationGRPCClient) Tables(ctx context.Context, in *pluginv2.TableInformationRequest, opts ...grpc.CallOption) (*pluginv2.TableInformationResponse, error) {
	return s.client.Tables(ctx, in, opts...)
}

var _ InformationServer = &informationGRPCServer{}
var _ InformationClient = &informationGRPCClient{}
