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

func (a *streamSDKAdapter) CanSubscribeToStream(ctx context.Context, protoReq *pluginv2.SubscribeToStreamRequest) (*pluginv2.SubscribeToStreamResponse, error) {
	if a.streamHandler == nil {
		return nil, status.Error(codes.Unimplemented, "not implemented")
	}
	resp, err := a.streamHandler.CanSubscribeToStream(ctx, FromProto().SubscribeToStreamRequest(protoReq))
	if err != nil {
		return nil, err
	}
	return ToProto().SubscribeToStreamResponse(resp), nil
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
