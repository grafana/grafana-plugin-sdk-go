package datasource

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AutoInstanceManager struct {
	instancemgmt.InstanceManager
}

func NewAutoInstanceManager(factoryFunc InstanceFactoryFunc) *AutoInstanceManager {
	return &AutoInstanceManager{InstanceManager: NewInstanceManager(factoryFunc)}
}

func (m *AutoInstanceManager) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	h, err := m.Get(req.PluginContext)
	if err != nil {
		return nil, err
	}
	if ds, ok := h.(backend.QueryDataHandler); ok {
		return ds.QueryData(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (m *AutoInstanceManager) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	h, err := m.Get(req.PluginContext)
	if err != nil {
		return nil, err
	}
	if ds, ok := h.(backend.CheckHealthHandler); ok {
		return ds.CheckHealth(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (m *AutoInstanceManager) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	h, err := m.Get(req.PluginContext)
	if err != nil {
		return err
	}
	if ds, ok := h.(backend.CallResourceHandler); ok {
		return ds.CallResource(ctx, req, sender)
	}
	return status.Error(codes.Unimplemented, "unimplemented")
}

func (m *AutoInstanceManager) SubscribeStream(ctx context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	h, err := m.Get(req.PluginContext)
	if err != nil {
		return nil, err
	}
	if ds, ok := h.(backend.StreamHandler); ok {
		return ds.SubscribeStream(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (m *AutoInstanceManager) PublishStream(ctx context.Context, req *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
	h, err := m.Get(req.PluginContext)
	if err != nil {
		return nil, err
	}
	if ds, ok := h.(backend.StreamHandler); ok {
		return ds.PublishStream(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (m *AutoInstanceManager) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender backend.StreamPacketSender) error {
	h, err := m.Get(req.PluginContext)
	if err != nil {
		return err
	}
	if ds, ok := h.(backend.StreamHandler); ok {
		return ds.RunStream(ctx, req, sender)
	}
	return status.Error(codes.Unimplemented, "unimplemented")
}
