package datasource

import (
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
)

// ServeOpts options for serving a data source plugin.
type ServeOpts struct {
	// CheckHealthHandler handler for health checks.
	// Optional to implement.
	backend.CheckHealthHandler

	// CallResourceHandler handler for resource calls.
	// Optional to implement.
	backend.CallResourceHandler

	// QueryDataHandler handler for data queries.
	// Required to implement.
	backend.QueryDataHandler

	// MaxGRPCReceiveMsgSize the max gRPC message size in bytes the plugin can receive.
	// If this is <= 0, gRPC uses the default 4MB.
	MaxGRPCReceiveMsgSize int

	// MaxGRPCSendMsgSize the max gRPC message size in bytes the plugin can send.
	// If this is <= 0, gRPC uses the default `math.MaxInt32`.
	MaxGRPCSendMsgSize int
}

// Plugin represent a data source plugin.
type Plugin interface {
	// ServeOpts returns ServeOpts.
	ServeOpts() ServeOpts

	// Serve starts serving the data source plugin over gRPC.
	Serve() error
}

// New creates a new data source plugin.
//
// If factory is nil, New panics.
// If serveFn is nil, New panics.
// If serveFN returns ServeOpts with a nil backend.QueryDataHandler, New panics.
func New(factory InstanceFactoryFunc, serveFn func(im instancemgmt.InstanceManager) ServeOpts) Plugin {
	if factory == nil {
		panic("datasource: factory cannot be nil")
	}

	if serveFn == nil {
		panic("datasource: serveFn cannot be nil")
	}

	ip := NewInstanceProvider(factory)
	im := instancemgmt.New(ip)
	opts := serveFn(im)

	if opts.QueryDataHandler == nil {
		panic("datasource: QueryDataHandler cannot be nil")
	}

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
		QueryDataHandler:      p.opts.QueryDataHandler,
		MaxGRPCReceiveMsgSize: p.opts.MaxGRPCReceiveMsgSize,
		MaxGRPCSendMsgSize:    p.opts.MaxGRPCSendMsgSize,
	}
	return backend.Serve(opts)
}
