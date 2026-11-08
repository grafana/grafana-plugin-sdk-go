package backend_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/handlertest"
)

func TestHandlerFromMiddlewares(t *testing.T) {
	var queryDataCalled bool
	var callResourceCalled bool
	var checkHealthCalled bool
	var collectMetricsCalled bool
	var subscribeStreamCalled bool
	var publishStreamCalled bool
	var runStreamCalled bool

	c := &handlertest.Handler{
		QueryDataFunc: func(_ context.Context, _ *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
			queryDataCalled = true
			return nil, nil
		},
		CallResourceFunc: func(_ context.Context, _ *backend.CallResourceRequest, _ backend.CallResourceResponseSender) error {
			callResourceCalled = true
			return nil
		},
		CheckHealthFunc: func(_ context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
			checkHealthCalled = true
			return nil, nil
		},
		CollectMetricsFunc: func(_ context.Context, _ *backend.CollectMetricsRequest) (*backend.CollectMetricsResult, error) {
			collectMetricsCalled = true
			return nil, nil
		},
		SubscribeStreamFunc: func(_ context.Context, _ *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
			subscribeStreamCalled = true
			return nil, nil
		},
		PublishStreamFunc: func(_ context.Context, _ *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
			publishStreamCalled = true
			return nil, nil
		},
		RunStreamFunc: func(_ context.Context, _ *backend.RunStreamRequest, _ *backend.StreamSender) error {
			runStreamCalled = true
			return nil
		},
	}

	require.NotNil(t, c)

	ctx := MiddlewareScenarioContext{}

	mwOne := ctx.NewMiddleware("mw1")
	mwTwo := ctx.NewMiddleware("mw2")

	d, err := backend.HandlerFromMiddlewares(c, mwOne, mwTwo)
	require.NoError(t, err)
	require.NotNil(t, d)

	_, _ = d.QueryData(context.Background(), &backend.QueryDataRequest{})
	require.True(t, queryDataCalled)

	sender := backend.CallResourceResponseSenderFunc(func(_ *backend.CallResourceResponse) error {
		return nil
	})

	_ = d.CallResource(context.Background(), &backend.CallResourceRequest{}, sender)
	require.True(t, callResourceCalled)

	_, _ = d.CheckHealth(context.Background(), &backend.CheckHealthRequest{})
	require.True(t, checkHealthCalled)

	_, _ = d.CollectMetrics(context.Background(), &backend.CollectMetricsRequest{})
	require.True(t, collectMetricsCalled)

	_, _ = d.SubscribeStream(context.Background(), &backend.SubscribeStreamRequest{})
	require.True(t, subscribeStreamCalled)

	_, _ = d.PublishStream(context.Background(), &backend.PublishStreamRequest{})
	require.True(t, publishStreamCalled)

	streamSender := backend.NewStreamSender(nil)
	_ = d.RunStream(context.Background(), &backend.RunStreamRequest{}, streamSender)
	require.True(t, runStreamCalled)

	require.Len(t, ctx.QueryDataCallChain, 4)
	require.EqualValues(t, []string{"before mw1", "before mw2", "after mw2", "after mw1"}, ctx.QueryDataCallChain)
	require.Len(t, ctx.CallResourceCallChain, 4)
	require.EqualValues(t, []string{"before mw1", "before mw2", "after mw2", "after mw1"}, ctx.CallResourceCallChain)
	require.Len(t, ctx.CheckHealthCallChain, 4)
	require.EqualValues(t, []string{"before mw1", "before mw2", "after mw2", "after mw1"}, ctx.CheckHealthCallChain)
	require.Len(t, ctx.CollectMetricsCallChain, 4)
	require.EqualValues(t, []string{"before mw1", "before mw2", "after mw2", "after mw1"}, ctx.CollectMetricsCallChain)
	require.Len(t, ctx.SubscribeStreamCallChain, 4)
	require.EqualValues(t, []string{"before mw1", "before mw2", "after mw2", "after mw1"}, ctx.SubscribeStreamCallChain)
	require.Len(t, ctx.PublishStreamCallChain, 4)
	require.EqualValues(t, []string{"before mw1", "before mw2", "after mw2", "after mw1"}, ctx.PublishStreamCallChain)
	require.Len(t, ctx.RunStreamCallChain, 4)
	require.EqualValues(t, []string{"before mw1", "before mw2", "after mw2", "after mw1"}, ctx.RunStreamCallChain)
}

type MiddlewareScenarioContext struct {
	QueryDataCallChain        []string
	QueryChunkedDataCallChain []string
	CallResourceCallChain     []string
	CollectMetricsCallChain   []string
	CheckHealthCallChain      []string
	SubscribeStreamCallChain  []string
	PublishStreamCallChain    []string
	RunStreamCallChain        []string
	InstanceSettingsCallChain []string
}

func (ctx *MiddlewareScenarioContext) NewMiddleware(name string) backend.HandlerMiddleware {
	return backend.HandlerMiddlewareFunc(func(next backend.Handler) backend.Handler {
		return &TestMiddleware{
			next: next,
			Name: name,
			sCtx: ctx,
		}
	})
}

type TestMiddleware struct {
	next backend.Handler
	sCtx *MiddlewareScenarioContext
	Name string
}

func (m *TestMiddleware) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	m.sCtx.QueryDataCallChain = append(m.sCtx.QueryDataCallChain, fmt.Sprintf("before %s", m.Name))
	res, err := m.next.QueryData(ctx, req)
	m.sCtx.QueryDataCallChain = append(m.sCtx.QueryDataCallChain, fmt.Sprintf("after %s", m.Name))
	return res, err
}

func (m *TestMiddleware) QueryChunkedData(ctx context.Context, req *backend.QueryChunkedDataRequest, w backend.ChunkedDataWriter) error {
	m.sCtx.QueryChunkedDataCallChain = append(m.sCtx.QueryChunkedDataCallChain, fmt.Sprintf("before %s", m.Name))
	err := m.next.QueryChunkedData(ctx, req, w)
	m.sCtx.QueryChunkedDataCallChain = append(m.sCtx.QueryChunkedDataCallChain, fmt.Sprintf("after %s", m.Name))
	return err
}

func (m *TestMiddleware) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	m.sCtx.CallResourceCallChain = append(m.sCtx.CallResourceCallChain, fmt.Sprintf("before %s", m.Name))
	err := m.next.CallResource(ctx, req, sender)
	m.sCtx.CallResourceCallChain = append(m.sCtx.CallResourceCallChain, fmt.Sprintf("after %s", m.Name))
	return err
}

func (m *TestMiddleware) CollectMetrics(ctx context.Context, req *backend.CollectMetricsRequest) (*backend.CollectMetricsResult, error) {
	m.sCtx.CollectMetricsCallChain = append(m.sCtx.CollectMetricsCallChain, fmt.Sprintf("before %s", m.Name))
	res, err := m.next.CollectMetrics(ctx, req)
	m.sCtx.CollectMetricsCallChain = append(m.sCtx.CollectMetricsCallChain, fmt.Sprintf("after %s", m.Name))
	return res, err
}

func (m *TestMiddleware) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	m.sCtx.CheckHealthCallChain = append(m.sCtx.CheckHealthCallChain, fmt.Sprintf("before %s", m.Name))
	res, err := m.next.CheckHealth(ctx, req)
	m.sCtx.CheckHealthCallChain = append(m.sCtx.CheckHealthCallChain, fmt.Sprintf("after %s", m.Name))
	return res, err
}

func (m *TestMiddleware) SubscribeStream(ctx context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	m.sCtx.SubscribeStreamCallChain = append(m.sCtx.SubscribeStreamCallChain, fmt.Sprintf("before %s", m.Name))
	res, err := m.next.SubscribeStream(ctx, req)
	m.sCtx.SubscribeStreamCallChain = append(m.sCtx.SubscribeStreamCallChain, fmt.Sprintf("after %s", m.Name))
	return res, err
}

func (m *TestMiddleware) PublishStream(ctx context.Context, req *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
	m.sCtx.PublishStreamCallChain = append(m.sCtx.PublishStreamCallChain, fmt.Sprintf("before %s", m.Name))
	res, err := m.next.PublishStream(ctx, req)
	m.sCtx.PublishStreamCallChain = append(m.sCtx.PublishStreamCallChain, fmt.Sprintf("after %s", m.Name))
	return res, err
}

func (m *TestMiddleware) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	m.sCtx.RunStreamCallChain = append(m.sCtx.RunStreamCallChain, fmt.Sprintf("before %s", m.Name))
	err := m.next.RunStream(ctx, req, sender)
	m.sCtx.RunStreamCallChain = append(m.sCtx.RunStreamCallChain, fmt.Sprintf("after %s", m.Name))
	return err
}
