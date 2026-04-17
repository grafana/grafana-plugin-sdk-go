package backend

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

func checkCtxLogger(ctx context.Context, t *testing.T, expParams map[string]any) {
	t.Helper()
	logAttrs := log.ContextualAttributesFromContext(ctx)
	if len(expParams) == 0 {
		require.Empty(t, logAttrs)
		return
	}

	require.NotEmpty(t, logAttrs)
	require.Truef(t, len(logAttrs)%2 == 0, "expected even number of log params, got %d", len(logAttrs))
	require.Equal(t, len(expParams)*2, len(logAttrs))
	for i := 0; i < len(logAttrs)/2; i++ {
		key := logAttrs[i*2].(string)
		val := logAttrs[i*2+1]
		expVal, ok := expParams[key]
		require.Truef(t, ok, "unexpected log param: %s", key)
		require.Equal(t, expVal, val)
	}
}

func TestContextualLogger(t *testing.T) {
	const pluginID = "plugin-id"
	const pluginVersion = "1.0.0"
	pCtx := &pluginv2.PluginContext{PluginId: pluginID, PluginVersion: pluginVersion}
	t.Run("DataSDKAdapter", func(t *testing.T) {
		run := make(chan struct{}, 1)
		handler := QueryDataHandlerFunc(func(ctx context.Context, _ *QueryDataRequest) (*QueryDataResponse, error) {
			checkCtxLogger(ctx, t, map[string]any{"endpoint": "queryData", "pluginId": pluginID, "pluginVersion": pluginVersion})
			run <- struct{}{}
			return NewQueryDataResponse(), nil
		})
		handlers := Handlers{
			QueryDataHandler: handler,
		}
		handlerWithMw, err := HandlerFromMiddlewares(handlers, newTenantIDMiddleware(), newContextualLoggerMiddleware())
		require.NoError(t, err)
		a := newDataSDKAdapter(handlerWithMw, handlerWithMw)
		_, err = a.QueryData(context.Background(), &pluginv2.QueryDataRequest{
			PluginContext: pCtx,
		})
		require.NoError(t, err)
		<-run
	})

	t.Run("DiagnosticsSDKAdapter", func(t *testing.T) {
		run := make(chan struct{}, 1)
		handler := CheckHealthHandlerFunc(func(ctx context.Context, _ *CheckHealthRequest) (*CheckHealthResult, error) {
			checkCtxLogger(ctx, t, map[string]any{"endpoint": "checkHealth", "pluginId": pluginID, "pluginVersion": pluginVersion})
			run <- struct{}{}
			return &CheckHealthResult{}, nil
		})
		handlers := Handlers{
			CheckHealthHandler: handler,
		}
		handlerWithMw, err := HandlerFromMiddlewares(handlers, newTenantIDMiddleware(), newContextualLoggerMiddleware())
		require.NoError(t, err)
		a := newDiagnosticsSDKAdapter(prometheus.DefaultGatherer, handlerWithMw)
		_, err = a.CheckHealth(context.Background(), &pluginv2.CheckHealthRequest{
			PluginContext: pCtx,
		})
		require.NoError(t, err)
		<-run
	})

	t.Run("ResourceSDKAdapter", func(t *testing.T) {
		run := make(chan struct{}, 1)
		handler := CallResourceHandlerFunc(func(ctx context.Context, _ *CallResourceRequest, _ CallResourceResponseSender) error {
			checkCtxLogger(ctx, t, map[string]any{"endpoint": "callResource", "pluginId": pluginID, "pluginVersion": pluginVersion})
			run <- struct{}{}
			return nil
		})
		handlers := Handlers{
			CallResourceHandler: handler,
		}
		handlerWithMw, err := HandlerFromMiddlewares(handlers, newTenantIDMiddleware(), newContextualLoggerMiddleware())
		require.NoError(t, err)
		a := newResourceSDKAdapter(handlerWithMw)
		err = a.CallResource(&pluginv2.CallResourceRequest{
			PluginContext: pCtx,
		}, newTestCallResourceServer())
		require.NoError(t, err)
		<-run
	})

	t.Run("StreamHandler", func(t *testing.T) {
		subscribeStreamRun := make(chan struct{}, 1)
		publishStreamRun := make(chan struct{}, 1)
		runStreamRun := make(chan struct{}, 1)
		handlers := Handlers{
			StreamHandler: &streamAdapter{
				subscribeStreamFunc: func(ctx context.Context, _ *SubscribeStreamRequest) (*SubscribeStreamResponse, error) {
					checkCtxLogger(ctx, t, map[string]any{"endpoint": "subscribeStream", "pluginId": pluginID, "pluginVersion": pluginVersion})
					subscribeStreamRun <- struct{}{}
					return &SubscribeStreamResponse{}, nil
				},
				publishStreamFunc: func(ctx context.Context, _ *PublishStreamRequest) (*PublishStreamResponse, error) {
					checkCtxLogger(ctx, t, map[string]any{"endpoint": "publishStream", "pluginId": pluginID, "pluginVersion": pluginVersion})
					publishStreamRun <- struct{}{}
					return &PublishStreamResponse{}, nil
				},
				runStreamFunc: func(ctx context.Context, _ *RunStreamRequest, _ *StreamSender) error {
					checkCtxLogger(ctx, t, map[string]any{"endpoint": "runStream", "pluginId": pluginID, "pluginVersion": pluginVersion})
					runStreamRun <- struct{}{}
					return nil
				},
			},
		}
		handlerWithMw, err := HandlerFromMiddlewares(handlers, newTenantIDMiddleware(), newContextualLoggerMiddleware())
		require.NoError(t, err)
		a := newStreamSDKAdapter(handlerWithMw)

		t.Run("SubscribeStream", func(t *testing.T) {
			_, err := a.SubscribeStream(context.Background(), &pluginv2.SubscribeStreamRequest{
				PluginContext: pCtx,
			})
			require.NoError(t, err)
			<-subscribeStreamRun
		})

		t.Run("PublishStream", func(t *testing.T) {
			_, err := a.PublishStream(context.Background(), &pluginv2.PublishStreamRequest{
				PluginContext: pCtx,
			})
			require.NoError(t, err)
			<-publishStreamRun
		})

		t.Run("RunStream", func(t *testing.T) {
			err := a.RunStream(&pluginv2.RunStreamRequest{
				PluginContext: pCtx,
			}, newTestRunStreamServer())
			require.NoError(t, err)
			<-runStreamRun
		})
	})
}
