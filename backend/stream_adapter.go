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

	ctx = setupContext(ctx, EndpointSubscribeStream)
	parsedReq := FromProto().SubscribeStreamRequest(protoReq)

	var resp *SubscribeStreamResponse
	err := wrapHandler(ctx, parsedReq.PluginContext, func(ctx context.Context) (RequestStatus, error) {
		var innerErr error
		resp, innerErr = a.streamHandler.SubscribeStream(ctx, parsedReq)
		return RequestStatusFromError(innerErr), innerErr
	})
	if err != nil {
		return nil, err
	}

	return ToProto().SubscribeStreamResponse(resp), nil
}

func (a *streamSDKAdapter) PublishStream(ctx context.Context, protoReq *pluginv2.PublishStreamRequest) (*pluginv2.PublishStreamResponse, error) {
	if a.streamHandler == nil {
		return nil, status.Error(codes.Unimplemented, "not implemented")
	}

	ctx = setupContext(ctx, EndpointPublishStream)
	parsedReq := FromProto().PublishStreamRequest(protoReq)

	var resp *PublishStreamResponse
	err := wrapHandler(ctx, parsedReq.PluginContext, func(ctx context.Context) (RequestStatus, error) {
		var innerErr error
		resp, innerErr = a.streamHandler.PublishStream(ctx, parsedReq)
		return RequestStatusFromError(innerErr), innerErr
	})
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
	ctx = setupContext(ctx, EndpointRunStream)
	parsedReq := FromProto().RunStreamRequest(protoReq)

	return wrapHandler(ctx, parsedReq.PluginContext, func(ctx context.Context) (RequestStatus, error) {
		sender := NewStreamSender(&runStreamServer{protoSrv: protoSrv})
		err := a.streamHandler.RunStream(ctx, parsedReq, sender)
		return RequestStatusFromError(err), err
	})
}
