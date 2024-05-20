package automanagement

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Manager is a helper to simplify instance management for plugin
// developers. It gets instancemgmt.InstanceManager on every call thus making
// sure datasource instance disposed on configuration change and new datasource
// instance created.
type Manager struct {
	instancemgmt.InstanceManager

	// For create requests, the storage engine will not cache an instance
	storage backend.StorageHandler
}

var (
	_ = backend.CollectMetricsHandler(&Manager{})
	_ = backend.CheckHealthHandler(&Manager{})
	_ = backend.QueryDataHandler(&Manager{})
	_ = backend.CallResourceHandler(&Manager{})
	_ = backend.StreamHandler(&Manager{})
	_ = backend.StorageHandler(&Manager{})
)

// NewManager creates Manager. It accepts datasource instance factory.
func NewManager(instanceManager instancemgmt.InstanceManager, storage backend.StorageHandler) *Manager {
	return &Manager{
		InstanceManager: instanceManager,
		storage:         storage,
	}
}

func (m *Manager) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	h, err := m.Get(ctx, req.PluginContext)
	if err != nil {
		return nil, err
	}
	if ds, ok := h.(backend.QueryDataHandler); ok {
		return ds.QueryData(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (m *Manager) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	h, err := m.Get(ctx, req.PluginContext)
	if err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: err.Error(),
		}, nil
	}
	if ds, ok := h.(backend.CheckHealthHandler); ok {
		return ds.CheckHealth(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (m *Manager) CollectMetrics(ctx context.Context, req *backend.CollectMetricsRequest) (*backend.CollectMetricsResult, error) {
	h, err := m.Get(ctx, req.PluginContext)
	if err != nil {
		return nil, err
	}
	if ds, ok := h.(backend.CollectMetricsHandler); ok {
		return ds.CollectMetrics(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (m *Manager) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	h, err := m.Get(ctx, req.PluginContext)
	if err != nil {
		return err
	}
	if ds, ok := h.(backend.CallResourceHandler); ok {
		return ds.CallResource(ctx, req, sender)
	}
	return status.Error(codes.Unimplemented, "unimplemented")
}

func (m *Manager) SubscribeStream(ctx context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	h, err := m.Get(ctx, req.PluginContext)
	if err != nil {
		return nil, err
	}
	if ds, ok := h.(backend.StreamHandler); ok {
		return ds.SubscribeStream(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (m *Manager) PublishStream(ctx context.Context, req *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
	h, err := m.Get(ctx, req.PluginContext)
	if err != nil {
		return nil, err
	}
	if ds, ok := h.(backend.StreamHandler); ok {
		return ds.PublishStream(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (m *Manager) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	h, err := m.Get(ctx, req.PluginContext)
	if err != nil {
		return err
	}
	if ds, ok := h.(backend.StreamHandler); ok {
		return ds.RunStream(ctx, req, sender)
	}
	return status.Error(codes.Unimplemented, "unimplemented")
}

func (m *Manager) MutateInstanceSettings(ctx context.Context, req *backend.InstanceSettingsAdmissionRequest) (*backend.InstanceSettingsResponse, error) {
	if m.storage == nil {
		return nil, status.Error(codes.Unimplemented, "unimplemented")
	}
	return m.storage.MutateInstanceSettings(ctx, req)
}

func (m *Manager) ValidateAdmission(ctx context.Context, req *backend.AdmissionRequest) (*backend.StorageResponse, error) {
	if m.storage == nil {
		return nil, status.Error(codes.Unimplemented, "unimplemented")
	}
	return m.storage.ValidateAdmission(ctx, req)
}

func (m *Manager) MutateAdmission(ctx context.Context, req *backend.AdmissionRequest) (*backend.StorageResponse, error) {
	if m.storage == nil {
		return nil, status.Error(codes.Unimplemented, "unimplemented")
	}
	return m.storage.ValidateAdmission(ctx, req)
}

func (m *Manager) ConvertObject(ctx context.Context, req *backend.ConversionRequest) (*backend.StorageResponse, error) {
	if m.storage == nil {
		return nil, status.Error(codes.Unimplemented, "unimplemented")
	}
	return m.storage.ConvertObject(ctx, req)
}
