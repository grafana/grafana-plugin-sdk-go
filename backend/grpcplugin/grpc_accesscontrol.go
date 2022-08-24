package grpcplugin

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
)

// HasAccessFunc is a wrapper to allow the use of
// ordinary functions as pluginv2.AccessControlClient. If f is a function
// with the appropriate signature, HasAccessHandlerFunc(f) is a
// Handler that calls f.
type HasAccessFunc func(ctx context.Context, req *pluginv2.HasAccessRequest) (*pluginv2.HasAccessResponse, error)

// HasAccess calls fn(ctx, req).
func (fn HasAccessFunc) HasAccess(ctx context.Context, req *pluginv2.HasAccessRequest, opts ...grpc.CallOption) (*pluginv2.HasAccessResponse, error) {
	return fn(ctx, req)
}

func newAccessControlClient(broker *plugin.GRPCBroker, callbackID uint32) pluginv2.AccessControlClient {
	return HasAccessFunc(func(helperCtx context.Context, helperReq *pluginv2.HasAccessRequest) (*pluginv2.HasAccessResponse, error) {
		conn, err := broker.Dial(callbackID)
		if err != nil {
			return nil, err
		}
		defer conn.Close()

		return pluginv2.NewAccessControlClient(conn).HasAccess(helperCtx, helperReq)
	})
}

// newAccessControlServer starts a new grpc AccessControlServer and returns the broker ID that it is bound to.
// When the provided ctx is Done, the grpc server is stopped.'
func newAccessControlServer(ctx context.Context, broker *plugin.GRPCBroker, acSrv pluginv2.AccessControlServer) uint32 {
	var s *grpc.Server
	serverFunc := func(opts []grpc.ServerOption) *grpc.Server {
		s = grpc.NewServer(opts...)
		pluginv2.RegisterAccessControlServer(s, acSrv)

		return s
	}

	callbackID := broker.NextId()

	go func() {
		broker.AcceptAndServe(callbackID, serverFunc)
		<-ctx.Done()
		s.Stop()
	}()

	return callbackID
}
