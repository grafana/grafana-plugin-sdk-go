package backend

import (
	"context"
	"strconv"

	bproto "github.com/grafana/grafana-plugin-sdk-go/genproto/go/backend_plugin"
	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type CoreGRPCClient struct {
	broker *plugin.GRPCBroker
	client bproto.CoreClient
}

// Plugin is the Grafana Backend plugin interface.
// It corresponds to: grafana.plugin protobuf: BackendPlugin Service | genproto/go/grafana_plugin: BackendPluginClient interface
type Plugin interface {
	Check(ctx context.Context, req *bproto.PluginStatusRequest) (*bproto.PluginStatusResponse, error)
	DataQuery(ctx context.Context, req *bproto.DataQueryRequest) (*bproto.DataQueryResponse, error)
	Resource(ctx context.Context, req *bproto.ResourceRequest) (*bproto.ResourceResponse, error)
}

type coreGRPCServer struct {
	broker *plugin.GRPCBroker
	Impl   coreWrapper
}

func (m *CoreGRPCClient) DataQuery(ctx context.Context, req *bproto.DataQueryRequest, api *PlatformAPI) (*bproto.DataQueryResponse, error) {
	if *api == nil {
		return m.client.DataQuery(ctx, req)
	}
	// if callback
	apiServer := &PlatformGrpcApiServer{*api}
	var s *grpc.Server
	serverFunc := func(opts []grpc.ServerOption) *grpc.Server {
		s = grpc.NewServer(opts...)
		bproto.RegisterGrafanaPlatformServer(s, apiServer)

		return s
	}
	brokeID := m.broker.NextId()
	go m.broker.AcceptAndServe(brokeID, serverFunc)

	metadata.AppendToOutgoingContext(ctx, "broker_requestId", string(brokeID))
	res, err := m.client.DataQuery(ctx, req)

	s.Stop()
	return res, err
}

func (m *coreGRPCServer) DataQuery(ctx context.Context, req *bproto.DataQueryRequest) (*bproto.DataQueryResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return m.Impl.DataQuery(ctx, req, nil)
	}
	rawReqIDValues := md.Get("broker_requestId") // TODO const
	if len(rawReqIDValues) != 1 {
		return m.Impl.DataQuery(ctx, req, nil)
	}
	id64, err := strconv.ParseUint(rawReqIDValues[0], 10, 32)
	if err != nil {
		return nil, err
	}
	conn, err := m.broker.Dial(uint32(id64))
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	api := &PlatformGrpcApiClient{bproto.NewGrafanaPlatformClient(conn)}
	return m.Impl.DataQuery(ctx, req, api)
}

func (m *CoreGRPCClient) Check(ctx context.Context, req *bproto.PluginStatusRequest) (*bproto.PluginStatusResponse, error) {
	return m.client.Check(ctx, req)
}

func (m *coreGRPCServer) Check(ctx context.Context, req *bproto.PluginStatusRequest) (*bproto.PluginStatusResponse, error) {
	return m.Impl.Check(ctx, req)
}

func (m *CoreGRPCClient) Resource(ctx context.Context, req *bproto.ResourceRequest) (*bproto.ResourceResponse, error) {
	return m.client.Resource(ctx, req)
}

func (m *coreGRPCServer) Resource(ctx context.Context, req *bproto.ResourceRequest) (*bproto.ResourceResponse, error) {
	return m.Impl.Resource(ctx, req)
}
