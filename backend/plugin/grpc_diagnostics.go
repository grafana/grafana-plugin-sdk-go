package plugin

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

type DiagnosticsServer interface {
	pluginv2.DiagnosticsServer
}

// DiagnosticsGRPCPlugin implements the GRPCPlugin interface from github.com/hashicorp/go-plugin.
type DiagnosticsGRPCPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	plugin.GRPCPlugin
	DiagnosticsServer DiagnosticsServer
}

func (p *DiagnosticsGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	pluginv2.RegisterDiagnosticsServer(s, &diagnosticsGRPCServer{
		server: p.DiagnosticsServer,
	})
	return nil
}

func (p *DiagnosticsGRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &diagnosticsGRPCClient{client: pluginv2.NewDiagnosticsClient(c)}, nil
}

type diagnosticsGRPCServer struct {
	server DiagnosticsServer
}

func (s *diagnosticsGRPCServer) CollectMetrics(ctx context.Context, req *pluginv2.CollectMetricsRequest) (*pluginv2.CollectMetricsResponse, error) {
	return s.server.CollectMetrics(ctx, req)
}

func (s *diagnosticsGRPCServer) CheckHealth(ctx context.Context, req *pluginv2.CheckHealthRequest) (*pluginv2.CheckHealthResponse, error) {
	return s.server.CheckHealth(ctx, req)
}

type diagnosticsGRPCClient struct {
	client pluginv2.DiagnosticsClient
}

func (s *diagnosticsGRPCClient) CollectMetrics(ctx context.Context, req *pluginv2.CollectMetricsRequest) (*pluginv2.CollectMetricsResponse, error) {
	return s.client.CollectMetrics(ctx, req)
}

func (s *diagnosticsGRPCClient) CheckHealth(ctx context.Context, req *pluginv2.CheckHealthRequest) (*pluginv2.CheckHealthResponse, error) {
	return s.client.CheckHealth(ctx, req)
}

var _ DiagnosticsServer = &diagnosticsGRPCServer{}
var _ DiagnosticsServer = &diagnosticsGRPCClient{}
