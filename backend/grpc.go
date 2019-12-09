package backend

import (
	"context"

	bproto "github.com/grafana/grafana-plugin-sdk-go/genproto/go/grafana_plugin"
)

type GRPCClient struct {
	client bproto.BackendPluginClient
}

// BackendPlugin is the Grafana Backend plugin interface.
// It corresponds to: grafana.plugin protobuf: BackendPlugin Service | genproto/go/grafana_plugin: BackendPluginClient interface
type BackendPlugin interface {
	Check(ctx context.Context, req *bproto.PluginStatusRequest) (*bproto.PluginStatusResponse, error)
	DataQuery(ctx context.Context, req *bproto.DataQueryRequest) (*bproto.DataQueryResponse, error)
}

type grpcServer struct {
	Impl backendPluginWrapper
}

func (m *GRPCClient) DataQuery(ctx context.Context, req *bproto.DataQueryRequest) (*bproto.DataQueryResponse, error) {
	return m.client.DataQuery(ctx, req)
}

func (m *grpcServer) DataQuery(ctx context.Context, req *bproto.DataQueryRequest) (*bproto.DataQueryResponse, error) {
	return m.Impl.DataQuery(ctx, req)
}

func (m *GRPCClient) Check(ctx context.Context, req *bproto.PluginStatusRequest) (*bproto.PluginStatusResponse, error) {
	return m.client.Check(ctx, req)
}

func (m *grpcServer) Check(ctx context.Context, req *bproto.PluginStatusRequest) (*bproto.PluginStatusResponse, error) {
	return m.Impl.Check(ctx, req)
}

func (m *GRPCClient) REST(ctx context.Context, req *bproto.RESTRequest) (*bproto.RESTResponse, error) {
	return m.client.REST(ctx, req)
}

func (m *grpcServer) REST(ctx context.Context, req *bproto.RESTRequest) (*bproto.RESTResponse, error) {
	return m.Impl.REST(ctx, req)
}
