package datasource

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type instanceManager struct {
	instancemgmt.InstanceManager
}

func (m *instanceManager) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	h, err := m.Get(req.PluginContext)
	if err != nil {
		return nil, err
	}
	if ds, ok := h.(backend.QueryDataHandler); ok {
		return ds.QueryData(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (m *instanceManager) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	h, err := m.Get(req.PluginContext)
	if err != nil {
		return nil, err
	}
	if ds, ok := h.(backend.CheckHealthHandler); ok {
		return ds.CheckHealth(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (m *instanceManager) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	h, err := m.Get(req.PluginContext)
	if err != nil {
		return err
	}
	if ds, ok := h.(backend.CallResourceHandler); ok {
		return ds.CallResource(ctx, req, sender)
	}
	return status.Error(codes.Unimplemented, "unimplemented")
}

func (m *instanceManager) SubscribeStream(ctx context.Context, req *backend.SubscribeStreamRequest) (*backend.SubscribeStreamResponse, error) {
	h, err := m.Get(req.PluginContext)
	if err != nil {
		return nil, err
	}
	if ds, ok := h.(backend.StreamHandler); ok {
		return ds.SubscribeStream(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (m *instanceManager) PublishStream(ctx context.Context, req *backend.PublishStreamRequest) (*backend.PublishStreamResponse, error) {
	h, err := m.Get(req.PluginContext)
	if err != nil {
		return nil, err
	}
	if ds, ok := h.(backend.StreamHandler); ok {
		return ds.PublishStream(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (m *instanceManager) RunStream(ctx context.Context, req *backend.RunStreamRequest, sender backend.StreamPacketSender) error {
	h, err := m.Get(req.PluginContext)
	if err != nil {
		return err
	}
	if ds, ok := h.(backend.StreamHandler); ok {
		return ds.RunStream(ctx, req, sender)
	}
	return status.Error(codes.Unimplemented, "unimplemented")
}

// ManageOpts can modify Manage behaviour.
type ManageOpts struct {
	// GRPCSettings settings for gPRC.
	GRPCSettings backend.GRPCSettings
}

// Manage starts serving the data source over gPRC with automatic instance management.
func Manage(factory InstanceFactoryFunc, opts ManageOpts) error {
	handler := &instanceManager{
		InstanceManager: NewInstanceManager(factory),
	}
	// TODO: do we need to ask user for explicit plugin capabilities here
	// as we don't have instance till first call?
	return backend.Serve(backend.ServeOpts{
		QueryDataHandler:    handler,
		CheckHealthHandler:  handler,
		CallResourceHandler: handler,
		StreamHandler:       handler,
		GRPCSettings:        opts.GRPCSettings,
	})
}
