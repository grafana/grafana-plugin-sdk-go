// Package pluginv3example is a minimal, self-contained example of implementing
// the grafana.plugin.v3 API (the modern successor to the legacy pluginv2 /
// backend.proto contract).
//
// It shows the intended shape for plugin authors:
//
//   - embed [grpcplugin.UnimplementedV3Server] to opt into V3 and stub every RPC,
//   - override only the RPCs the plugin actually serves,
//   - work with the generated protobuf messages directly — there is no
//     PluginContext and no hand-written SDK-native wrapper type. Messages use
//     the edition-2024 opaque API (getters + builders).
//
// See the runnable Example in this package for wiring a server and client.
package pluginv3example

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"github.com/grafana/grafana-plugin-sdk-go/backend/grpcplugin"
	pluginv3 "github.com/grafana/grafana-plugin-sdk-go/genproto/grafana/plugin/v3"
)

// Plugin implements the grafana.plugin.v3 API.
//
// By embedding grpcplugin.UnimplementedV3Server it satisfies the full V3
// contract, so any RPC it does not override (here: MutateAdmission and
// ConvertObjects) automatically responds with gRPC codes.Unimplemented.
type Plugin struct {
	grpcplugin.UnimplementedV3Server
}

// Compile-time assurance that Plugin is a complete V3 server.
var _ grpcplugin.V3Server = (*Plugin)(nil)

// CallRoute handles an HTTP-style request and streams a response. Note the
// absence of a PluginContext, and that the request/response are the generated
// protobuf messages themselves — accessed via getters and constructed via the
// opaque-API builder.
func (p *Plugin) CallRoute(req *pluginv3.CallRouteRequest, stream grpc.ServerStreamingServer[pluginv3.CallRouteResponse]) error {
	body := fmt.Sprintf("hello from %s %s", req.GetMethod(), req.GetPath())

	return stream.Send(pluginv3.CallRouteResponse_builder{
		Code: proto.Int32(200),
		Body: []byte(body),
	}.Build())
}

// ValidateAdmission shows overriding a unary RPC as well.
func (p *Plugin) ValidateAdmission(_ context.Context, _ *pluginv3.ValidateAdmissionRequest) (*pluginv3.ValidateAdmissionResponse, error) {
	return pluginv3.ValidateAdmissionResponse_builder{
		Allowed: proto.Bool(true),
	}.Build(), nil
}
