package backend

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/grafana/grafana-plugin-sdk-go/backend/tenant"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// streamSDKAdapter adapter between low level plugin protocol and SDK interfaces.
type streamSDKAdapter struct {
	streamHandler StreamHandler
}

func newStreamSDKAdapter(handler StreamHandler) *streamSDKAdapter {
	return &streamSDKAdapter{
		streamHandler: handler,
	}
}

func (a *streamSDKAdapter) SubscribeStream(ctx context.Context, protoReq *pluginv2.SubscribeStreamRequest) (*pluginv2.SubscribeStreamResponse, error) {
	if a.streamHandler == nil {
		return nil, status.Error(codes.Unimplemented, "not implemented")
	}
	if tid, exists := tenant.IDFromIncomingGRPCContext(ctx); exists {
		ctx = tenant.WithTenant(ctx, tid)
	}
	resp, err := a.streamHandler.SubscribeStream(ctx, FromProto().SubscribeStreamRequest(protoReq))
	if err != nil {
		return nil, err
	}
	return ToProto().SubscribeStreamResponse(resp), nil
}

func (a *streamSDKAdapter) PublishStream(ctx context.Context, protoReq *pluginv2.PublishStreamRequest) (*pluginv2.PublishStreamResponse, error) {
	if a.streamHandler == nil {
		return nil, status.Error(codes.Unimplemented, "not implemented")
	}
	if tid, exists := tenant.IDFromIncomingGRPCContext(ctx); exists {
		ctx = tenant.WithTenant(ctx, tid)
	}
	resp, err := a.streamHandler.PublishStream(ctx, FromProto().PublishStreamRequest(protoReq))
	if err != nil {
		return nil, err
	}
	return ToProto().PublishStreamResponse(resp), nil
}

type runStreamServer struct {
	protoSrv pluginv2.Stream_RunStreamServer
}

func (r *runStreamServer) Send(packet *StreamPacket) error {
	return r.protoSrv.Send(ToProto().StreamPacket(packet))
}

func (a *streamSDKAdapter) RunStream(protoReq *pluginv2.RunStreamRequest, protoSrv pluginv2.Stream_RunStreamServer) error {
	if a.streamHandler == nil {
		return status.Error(codes.Unimplemented, "not implemented")
	}
	ctx := protoSrv.Context()
	if tid, exists := tenant.IDFromIncomingGRPCContext(ctx); exists {
		ctx = tenant.WithTenant(ctx, tid)
	}
	sender := NewStreamSender(&runStreamServer{protoSrv: protoSrv})
	return a.streamHandler.RunStream(ctx, FromProto().RunStreamRequest(protoReq), sender)
}
