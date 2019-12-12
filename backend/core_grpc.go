package backend

import (
	"context"

	bproto "github.com/grafana/grafana-plugin-sdk-go/genproto/go/backend_plugin"
	plugin "github.com/hashicorp/go-plugin"
)

type CoreGRPCClient struct {
	broker *plugin.GRPCBroker
	client bproto.CoreClient
}

// Plugin is the Grafana Backend plugin interface.
// It corresponds to: grafana.plugin protobuf: BackendPlugin Service | genproto/go/grafana_plugin: BackendPluginClient interface
type Plugin interface {
	Check(ctx context.Context, req *bproto.PluginStatusRequest) (*bproto.PluginStatusResponse, error)
	DataQuery(ctx context.Context, req *bproto.DataQueryRequest) (*bproto.DataQueryResponse, error)
	Resource(ctx context.Context, req *bproto.ResourceRequest) (*bproto.ResourceResponse, error)
}

type coreGRPCServer struct {
	broker *plugin.GRPCBroker
	Impl   coreWrapper
}

func (m *CoreGRPCClient) DataQuery(ctx context.Context, req *bproto.DataQueryRequest) (*bproto.DataQueryResponse, error) {
	return m.client.DataQuery(ctx, req)
}

func (m *coreGRPCServer) DataQuery(ctx context.Context, req *bproto.DataQueryRequest) (*bproto.DataQueryResponse, error) {
	return m.Impl.DataQuery(ctx, req)
}

func (m *CoreGRPCClient) Check(ctx context.Context, req *bproto.PluginStatusRequest) (*bproto.PluginStatusResponse, error) {
	return m.client.Check(ctx, req)
}

func (m *coreGRPCServer) Check(ctx context.Context, req *bproto.PluginStatusRequest) (*bproto.PluginStatusResponse, error) {
	return m.Impl.Check(ctx, req)
}

func (m *CoreGRPCClient) Resource(ctx context.Context, req *bproto.ResourceRequest) (*bproto.ResourceResponse, error) {
	return m.client.Resource(ctx, req)
}

func (m *coreGRPCServer) Resource(ctx context.Context, req *bproto.ResourceRequest) (*bproto.ResourceResponse, error) {
	return m.Impl.Resource(ctx, req)
}
