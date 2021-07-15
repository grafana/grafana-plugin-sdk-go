package plugintest

import (
	"context"
	"errors"
	"io"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PluginClient represents a client to a running plugin.
type PluginClient struct {
	dataClient        pluginv2.DataClient
	diagnosticsClient pluginv2.DiagnosticsClient
	resourceClient    pluginv2.ResourceClient
}

// NewPluginClient takes the address of a plugin, attempts to connect to it,
// and returns a PluginClient to that plugin. NewPluginClient blocks while
// establishing connection to the plugin.
func NewPluginClient(addr string) (*PluginClient, ShutdownFunc, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, func() {}, err
	}

	plugin := &PluginClient{
		diagnosticsClient: pluginv2.NewDiagnosticsClient(conn),
		dataClient:        pluginv2.NewDataClient(conn),
		resourceClient:    pluginv2.NewResourceClient(conn),
	}

	return plugin, func() {
		conn.Close()
	}, nil
}

// CheckHealth makes a CheckHealth request to the connected plugin
func (p *PluginClient) CheckHealth(ctx context.Context, r *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	req := &pluginv2.CheckHealthRequest{
		PluginContext: backend.ToProto().PluginContext(r.PluginContext),
	}

	resp, err := p.diagnosticsClient.CheckHealth(ctx, req)
	if err != nil {
		return nil, err
	}

	return backend.FromProto().CheckHealthResponse(resp), nil
}

// CallResource makes a CallResource request to the connected plugin
func (p *PluginClient) CallResource(ctx context.Context, r *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	protoReq := backend.ToProto().CallResourceRequest(r)
	protoStream, err := p.resourceClient.CallResource(ctx, protoReq)
	if err != nil {
		if status.Code(err) == codes.Unimplemented {
			return errors.New("method not implemented")
		}

		return err
	}

	for {
		protoResp, err := protoStream.Recv()
		if err != nil {
			if status.Code(err) == codes.Unimplemented {
				return errors.New("method not implemented")
			}

			if errors.Is(err, io.EOF) {
				return nil
			}

			return err
		}

		if err := sender.Send(backend.FromProto().CallResourceResponse(protoResp)); err != nil {
			return err
		}
	}
}

// QueryData makes a QueryData request to the connected plugin
func (p *PluginClient) QueryData(ctx context.Context, r *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	req := backend.ToProto().QueryDataRequest(r)

	resp, err := p.dataClient.QueryData(ctx, req)
	if err != nil {
		return nil, err
	}

	return backend.FromProto().QueryDataResponse(resp)
}
