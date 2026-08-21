package grpcplugin

import (
	"context"

	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// CustomRouteServer is the server API for the CustomRoute service.
type CustomRouteServer interface {
	pluginv2.CustomRouteServer
}

// CustomRouteClient is the client API for the CustomRoute service.
type CustomRouteClient interface {
	pluginv2.CustomRouteClient
}

// CustomRouteGRPCPlugin implements the GRPCPlugin interface from github.com/hashicorp/go-plugin.
type CustomRouteGRPCPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	plugin.GRPCPlugin
	CustomRouteServer CustomRouteServer
}

// GRPCServer registers p as a custom route gRPC server.
func (p *CustomRouteGRPCPlugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	pluginv2.RegisterCustomRouteServer(s, &customRouteGRPCServer{
		server: p.CustomRouteServer,
	})
	return nil
}

// GRPCClient returns c as a custom route gRPC client.
func (p *CustomRouteGRPCPlugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &customRouteGRPCClient{client: pluginv2.NewCustomRouteClient(c)}, nil
}

type customRouteGRPCServer struct {
	server CustomRouteServer
}

// CallCustomRoute calls a custom route.
func (s *customRouteGRPCServer) CallCustomRoute(req *pluginv2.CallCustomRouteRequest, srv pluginv2.CustomRoute_CallCustomRouteServer) error {
	return s.server.CallCustomRoute(req, srv)
}

type customRouteGRPCClient struct {
	client pluginv2.CustomRouteClient
}

// CallCustomRoute calls a custom route.
func (m *customRouteGRPCClient) CallCustomRoute(ctx context.Context, req *pluginv2.CallCustomRouteRequest, opts ...grpc.CallOption) (pluginv2.CustomRoute_CallCustomRouteClient, error) {
	return m.client.CallCustomRoute(ctx, req, opts...)
}

var _ CustomRouteServer = &customRouteGRPCServer{}
var _ CustomRouteClient = &customRouteGRPCClient{}
