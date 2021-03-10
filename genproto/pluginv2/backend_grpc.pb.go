// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package pluginv2

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// ResourceClient is the client API for Resource service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ResourceClient interface {
	CallResource(ctx context.Context, in *CallResourceRequest, opts ...grpc.CallOption) (Resource_CallResourceClient, error)
}

type resourceClient struct {
	cc grpc.ClientConnInterface
}

func NewResourceClient(cc grpc.ClientConnInterface) ResourceClient {
	return &resourceClient{cc}
}

func (c *resourceClient) CallResource(ctx context.Context, in *CallResourceRequest, opts ...grpc.CallOption) (Resource_CallResourceClient, error) {
	stream, err := c.cc.NewStream(ctx, &Resource_ServiceDesc.Streams[0], "/pluginv2.Resource/CallResource", opts...)
	if err != nil {
		return nil, err
	}
	x := &resourceCallResourceClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Resource_CallResourceClient interface {
	Recv() (*CallResourceResponse, error)
	grpc.ClientStream
}

type resourceCallResourceClient struct {
	grpc.ClientStream
}

func (x *resourceCallResourceClient) Recv() (*CallResourceResponse, error) {
	m := new(CallResourceResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ResourceServer is the server API for Resource service.
// All implementations should embed UnimplementedResourceServer
// for forward compatibility
type ResourceServer interface {
	CallResource(*CallResourceRequest, Resource_CallResourceServer) error
}

// UnimplementedResourceServer should be embedded to have forward compatible implementations.
type UnimplementedResourceServer struct {
}

func (UnimplementedResourceServer) CallResource(*CallResourceRequest, Resource_CallResourceServer) error {
	return status.Errorf(codes.Unimplemented, "method CallResource not implemented")
}

// UnsafeResourceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ResourceServer will
// result in compilation errors.
type UnsafeResourceServer interface {
	mustEmbedUnimplementedResourceServer()
}

func RegisterResourceServer(s grpc.ServiceRegistrar, srv ResourceServer) {
	s.RegisterService(&Resource_ServiceDesc, srv)
}

func _Resource_CallResource_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(CallResourceRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ResourceServer).CallResource(m, &resourceCallResourceServer{stream})
}

type Resource_CallResourceServer interface {
	Send(*CallResourceResponse) error
	grpc.ServerStream
}

type resourceCallResourceServer struct {
	grpc.ServerStream
}

func (x *resourceCallResourceServer) Send(m *CallResourceResponse) error {
	return x.ServerStream.SendMsg(m)
}

// Resource_ServiceDesc is the grpc.ServiceDesc for Resource service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Resource_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "pluginv2.Resource",
	HandlerType: (*ResourceServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "CallResource",
			Handler:       _Resource_CallResource_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "backend.proto",
}

// DataClient is the client API for Data service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type DataClient interface {
	QueryData(ctx context.Context, in *QueryDataRequest, opts ...grpc.CallOption) (*QueryDataResponse, error)
}

type dataClient struct {
	cc grpc.ClientConnInterface
}

func NewDataClient(cc grpc.ClientConnInterface) DataClient {
	return &dataClient{cc}
}

func (c *dataClient) QueryData(ctx context.Context, in *QueryDataRequest, opts ...grpc.CallOption) (*QueryDataResponse, error) {
	out := new(QueryDataResponse)
	err := c.cc.Invoke(ctx, "/pluginv2.Data/QueryData", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DataServer is the server API for Data service.
// All implementations should embed UnimplementedDataServer
// for forward compatibility
type DataServer interface {
	QueryData(context.Context, *QueryDataRequest) (*QueryDataResponse, error)
}

// UnimplementedDataServer should be embedded to have forward compatible implementations.
type UnimplementedDataServer struct {
}

func (UnimplementedDataServer) QueryData(context.Context, *QueryDataRequest) (*QueryDataResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method QueryData not implemented")
}

// UnsafeDataServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DataServer will
// result in compilation errors.
type UnsafeDataServer interface {
	mustEmbedUnimplementedDataServer()
}

func RegisterDataServer(s grpc.ServiceRegistrar, srv DataServer) {
	s.RegisterService(&Data_ServiceDesc, srv)
}

func _Data_QueryData_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryDataRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DataServer).QueryData(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pluginv2.Data/QueryData",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DataServer).QueryData(ctx, req.(*QueryDataRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Data_ServiceDesc is the grpc.ServiceDesc for Data service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Data_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "pluginv2.Data",
	HandlerType: (*DataServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "QueryData",
			Handler:    _Data_QueryData_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "backend.proto",
}

// StreamClient is the client API for Stream service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type StreamClient interface {
	// Called when a user tries to connect to a plugin/datasource managed channel.
	CanSubscribeToStream(ctx context.Context, in *SubscribeToStreamRequest, opts ...grpc.CallOption) (*SubscribeToStreamResponse, error)
	// RunStream will be initiated by Grafana to consume a stream from a plugin.
	// For streams with keepalive set this will only be called once the first client
	// successfully subscribed to a stream channel. And when there are no longer any
	// subscribers, the call will be terminated by Grafana.
	RunStream(ctx context.Context, in *RunStreamRequest, opts ...grpc.CallOption) (Stream_RunStreamClient, error)
}

type streamClient struct {
	cc grpc.ClientConnInterface
}

func NewStreamClient(cc grpc.ClientConnInterface) StreamClient {
	return &streamClient{cc}
}

func (c *streamClient) CanSubscribeToStream(ctx context.Context, in *SubscribeToStreamRequest, opts ...grpc.CallOption) (*SubscribeToStreamResponse, error) {
	out := new(SubscribeToStreamResponse)
	err := c.cc.Invoke(ctx, "/pluginv2.Stream/CanSubscribeToStream", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *streamClient) RunStream(ctx context.Context, in *RunStreamRequest, opts ...grpc.CallOption) (Stream_RunStreamClient, error) {
	stream, err := c.cc.NewStream(ctx, &Stream_ServiceDesc.Streams[0], "/pluginv2.Stream/RunStream", opts...)
	if err != nil {
		return nil, err
	}
	x := &streamRunStreamClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Stream_RunStreamClient interface {
	Recv() (*StreamPacket, error)
	grpc.ClientStream
}

type streamRunStreamClient struct {
	grpc.ClientStream
}

func (x *streamRunStreamClient) Recv() (*StreamPacket, error) {
	m := new(StreamPacket)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// StreamServer is the server API for Stream service.
// All implementations should embed UnimplementedStreamServer
// for forward compatibility
type StreamServer interface {
	// Called when a user tries to connect to a plugin/datasource managed channel.
	CanSubscribeToStream(context.Context, *SubscribeToStreamRequest) (*SubscribeToStreamResponse, error)
	// RunStream will be initiated by Grafana to consume a stream from a plugin.
	// For streams with keepalive set this will only be called once the first client
	// successfully subscribed to a stream channel. And when there are no longer any
	// subscribers, the call will be terminated by Grafana.
	RunStream(*RunStreamRequest, Stream_RunStreamServer) error
}

// UnimplementedStreamServer should be embedded to have forward compatible implementations.
type UnimplementedStreamServer struct {
}

func (UnimplementedStreamServer) CanSubscribeToStream(context.Context, *SubscribeToStreamRequest) (*SubscribeToStreamResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CanSubscribeToStream not implemented")
}
func (UnimplementedStreamServer) RunStream(*RunStreamRequest, Stream_RunStreamServer) error {
	return status.Errorf(codes.Unimplemented, "method RunStream not implemented")
}

// UnsafeStreamServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to StreamServer will
// result in compilation errors.
type UnsafeStreamServer interface {
	mustEmbedUnimplementedStreamServer()
}

func RegisterStreamServer(s grpc.ServiceRegistrar, srv StreamServer) {
	s.RegisterService(&Stream_ServiceDesc, srv)
}

func _Stream_CanSubscribeToStream_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SubscribeToStreamRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(StreamServer).CanSubscribeToStream(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pluginv2.Stream/CanSubscribeToStream",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(StreamServer).CanSubscribeToStream(ctx, req.(*SubscribeToStreamRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Stream_RunStream_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(RunStreamRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(StreamServer).RunStream(m, &streamRunStreamServer{stream})
}

type Stream_RunStreamServer interface {
	Send(*StreamPacket) error
	grpc.ServerStream
}

type streamRunStreamServer struct {
	grpc.ServerStream
}

func (x *streamRunStreamServer) Send(m *StreamPacket) error {
	return x.ServerStream.SendMsg(m)
}

// Stream_ServiceDesc is the grpc.ServiceDesc for Stream service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Stream_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "pluginv2.Stream",
	HandlerType: (*StreamServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CanSubscribeToStream",
			Handler:    _Stream_CanSubscribeToStream_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "RunStream",
			Handler:       _Stream_RunStream_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "backend.proto",
}

// DiagnosticsClient is the client API for Diagnostics service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type DiagnosticsClient interface {
	CheckHealth(ctx context.Context, in *CheckHealthRequest, opts ...grpc.CallOption) (*CheckHealthResponse, error)
	CollectMetrics(ctx context.Context, in *CollectMetricsRequest, opts ...grpc.CallOption) (*CollectMetricsResponse, error)
}

type diagnosticsClient struct {
	cc grpc.ClientConnInterface
}

func NewDiagnosticsClient(cc grpc.ClientConnInterface) DiagnosticsClient {
	return &diagnosticsClient{cc}
}

func (c *diagnosticsClient) CheckHealth(ctx context.Context, in *CheckHealthRequest, opts ...grpc.CallOption) (*CheckHealthResponse, error) {
	out := new(CheckHealthResponse)
	err := c.cc.Invoke(ctx, "/pluginv2.Diagnostics/CheckHealth", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *diagnosticsClient) CollectMetrics(ctx context.Context, in *CollectMetricsRequest, opts ...grpc.CallOption) (*CollectMetricsResponse, error) {
	out := new(CollectMetricsResponse)
	err := c.cc.Invoke(ctx, "/pluginv2.Diagnostics/CollectMetrics", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DiagnosticsServer is the server API for Diagnostics service.
// All implementations should embed UnimplementedDiagnosticsServer
// for forward compatibility
type DiagnosticsServer interface {
	CheckHealth(context.Context, *CheckHealthRequest) (*CheckHealthResponse, error)
	CollectMetrics(context.Context, *CollectMetricsRequest) (*CollectMetricsResponse, error)
}

// UnimplementedDiagnosticsServer should be embedded to have forward compatible implementations.
type UnimplementedDiagnosticsServer struct {
}

func (UnimplementedDiagnosticsServer) CheckHealth(context.Context, *CheckHealthRequest) (*CheckHealthResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CheckHealth not implemented")
}
func (UnimplementedDiagnosticsServer) CollectMetrics(context.Context, *CollectMetricsRequest) (*CollectMetricsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CollectMetrics not implemented")
}

// UnsafeDiagnosticsServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DiagnosticsServer will
// result in compilation errors.
type UnsafeDiagnosticsServer interface {
	mustEmbedUnimplementedDiagnosticsServer()
}

func RegisterDiagnosticsServer(s grpc.ServiceRegistrar, srv DiagnosticsServer) {
	s.RegisterService(&Diagnostics_ServiceDesc, srv)
}

func _Diagnostics_CheckHealth_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CheckHealthRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DiagnosticsServer).CheckHealth(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pluginv2.Diagnostics/CheckHealth",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DiagnosticsServer).CheckHealth(ctx, req.(*CheckHealthRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Diagnostics_CollectMetrics_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CollectMetricsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DiagnosticsServer).CollectMetrics(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/pluginv2.Diagnostics/CollectMetrics",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DiagnosticsServer).CollectMetrics(ctx, req.(*CollectMetricsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Diagnostics_ServiceDesc is the grpc.ServiceDesc for Diagnostics service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Diagnostics_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "pluginv2.Diagnostics",
	HandlerType: (*DiagnosticsServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CheckHealth",
			Handler:    _Diagnostics_CheckHealth_Handler,
		},
		{
			MethodName: "CollectMetrics",
			Handler:    _Diagnostics_CollectMetrics_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "backend.proto",
}
