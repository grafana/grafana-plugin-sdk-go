package grpcplugin

import (
	"context"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// TabularInformationServer represents a stream server.
type TabularInformationServer interface {
	pluginv2.TabularInformationServer
}

// TabularInformationClient represents a stream client.
type TabularInformationClient interface {
	pluginv2.TabularInformationClient
}

// InformationGRPCPlugin implements the GRPCPlugin interface from github.com/hashicorp/go-plugin.
type TabularInformationGRPCPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	plugin.GRPCPlugin
	TabularInformationServer TabularInformationServer
}

// GRPCServer registers p as a resource gRPC server.
func (p *TabularInformationGRPCPlugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	pluginv2.RegisterTabularInformationServer(s, &tabularInformationGRPCServer{
		server: p.TabularInformationServer,
	})
	return nil
}

// GRPCClient returns c as a resource gRPC client.
func (p *TabularInformationGRPCPlugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &tabularInformationGRPCClient{client: pluginv2.NewTabularInformationClient(c)}, nil
}

type tabularInformationGRPCServer struct {
	server TabularInformationServer
}

func (s tabularInformationGRPCServer) Tables(ctx context.Context, request *pluginv2.TableInformationRequest) (*pluginv2.TableInformationResponse, error) {
	return s.server.Tables(ctx, request)
}

type tabularInformationGRPCClient struct {
	client TabularInformationClient
}

func (s tabularInformationGRPCClient) Tables(ctx context.Context, in *pluginv2.TableInformationRequest, opts ...grpc.CallOption) (*pluginv2.TableInformationResponse, error) {
	return s.client.Tables(ctx, in, opts...)
}

var _ TabularInformationServer = &tabularInformationGRPCServer{}
var _ TabularInformationClient = &tabularInformationGRPCClient{}
