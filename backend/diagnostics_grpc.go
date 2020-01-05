package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

// DiagnosticsGRPCPlugin implements plugin.GRPCPlugin for the go-plugin package.
type DiagnosticsGRPCPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	plugin.GRPCPlugin
	GRPCPluginProvider func() pluginv2.DiagnosticsServer
}

func (p *DiagnosticsGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	pluginv2.RegisterDiagnosticsServer(s, p.GRPCPluginProvider())
	return nil
}

func (p *DiagnosticsGRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCDiagnosticsClient{
		client: pluginv2.NewDiagnosticsClient(c),
	}, nil
}

// GRPCDiagnosticsClient handles the client, or core side of the plugin rpc connection.
// The GRPCDiagnosticsClient methods are mostly a translation layer between the
// SDK types and the grpc proto types, directly converting between the two.
type GRPCDiagnosticsClient struct {
	client pluginv2.DiagnosticsClient
}

func (c *GRPCDiagnosticsClient) CollectMetrics(ctx context.Context, req *pluginv2.CollectMetrics_Request) (*pluginv2.CollectMetrics_Response, error) {
	return c.client.CollectMetrics(ctx, req)
}

func (c *GRPCDiagnosticsClient) CheckHealth(ctx context.Context, req *pluginv2.CheckHealth_Request) (*pluginv2.CheckHealth_Response, error) {
	return c.client.CheckHealth(ctx, req)
}

// GRPCDiagnosticsServer handles the server, or plugin side of the rpc connection.
type GRPCDiagnosticsServer struct {
	handler DiagnosticsHandler
}

func (s *GRPCDiagnosticsServer) CollectMetrics(ctx context.Context, req *pluginv2.CollectMetrics_Request) (*pluginv2.CollectMetrics_Response, error) {
	res, err := s.handler.CollectMetrics(ctx)
	if err != nil {
		return nil, err
	}

	return res.toProtobuf(), nil
}

func (s *GRPCDiagnosticsServer) CheckHealth(ctx context.Context, req *pluginv2.CheckHealth_Request) (*pluginv2.CheckHealth_Response, error) {
	res, err := s.handler.CheckHealth(ctx)
	if err != nil {
		return nil, err
	}

	return res.toProtobuf(), nil
}

var _ pluginv2.DiagnosticsServer = &GRPCDiagnosticsServer{}
