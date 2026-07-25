package pluginv3example_test

import (
	"context"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/pluginv3example"
	pluginv3 "github.com/grafana/grafana-plugin-sdk-go/genproto/grafana/plugin/v3"
)

// Example wires the example Plugin up over an in-process gRPC connection and
// calls its CallRoute RPC. Registration uses the generated
// pluginv3.RegisterRouteServiceServer directly — no SDK wrapper sits between the
// plugin and the protobuf contract.
func Example() {
	lis := bufconn.Listen(1024 * 1024)

	srv := grpc.NewServer()
	pluginv3.RegisterRouteServiceServer(srv, &pluginv3example.Plugin{})
	go func() {
		if err := srv.Serve(lis); err != nil {
			log.Printf("server exited: %v", err)
		}
	}()
	defer srv.Stop()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := pluginv3.NewRouteServiceClient(conn)
	stream, err := client.CallRoute(context.Background(), pluginv3.CallRouteRequest_builder{
		Method: proto.String("GET"),
		Path:   proto.String("/status"),
	}.Build())
	if err != nil {
		log.Fatal(err)
	}

	resp, err := stream.Recv()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%d %s\n", resp.GetCode(), resp.GetBody())
	// Output: 200 hello from GET /status
}
