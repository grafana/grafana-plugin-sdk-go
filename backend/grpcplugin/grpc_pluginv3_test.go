package grpcplugin

import (
	"context"
	"testing"

	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	pluginv3 "github.com/grafana/grafana-plugin-sdk-go/genproto/grafana/plugin/v3"
)

// testV3Server implements the V3 contract by embedding UnimplementedV3Server and
// overriding two RPCs. ConvertObjects is intentionally left to the stub so the
// test can assert it responds with codes.Unimplemented.
type testV3Server struct {
	UnimplementedV3Server
}

func (testV3Server) CallRoute(req *pluginv3.CallRouteRequest, stream grpc.ServerStreamingServer[pluginv3.CallRouteResponse]) error {
	return stream.Send(pluginv3.CallRouteResponse_builder{
		Code: proto.Int32(200),
		Body: []byte("ok:" + req.GetPath()),
	}.Build())
}

func (testV3Server) ValidateAdmission(_ context.Context, _ *pluginv3.ValidateAdmissionRequest) (*pluginv3.ValidateAdmissionResponse, error) {
	return pluginv3.ValidateAdmissionResponse_builder{Allowed: proto.Bool(true)}.Build(), nil
}

// TestServeV3 exercises the real go-plugin dispensing path: it builds the same
// PluginSet that Serve builds (via pluginSet), stands it up in-process with
// go-plugin's test helper, then dispenses each V3 service by name and calls it.
func TestServeV3(t *testing.T) {
	ps := pluginSet(ServeOpts{V3Server: testV3Server{}})

	client, server := plugin.TestPluginGRPCConn(t, false, ps)
	defer server.Stop()
	defer func() { _ = client.Close() }()

	ctx := context.Background()

	t.Run("route CallRoute streams a response", func(t *testing.T) {
		raw, err := client.Dispense("v3-route")
		if err != nil {
			t.Fatal(err)
		}
		rc := raw.(pluginv3.RouteServiceClient)

		stream, err := rc.CallRoute(ctx, pluginv3.CallRouteRequest_builder{
			Method: proto.String("GET"),
			Path:   proto.String("/health"),
		}.Build())
		if err != nil {
			t.Fatal(err)
		}
		resp, err := stream.Recv()
		if err != nil {
			t.Fatal(err)
		}
		if got, want := resp.GetCode(), int32(200); got != want {
			t.Fatalf("code = %d, want %d", got, want)
		}
		if got, want := string(resp.GetBody()), "ok:/health"; got != want {
			t.Fatalf("body = %q, want %q", got, want)
		}
	})

	t.Run("admission ValidateAdmission returns allowed", func(t *testing.T) {
		raw, err := client.Dispense("v3-admission-control")
		if err != nil {
			t.Fatal(err)
		}
		ac := raw.(pluginv3.AdmissionControlServiceClient)

		resp, err := ac.ValidateAdmission(ctx, pluginv3.ValidateAdmissionRequest_builder{
			Operation: pluginv3.Operation_OPERATION_CREATE.Enum(),
		}.Build())
		if err != nil {
			t.Fatal(err)
		}
		if !resp.GetAllowed() {
			t.Fatal("expected allowed = true")
		}
	})

	t.Run("un-overridden ConvertObjects reports Unimplemented", func(t *testing.T) {
		raw, err := client.Dispense("v3-resource-conversion")
		if err != nil {
			t.Fatal(err)
		}
		cc := raw.(pluginv3.ResourceConversionServiceClient)

		_, err = cc.ConvertObjects(ctx, pluginv3.ConvertObjectsRequest_builder{
			Uid: proto.String("abc"),
		}.Build())
		if status.Code(err) != codes.Unimplemented {
			t.Fatalf("expected codes.Unimplemented, got %v", err)
		}
	})
}
