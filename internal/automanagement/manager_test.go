package automanagement

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/stretchr/testify/require"
)

type lifetimeHandler struct {
	entered  chan struct{}
	resume   chan struct{}
	disposed atomic.Bool
}

func (h *lifetimeHandler) wait() error {
	close(h.entered)
	<-h.resume
	if h.disposed.Load() {
		return errors.New("handler used a disposed instance")
	}
	return nil
}
func (h *lifetimeHandler) Dispose() { h.disposed.Store(true) }
func (h *lifetimeHandler) QueryData(context.Context, *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	return nil, h.wait()
}
func (h *lifetimeHandler) QueryChunkedData(context.Context, *backend.QueryChunkedDataRequest, backend.ChunkedDataWriter) error {
	return h.wait()
}
func (h *lifetimeHandler) CheckHealth(context.Context, *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	return nil, h.wait()
}
func (h *lifetimeHandler) CallResource(context.Context, *backend.CallResourceRequest, backend.CallResourceResponseSender) error {
	return h.wait()
}
func (h *lifetimeHandler) SubscribeStream(context.Context, *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	return nil, h.wait()
}
func (h *lifetimeHandler) PublishStream(context.Context, *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
	return nil, h.wait()
}
func (h *lifetimeHandler) RunStream(context.Context, *backend.RunStreamRequest, *backend.StreamSender) error {
	return h.wait()
}

type lifetimeProvider struct{ handlers []*lifetimeHandler }

func (*lifetimeProvider) GetKey(context.Context, backend.PluginContext) (any, error) {
	return "key", nil
}
func (*lifetimeProvider) NeedsUpdate(_ context.Context, pc backend.PluginContext, cached instancemgmt.CachedInstance) bool {
	return !pc.DataSourceInstanceSettings.Updated.Equal(cached.PluginContext.DataSourceInstanceSettings.Updated)
}
func (p *lifetimeProvider) NewInstance(context.Context, backend.PluginContext) (instancemgmt.Instance, error) {
	h := &lifetimeHandler{entered: make(chan struct{}), resume: make(chan struct{})}
	p.handlers = append(p.handlers, h)
	return h, nil
}

func TestHandlersKeepAcquiredInstanceAlive(t *testing.T) {
	calls := map[string]func(context.Context, *Manager, backend.PluginContext) error{
		"QueryData": func(ctx context.Context, m *Manager, pc backend.PluginContext) error {
			_, err := m.QueryData(ctx, &backend.QueryDataRequest{PluginContext: pc})
			return err
		},
		"QueryChunkedData": func(ctx context.Context, m *Manager, pc backend.PluginContext) error {
			return m.QueryChunkedData(ctx, &backend.QueryChunkedDataRequest{PluginContext: pc}, nil)
		},
		"CheckHealth": func(ctx context.Context, m *Manager, pc backend.PluginContext) error {
			_, err := m.CheckHealth(ctx, &backend.CheckHealthRequest{PluginContext: pc})
			return err
		},
		"CallResource": func(ctx context.Context, m *Manager, pc backend.PluginContext) error {
			return m.CallResource(ctx, &backend.CallResourceRequest{PluginContext: pc}, nil)
		},
		"SubscribeStream": func(ctx context.Context, m *Manager, pc backend.PluginContext) error {
			_, err := m.SubscribeStream(ctx, &backend.SubscribeStreamRequest{PluginContext: pc})
			return err
		},
		"PublishStream": func(ctx context.Context, m *Manager, pc backend.PluginContext) error {
			_, err := m.PublishStream(ctx, &backend.PublishStreamRequest{PluginContext: pc})
			return err
		},
		"RunStream": func(ctx context.Context, m *Manager, pc backend.PluginContext) error {
			return m.RunStream(ctx, &backend.RunStreamRequest{PluginContext: pc}, nil)
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			provider := &lifetimeProvider{}
			instances := instancemgmt.NewTTLInstanceManager(provider)
			m := NewManager(instances)
			pc := backend.PluginContext{DataSourceInstanceSettings: &backend.DataSourceInstanceSettings{Updated: time.Now()}}
			first, err := instances.Get(context.Background(), pc)
			require.NoError(t, err)
			old := first.(*lifetimeHandler)
			var once sync.Once
			resume := func() { once.Do(func() { close(old.resume) }) }
			t.Cleanup(resume)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- call(ctx, m, pc) }()
			select {
			case <-old.entered:
			case <-time.After(time.Second):
				t.Fatal("handler did not start")
			}
			cancel() // Cancellation is not proof the handler has finished.
			updated := pc
			updated.DataSourceInstanceSettings = &backend.DataSourceInstanceSettings{Updated: pc.DataSourceInstanceSettings.Updated.Add(time.Second)}
			_, err = instances.Get(context.Background(), updated)
			require.NoError(t, err)
			require.False(t, old.disposed.Load())
			resume()
			select {
			case err := <-done:
				require.NoError(t, err)
			case <-time.After(time.Second):
				t.Fatal("handler did not finish")
			}
			require.True(t, old.disposed.Load())
		})
	}
}
