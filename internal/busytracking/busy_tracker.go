package busytracking

import (
	"context"
	"sync/atomic"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
)

// NewManager wraps an InstanceManager with automatic busy tracking
// for all instances. This prevents disposal of instances while they're actively
// processing requests.
func NewManager(manager instancemgmt.InstanceManager) instancemgmt.InstanceManager {
	return &busyTrackingInstanceManager{manager}
}

// busyTrackingInstanceManager automatically wraps all instances with busy tracking
type busyTrackingInstanceManager struct {
	instancemgmt.InstanceManager
}

// Get wraps the returned instance with automatic busy tracking
func (w *busyTrackingInstanceManager) Get(ctx context.Context, pluginContext backend.PluginContext) (instancemgmt.Instance, error) {
	instance, err := w.InstanceManager.Get(ctx, pluginContext)
	if err != nil {
		return nil, err
	}

	return &busyTrackingWrapper{Instance: instance}, nil
}

// busyTrackingWrapper automatically wraps any instance with busy tracking
type busyTrackingWrapper struct {
	instancemgmt.Instance
	activeRequests atomic.Int32
}

// SetBusy marks this instance as busy or idle
func (w *busyTrackingWrapper) SetBusy(busy bool) {
	if busy {
		w.activeRequests.Add(1)
	} else {
		w.activeRequests.Add(-1)
	}
}

// Busy returns true if this instance is currently busy
func (w *busyTrackingWrapper) Busy() bool {
	return w.activeRequests.Load() > 0
}

// Dispose delegates to the wrapped instance if it supports disposal
func (w *busyTrackingWrapper) Dispose() {
	if disposer, ok := w.Instance.(instancemgmt.InstanceDisposer); ok {
		disposer.Dispose()
	}
}

// QueryData delegates to the wrapped instance with busy tracking
func (w *busyTrackingWrapper) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	if handler, ok := w.Instance.(backend.QueryDataHandler); ok {
		w.SetBusy(true)
		defer w.SetBusy(false)
		return handler.QueryData(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "QueryData not implemented")
}

// CheckHealth delegates to the wrapped instance with busy tracking
func (w *busyTrackingWrapper) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	if handler, ok := w.Instance.(backend.CheckHealthHandler); ok {
		w.SetBusy(true)
		defer w.SetBusy(false)
		return handler.CheckHealth(ctx, req)
	}
	return &backend.CheckHealthResult{Status: backend.HealthStatusUnknown, Message: "CheckHealth not implemented"}, nil
}

// CallResource delegates to the wrapped instance with busy tracking
func (w *busyTrackingWrapper) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	if handler, ok := w.Instance.(backend.CallResourceHandler); ok {
		w.SetBusy(true)
		defer w.SetBusy(false)
		return handler.CallResource(ctx, req, sender)
	}
	return status.Error(codes.Unimplemented, "CallResource not implemented")
}

// SubscribeStream delegates to the wrapped instance with busy tracking
func (w *busyTrackingWrapper) SubscribeStream(ctx context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	if handler, ok := w.Instance.(backend.StreamHandler); ok {
		w.SetBusy(true)
		defer w.SetBusy(false)
		return handler.SubscribeStream(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "SubscribeStream not implemented")
}

// PublishStream delegates to the wrapped instance with busy tracking
func (w *busyTrackingWrapper) PublishStream(ctx context.Context, req *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
	if handler, ok := w.Instance.(backend.StreamHandler); ok {
		w.SetBusy(true)
		defer w.SetBusy(false)
		return handler.PublishStream(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "PublishStream not implemented")
}

// RunStream delegates to the wrapped instance with busy tracking
func (w *busyTrackingWrapper) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	if handler, ok := w.Instance.(backend.StreamHandler); ok {
		w.SetBusy(true)
		defer w.SetBusy(false)
		return handler.RunStream(ctx, req, sender)
	}
	return status.Error(codes.Unimplemented, "RunStream not implemented")
}
