package grpcplugin

import (
	"context"

	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	pluginv3 "github.com/grafana/grafana-plugin-sdk-go/genproto/grafana/plugin/v3"
)

// V3Server is implemented by plugins that serve the grafana.plugin.v3 API — the
// modern successor to the legacy genproto/pluginv2 (backend.proto) contract.
//
// Unlike the legacy services (Data, Resource, Diagnostics, ...), the V3 service
// contracts are the generated gRPC interfaces themselves: there is
// intentionally no hand-written Go wrapper translating between the protobuf
// types and an SDK-native type. Implementations embed UnimplementedV3Server and
// override the RPCs they support.
type V3Server interface {
	pluginv3.RouteServiceServer
	pluginv3.AdmissionControlServiceServer
	pluginv3.ResourceConversionServiceServer
}

// UnimplementedV3Server is the stub that plugin authors embed to implement the
// grafana.plugin.v3 API. Embedding it makes the V3 opt-in explicit and supplies
// default (gRPC "Unimplemented") handlers for every V3 RPC, so a plugin only
// needs to override the RPCs it actually serves.
type UnimplementedV3Server struct {
	pluginv3.UnimplementedRouteServiceServer
	pluginv3.UnimplementedAdmissionControlServiceServer
	pluginv3.UnimplementedResourceConversionServiceServer
}

// Compile-time assurance that the stub satisfies the V3 contract.
var _ V3Server = UnimplementedV3Server{}

// The types below are thin go-plugin adapters. go-plugin dispenses plugins by
// name and requires each to implement plugin.GRPCPlugin; the generated code
// only provides Register*Server / New*Client. Each adapter registers the
// generated gRPC service directly — no wrapping server type is inserted.

type routeGRPCPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	plugin.GRPCPlugin
	server pluginv3.RouteServiceServer
}

func (p *routeGRPCPlugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	pluginv3.RegisterRouteServiceServer(s, p.server)
	return nil
}

func (p *routeGRPCPlugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return pluginv3.NewRouteServiceClient(c), nil
}

type admissionControlGRPCPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	plugin.GRPCPlugin
	server pluginv3.AdmissionControlServiceServer
}

func (p *admissionControlGRPCPlugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	pluginv3.RegisterAdmissionControlServiceServer(s, p.server)
	return nil
}

func (p *admissionControlGRPCPlugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return pluginv3.NewAdmissionControlServiceClient(c), nil
}

type resourceConversionGRPCPlugin struct {
	plugin.NetRPCUnsupportedPlugin
	plugin.GRPCPlugin
	server pluginv3.ResourceConversionServiceServer
}

func (p *resourceConversionGRPCPlugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	pluginv3.RegisterResourceConversionServiceServer(s, p.server)
	return nil
}

func (p *resourceConversionGRPCPlugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return pluginv3.NewResourceConversionServiceClient(c), nil
}
