package app

import (
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
)

// ServeOpts options for serving an app plugin.
type ServeOpts struct {
	// CheckHealthHandler handler for health checks.
	// Optional to implement.
	backend.CheckHealthHandler

	// CallResourceHandler handler for resource calls.
	// Optional to implement.
	backend.CallResourceHandler

	// MaxGRPCReceiveMsgSize the max gRPC message size in bytes the plugin can receive.
	// If this is <= 0, gRPC uses the default 4MB.
	MaxGRPCReceiveMsgSize int

	// MaxGRPCSendMsgSize the max gRPC message size in bytes the plugin can send.
	// If this is <= 0, gRPC uses the default `math.MaxInt32`.
	MaxGRPCSendMsgSize int
}

// Plugin represent an app plugin.
type Plugin interface {
	// ServeOpts returns ServeOpts.
	ServeOpts() ServeOpts

	// Serve starts serving the data source plugin over gRPC.
	Serve() error
}

// New creates a new app plugin.
//
// If factory is nil, New panics.
// If serveFn is nil, New panics.
func New(factory InstanceFactoryFunc, serveFn func(im instancemgmt.InstanceManager) ServeOpts) Plugin {
	if factory == nil {
		panic("app: factory cannot be nil")
	}

	if serveFn == nil {
		panic("app: serveFn cannot be nil")
	}

	ip := NewInstanceProvider(factory)
	im := instancemgmt.New(ip)
	opts := serveFn(im)

	return &plugin{
		im:   im,
		opts: opts,
	}
}

type plugin struct {
	im   instancemgmt.InstanceManager
	opts ServeOpts
}

func (p *plugin) ServeOpts() ServeOpts {
	return p.opts
}

func (p *plugin) Serve() error {
	opts := backend.ServeOpts{
		CheckHealthHandler:    p.opts.CheckHealthHandler,
		CallResourceHandler:   p.opts.CallResourceHandler,
		MaxGRPCReceiveMsgSize: p.opts.MaxGRPCReceiveMsgSize,
		MaxGRPCSendMsgSize:    p.opts.MaxGRPCSendMsgSize,
	}
	return backend.Serve(opts)
}
