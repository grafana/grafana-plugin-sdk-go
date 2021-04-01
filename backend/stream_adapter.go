package backend

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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
	resp, err := a.streamHandler.PublishStream(ctx, FromProto().PublishStreamRequest(protoReq))
	if err != nil {
		return nil, err
	}
	return ToProto().PublishStreamResponse(resp), nil
}

type streamPacketSenderFunc func(resp *StreamPacket) error

func (fn streamPacketSenderFunc) Send(resp *StreamPacket) error {
	return fn(resp)
}

func (a *streamSDKAdapter) RunStream(protoReq *pluginv2.RunStreamRequest, protoSrv pluginv2.Stream_RunStreamServer) error {
	if a.streamHandler == nil {
		return status.Error(codes.Unimplemented, "not implemented")
	}

	fn := streamPacketSenderFunc(func(p *StreamPacket) error {
		return protoSrv.Send(ToProto().StreamPacket(p))
	})

	return a.streamHandler.RunStream(protoSrv.Context(), FromProto().RunStreamRequest(protoReq), fn)
}
