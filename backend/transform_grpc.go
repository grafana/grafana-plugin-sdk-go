package backend

import (
	"context"
	"fmt"
	"strconv"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	plugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type transformGRPCServer struct {
	backend   backendWrapper
	transform transformWrapper
	broker    *plugin.GRPCBroker
}

func (s *transformGRPCServer) GetSchema(ctx context.Context, req *pluginv2.GetSchema_Request) (*pluginv2.GetSchema_Response, error) {
	return nil, nil
}

func (s *transformGRPCServer) ValidatePluginConfig(ctx context.Context, req *pluginv2.ValidatePluginConfig_Request) (*pluginv2.ValidatePluginConfig_Response, error) {
	return nil, nil
}

func (s *transformGRPCServer) Configure(ctx context.Context, req *pluginv2.Configure_Request) (*pluginv2.Configure_Response, error) {
	return nil, nil
}

func (s *transformGRPCServer) CallResource(ctx context.Context, req *pluginv2.CallResource_Request) (*pluginv2.CallResource_Response, error) {
	return nil, nil
}

func (t *transformGRPCServer) QueryData(ctx context.Context, req *pluginv2.QueryData_Request) (*pluginv2.QueryData_Response, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("transform request is missing metadata")
	}
	rawReqIDValues := md.Get("broker_requestId") // TODO const
	if len(rawReqIDValues) != 1 {
		return nil, fmt.Errorf("transform request metadta is missing broker_requestId")
	}
	id64, err := strconv.ParseUint(rawReqIDValues[0], 10, 32)
	if err != nil {
		return nil, err
	}
	conn, err := t.broker.Dial(uint32(id64))
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	api := &TransformCallBackGrpcClient{pluginv2.NewTransformCallBackClient(conn)}
	return t.transform.QueryData(ctx, req, api)
}

func (s *transformGRPCServer) CollectMetrics(ctx context.Context, req *pluginv2.CollectMetrics_Request) (*pluginv2.CollectMetrics_Response, error) {
	return nil, nil
}

func (s *transformGRPCServer) CheckHealth(ctx context.Context, req *pluginv2.CheckHealth_Request) (*pluginv2.CheckHealth_Response, error) {
	return nil, nil
}

type transformGRPCClient struct {
	pluginv2.BackendServer
	broker *plugin.GRPCBroker
	client pluginv2.BackendClient
}

func (c *transformGRPCClient) GetSchema(ctx context.Context, req *pluginv2.GetSchema_Request) (*pluginv2.GetSchema_Response, error) {
	return c.client.GetSchema(ctx, req)
}

func (c *transformGRPCClient) ValidatePluginConfig(ctx context.Context, req *pluginv2.ValidatePluginConfig_Request) (*pluginv2.ValidatePluginConfig_Response, error) {
	return c.client.ValidatePluginConfig(ctx, req)
}

func (c *transformGRPCClient) Configure(ctx context.Context, req *pluginv2.Configure_Request) (*pluginv2.Configure_Response, error) {
	return c.client.Configure(ctx, req)
}

func (c *transformGRPCClient) CallResource(ctx context.Context, req *pluginv2.CallResource_Request) (*pluginv2.CallResource_Response, error) {
	return c.client.CallResource(ctx, req)
}

func (c *transformGRPCClient) QueryData(ctx context.Context, req *pluginv2.QueryData_Request, callBack TransformCallBack) (*pluginv2.QueryData_Response, error) {
	callBackServer := &TransformCallBackGrpcServer{Impl: callBack}

	var s *grpc.Server
	serverFunc := func(opts []grpc.ServerOption) *grpc.Server {
		s = grpc.NewServer(opts...)
		pluginv2.RegisterTransformCallBackServer(s, callBackServer)

		return s
	}
	brokerID := c.broker.NextId()
	go c.broker.AcceptAndServe(brokerID, serverFunc)
	metadata.AppendToOutgoingContext(ctx, "broker_requestId", string(brokerID))
	res, err := c.client.QueryData(ctx, req)
	s.Stop()
	return res, err
}

func (c *transformGRPCClient) CollectMetrics(ctx context.Context, req *pluginv2.CollectMetrics_Request) (*pluginv2.CollectMetrics_Response, error) {
	return c.client.CollectMetrics(ctx, req)
}

func (c *transformGRPCClient) CheckHealth(ctx context.Context, req *pluginv2.CheckHealth_Request) (*pluginv2.CheckHealth_Response, error) {
	return c.client.CheckHealth(ctx, req)
}

// Callback

type TransformCallBackGrpcClient struct {
	client pluginv2.TransformCallBackClient
}

func (t *TransformCallBackGrpcClient) QueryData(ctx context.Context, req *pluginv2.QueryData_Request) (*pluginv2.QueryData_Response, error) {
	return t.client.QueryData(ctx, req)
}

type TransformCallBackGrpcServer struct {
	Impl TransformCallBack
}

func (g *TransformCallBackGrpcServer) QueryData(ctx context.Context, req *pluginv2.QueryData_Request) (*pluginv2.QueryData_Response, error) {
	return g.Impl.QueryData(ctx, req)
}
