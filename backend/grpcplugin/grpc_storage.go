package grpcplugin

import (
	"context"

	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// StorageServer represents a data server.
type StorageServer interface {
	pluginv2.StorageServer
}

// StorageClient represents a data client.
type StorageClient interface {
	pluginv2.StorageClient
}

// StorageGRPCPlugin implements the GRPCPlugin interface from github.com/hashicorp/go-plugin.
type StorageGRPCPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	plugin.GRPCPlugin
	StorageServer StorageServer
}

// GRPCServer registers p as a data gRPC server.
func (p *StorageGRPCPlugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	pluginv2.RegisterStorageServer(s, &storageGRPCServer{
		server: p.StorageServer,
	})
	return nil
}

// GRPCClient returns c as a data gRPC client.
func (p *StorageGRPCPlugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &storageGRPCClient{client: pluginv2.NewStorageClient(c)}, nil
}

type storageGRPCServer struct {
	server StorageServer
}

func (s *storageGRPCServer) MutateInstanceSettings(ctx context.Context, req *pluginv2.InstanceSettingsAdmissionRequest) (*pluginv2.InstanceSettingsResponse, error) {
	return s.server.MutateInstanceSettings(ctx, req)
}

func (s *storageGRPCServer) ValidateAdmission(ctx context.Context, req *pluginv2.AdmissionRequest) (*pluginv2.StorageResponse, error) {
	return s.server.ValidateAdmission(ctx, req)
}

func (s *storageGRPCServer) MutateAdmission(ctx context.Context, req *pluginv2.AdmissionRequest) (*pluginv2.StorageResponse, error) {
	return s.server.MutateAdmission(ctx, req)
}

func (s *storageGRPCServer) ConvertObject(ctx context.Context, req *pluginv2.ConversionRequest) (*pluginv2.StorageResponse, error) {
	return s.server.ConvertObject(ctx, req)
}

type storageGRPCClient struct {
	client pluginv2.StorageClient
}

func (s *storageGRPCClient) MutateInstanceSettings(ctx context.Context, req *pluginv2.InstanceSettingsAdmissionRequest, opts ...grpc.CallOption) (*pluginv2.InstanceSettingsResponse, error) {
	return s.client.MutateInstanceSettings(ctx, req, opts...)
}

func (s *storageGRPCClient) ValidateAdmission(ctx context.Context, req *pluginv2.AdmissionRequest, opts ...grpc.CallOption) (*pluginv2.StorageResponse, error) {
	return s.client.ValidateAdmission(ctx, req, opts...)
}

func (s *storageGRPCClient) MutateAdmission(ctx context.Context, req *pluginv2.AdmissionRequest, opts ...grpc.CallOption) (*pluginv2.StorageResponse, error) {
	return s.client.MutateAdmission(ctx, req, opts...)
}

func (s *storageGRPCClient) ConvertObject(ctx context.Context, req *pluginv2.ConversionRequest, opts ...grpc.CallOption) (*pluginv2.StorageResponse, error) {
	return s.client.ConvertObject(ctx, req, opts...)
}

var _ StorageServer = &storageGRPCServer{}
var _ StorageClient = &storageGRPCClient{}
