package backend

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	"github.com/grafana/grafana-plugin-sdk-go/internal/tenant"
)

func TestSubscribeStream(t *testing.T) {
	t.Run("When tenant information is attached to incoming context, it is propagated from adapter to handler", func(t *testing.T) {
		tid := "123456"
		a := newStreamSDKAdapter(&streamAdapter{
			subscribeStreamFunc: func(ctx context.Context, _ *SubscribeStreamRequest) (*SubscribeStreamResponse, error) {
				require.Equal(t, tid, tenant.IDFromContext(ctx))
				return &SubscribeStreamResponse{}, nil
			},
		})

		ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
			tenant.CtxKey: tid,
		}))

		_, err := a.SubscribeStream(ctx, &pluginv2.SubscribeStreamRequest{
			PluginContext: &pluginv2.PluginContext{},
		})
		require.NoError(t, err)
	})

	t.Run("It should not crash if a panic occurs in the handler", func(t *testing.T) {
		a := newStreamSDKAdapter(&streamAdapter{
			subscribeStreamFunc: func(_ context.Context, _ *SubscribeStreamRequest) (*SubscribeStreamResponse, error) {
				panic("test")
			},
		})

		_, err := a.SubscribeStream(context.Background(), &pluginv2.SubscribeStreamRequest{
			PluginContext: &pluginv2.PluginContext{},
		})
		require.ErrorContains(t, err, "internal server error")
	})
}

func TestPublishStream(t *testing.T) {
	t.Run("When tenant information is attached to incoming context, it is propagated from adapter to handler", func(t *testing.T) {
		tid := "123456"
		a := newStreamSDKAdapter(&streamAdapter{
			publishStreamFunc: func(ctx context.Context, _ *PublishStreamRequest) (*PublishStreamResponse, error) {
				require.Equal(t, tid, tenant.IDFromContext(ctx))
				return &PublishStreamResponse{}, nil
			},
		})

		ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
			tenant.CtxKey: tid,
		}))

		_, err := a.PublishStream(ctx, &pluginv2.PublishStreamRequest{
			PluginContext: &pluginv2.PluginContext{},
		})
		require.NoError(t, err)
	})

	t.Run("It should not crash if a panic occurs in the handler", func(t *testing.T) {
		a := newStreamSDKAdapter(&streamAdapter{
			publishStreamFunc: func(_ context.Context, _ *PublishStreamRequest) (*PublishStreamResponse, error) {
				panic("test")
			},
		})

		_, err := a.PublishStream(context.Background(), &pluginv2.PublishStreamRequest{
			PluginContext: &pluginv2.PluginContext{},
		})
		require.ErrorContains(t, err, "internal server error")
	})
}

func TestRunStream(t *testing.T) {
	t.Run("When tenant information is attached to incoming context, it is propagated from adapter to handler", func(t *testing.T) {
		tid := "123456"
		a := newStreamSDKAdapter(&streamAdapter{
			runStreamFunc: func(ctx context.Context, _ *RunStreamRequest, _ *StreamSender) error {
				require.Equal(t, tid, tenant.IDFromContext(ctx))
				return nil
			},
		})

		testSrv := newTestRunStreamServer()
		testSrv.WithContext(metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
			tenant.CtxKey: tid,
		})))

		err := a.RunStream(&pluginv2.RunStreamRequest{
			PluginContext: &pluginv2.PluginContext{},
		}, testSrv)
		require.NoError(t, err)
	})

	t.Run("It should not crash if a panic occurs in the handler", func(t *testing.T) {
		a := newStreamSDKAdapter(&streamAdapter{
			runStreamFunc: func(_ context.Context, _ *RunStreamRequest, _ *StreamSender) error {
				panic("test")
			},
		})

		testSrv := newTestRunStreamServer()
		err := a.RunStream(&pluginv2.RunStreamRequest{
			PluginContext: &pluginv2.PluginContext{},
		}, testSrv)
		require.ErrorContains(t, err, "internal server error")
	})
}

type streamAdapter struct {
	subscribeStreamFunc func(context.Context, *SubscribeStreamRequest) (*SubscribeStreamResponse, error)
	publishStreamFunc   func(context.Context, *PublishStreamRequest) (*PublishStreamResponse, error)
	runStreamFunc       func(context.Context, *RunStreamRequest, *StreamSender) error
}

func (a *streamAdapter) SubscribeStream(ctx context.Context, req *SubscribeStreamRequest) (*SubscribeStreamResponse, error) {
	if a.subscribeStreamFunc != nil {
		return a.subscribeStreamFunc(ctx, req)
	}
	return nil, nil
}

func (a *streamAdapter) PublishStream(ctx context.Context, req *PublishStreamRequest) (*PublishStreamResponse, error) {
	if a.publishStreamFunc != nil {
		return a.publishStreamFunc(ctx, req)
	}
	return nil, nil
}

func (a *streamAdapter) RunStream(ctx context.Context, req *RunStreamRequest, sender *StreamSender) error {
	if a.runStreamFunc != nil {
		return a.runStreamFunc(ctx, req, sender)
	}
	return nil
}

type testRunStreamServer struct {
	ctx          context.Context
	respMessages []*pluginv2.StreamPacket
}

func newTestRunStreamServer() *testRunStreamServer {
	return &testRunStreamServer{
		respMessages: []*pluginv2.StreamPacket{},
		ctx:          context.Background(),
	}
}

func (srv *testRunStreamServer) Send(resp *pluginv2.StreamPacket) error {
	srv.respMessages = append(srv.respMessages, resp)
	return nil
}

func (srv *testRunStreamServer) SetHeader(metadata.MD) error {
	return nil
}

func (srv *testRunStreamServer) SendHeader(metadata.MD) error {
	return nil
}

func (srv *testRunStreamServer) SetTrailer(metadata.MD) {

}

func (srv *testRunStreamServer) Context() context.Context {
	return srv.ctx
}

func (srv *testRunStreamServer) SendMsg(_ interface{}) error {
	return nil
}

func (srv *testRunStreamServer) RecvMsg(_ interface{}) error {
	return nil
}

func (srv *testRunStreamServer) WithContext(ctx context.Context) {
	srv.ctx = ctx
}
