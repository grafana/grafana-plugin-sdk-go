package backend

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

func checkCtxLogger(t *testing.T, ctx context.Context) {
	// Make sure we have a ctx logger and that it's different from the DefaultLogger
	ctxLogger := log.FromContext(ctx)
	require.NotEqual(t, log.DefaultLogger, ctxLogger)
	require.NotNil(t, ctxLogger)
}

func TestContextualLogger(t *testing.T) {
	t.Run("DataSDKAdapter", func(t *testing.T) {
		run := make(chan struct{}, 1)
		a := newDataSDKAdapter(QueryDataHandlerFunc(func(ctx context.Context, req *QueryDataRequest) (*QueryDataResponse, error) {
			checkCtxLogger(t, ctx)
			run <- struct{}{}
			return NewQueryDataResponse(), nil
		}))
		_, err := a.QueryData(context.Background(), &pluginv2.QueryDataRequest{
			PluginContext: &pluginv2.PluginContext{},
		})
		require.NoError(t, err)
		<-run
	})

	t.Run("DiagnosticsSDKAdapter", func(t *testing.T) {
		run := make(chan struct{}, 1)
		a := newDiagnosticsSDKAdapter(prometheus.DefaultGatherer, CheckHealthHandlerFunc(func(ctx context.Context, req *CheckHealthRequest) (*CheckHealthResult, error) {
			checkCtxLogger(t, ctx)
			run <- struct{}{}
			return &CheckHealthResult{}, nil
		}))
		_, err := a.CheckHealth(context.Background(), &pluginv2.CheckHealthRequest{
			PluginContext: &pluginv2.PluginContext{},
		})
		require.NoError(t, err)
		<-run
	})

	t.Run("ResourceSDKAdapter", func(t *testing.T) {
		run := make(chan struct{}, 1)
		a := newResourceSDKAdapter(CallResourceHandlerFunc(func(ctx context.Context, req *CallResourceRequest, sender CallResourceResponseSender) error {
			checkCtxLogger(t, ctx)
			run <- struct{}{}
			return nil
		}))
		err := a.CallResource(&pluginv2.CallResourceRequest{
			PluginContext: &pluginv2.PluginContext{},
		}, newTestCallResourceServer())
		require.NoError(t, err)
		<-run
	})

	t.Run("StreamHandler", func(t *testing.T) {
		subscribeStreamRun := make(chan struct{}, 1)
		publishStreamRun := make(chan struct{}, 1)
		runStreamRun := make(chan struct{}, 1)
		a := newStreamSDKAdapter(&streamAdapter{
			subscribeStreamFunc: func(ctx context.Context, request *SubscribeStreamRequest) (*SubscribeStreamResponse, error) {
				checkCtxLogger(t, ctx)
				subscribeStreamRun <- struct{}{}
				return &SubscribeStreamResponse{}, nil
			},
			publishStreamFunc: func(ctx context.Context, request *PublishStreamRequest) (*PublishStreamResponse, error) {
				checkCtxLogger(t, ctx)
				publishStreamRun <- struct{}{}
				return &PublishStreamResponse{}, nil
			},
			runStreamFunc: func(ctx context.Context, request *RunStreamRequest, sender *StreamSender) error {
				checkCtxLogger(t, ctx)
				runStreamRun <- struct{}{}
				return nil
			},
		})

		t.Run("SubscribeStream", func(t *testing.T) {
			_, err := a.SubscribeStream(context.Background(), &pluginv2.SubscribeStreamRequest{
				PluginContext: &pluginv2.PluginContext{},
			})
			require.NoError(t, err)
			<-subscribeStreamRun
		})

		t.Run("PublishStream", func(t *testing.T) {
			_, err := a.PublishStream(context.Background(), &pluginv2.PublishStreamRequest{
				PluginContext: &pluginv2.PluginContext{},
			})
			require.NoError(t, err)
			<-publishStreamRun
		})

		t.Run("RunStream", func(t *testing.T) {
			err := a.RunStream(&pluginv2.RunStreamRequest{
				PluginContext: &pluginv2.PluginContext{},
			}, newTestRunStreamServer())
			require.NoError(t, err)
			<-runStreamRun
		})
	})
}
