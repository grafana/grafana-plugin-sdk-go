package automanagement

import (
	"context"
	"errors"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Manager creates plugin instances on demand and keeps them alive for the full
// handler call when its instance manager supports acquisition.
type Manager struct {
	instancemgmt.InstanceManager
}

// NewManager creates Manager. It accepts a datasource instance manager.
func NewManager(instanceManager instancemgmt.InstanceManager) *Manager {
	return &Manager{InstanceManager: instanceManager}
}

func (m *Manager) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	h, release, err := instancemgmt.Acquire(ctx, m.InstanceManager, req.PluginContext)
	if err != nil {
		if len(req.Queries) == 0 {
			return nil, err
		}
		var esErr backend.ErrorWithSource
		if !errors.As(err, &esErr) {
			return nil, err
		}
		resp := backend.NewQueryDataResponse()
		resp.Responses[req.Queries[0].RefID] = backend.DataResponse{
			Error:       err,
			ErrorSource: esErr.ErrorSource(),
		}
		return resp, nil
	}
	defer release()
	if ds, ok := h.(backend.QueryDataHandler); ok {
		return ds.QueryData(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (m *Manager) QueryChunkedData(ctx context.Context, req *backend.QueryChunkedDataRequest, w backend.ChunkedDataWriter) error {
	h, release, err := instancemgmt.Acquire(ctx, m.InstanceManager, req.PluginContext)
	if err != nil {
		return err
	}
	defer release()
	if ds, ok := h.(backend.QueryChunkedDataHandler); ok {
		return ds.QueryChunkedData(ctx, req, w)
	}
	return status.Error(codes.Unimplemented, "unimplemented")
}

func (m *Manager) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	h, release, err := instancemgmt.Acquire(ctx, m.InstanceManager, req.PluginContext)
	if err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: err.Error(),
		}, nil
	}
	defer release()
	if ds, ok := h.(backend.CheckHealthHandler); ok {
		return ds.CheckHealth(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (m *Manager) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	h, release, err := instancemgmt.Acquire(ctx, m.InstanceManager, req.PluginContext)
	if err != nil {
		return err
	}
	defer release()
	if ds, ok := h.(backend.CallResourceHandler); ok {
		return ds.CallResource(ctx, req, sender)
	}
	return status.Error(codes.Unimplemented, "unimplemented")
}

func (m *Manager) SubscribeStream(ctx context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	h, release, err := instancemgmt.Acquire(ctx, m.InstanceManager, req.PluginContext)
	if err != nil {
		return nil, err
	}
	defer release()
	if ds, ok := h.(backend.StreamHandler); ok {
		return ds.SubscribeStream(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (m *Manager) PublishStream(ctx context.Context, req *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
	h, release, err := instancemgmt.Acquire(ctx, m.InstanceManager, req.PluginContext)
	if err != nil {
		return nil, err
	}
	defer release()
	if ds, ok := h.(backend.StreamHandler); ok {
		return ds.PublishStream(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (m *Manager) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender *backend.StreamSender) error {
	h, release, err := instancemgmt.Acquire(ctx, m.InstanceManager, req.PluginContext)
	if err != nil {
		return err
	}
	defer release()
	if ds, ok := h.(backend.StreamHandler); ok {
		return ds.RunStream(ctx, req, sender)
	}
	return status.Error(codes.Unimplemented, "unimplemented")
}
