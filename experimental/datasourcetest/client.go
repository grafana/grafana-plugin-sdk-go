package datasourcetest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

type TestPluginClient struct {
	DataClient        pluginv2.DataClient
	DiagnosticsClient pluginv2.DiagnosticsClient
	ResourceClient    pluginv2.ResourceClient

	conn *grpc.ClientConn
}

func newTestPluginClient(addr string) (*TestPluginClient, error) {
	c, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &TestPluginClient{
		conn:              c,
		DiagnosticsClient: pluginv2.NewDiagnosticsClient(c),
		DataClient:        pluginv2.NewDataClient(c),
		ResourceClient:    pluginv2.NewResourceClient(c),
	}, nil
}

func (p *TestPluginClient) QueryData(ctx context.Context, r *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	req := backend.ToProto().QueryDataRequest(r)

	resp, err := p.DataClient.QueryData(ctx, req)
	if err != nil {
		return nil, err
	}

	return backend.FromProto().QueryDataResponse(resp)
}

func (p *TestPluginClient) QueryChunkedData(ctx context.Context, r *backend.QueryChunkedDataRequest) (*backend.QueryDataResponse, error) {
	req := backend.ToProto().QueryChunkedDataRequest(r)

	stream, err := p.DataClient.QueryChunkedData(ctx, req)
	if err != nil {
		return nil, err
	}

	responses := make(backend.Responses)
	frameByKey := make(map[string]*data.Frame)
	var frame *data.Frame

	for {
		sr, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				// End of stream, return accumulated responses
				return &backend.QueryDataResponse{Responses: responses}, nil
			}
			return nil, err
		}

		rsp := responses[sr.RefId]
		if len(sr.Frame) > 0 {
			key := fmt.Sprintf("%s|%s", sr.RefId, sr.FrameId)
			frame = frameByKey[key]
			if frame != nil {
				if err = data.AppendJSONData(frame, sr.Frame); err != nil {
					return nil, fmt.Errorf("error appending data %w", err)
				}
			} else {
				frame = &data.Frame{}
				if err = json.Unmarshal(sr.Frame, frame); err != nil {
					return nil, fmt.Errorf("error parsing response %w", err)
				}
				frameByKey[key] = frame
				rsp.Frames = append(rsp.Frames, frame)
			}
		}

		rsp.Status = backend.Status(sr.Status)
		if sr.Error != "" {
			rsp.Error = errors.New(sr.Error)
		}
		if sr.ErrorSource != "" {
			rsp.ErrorSource = backend.ErrorSource(sr.ErrorSource)
		}
		responses[sr.RefId] = rsp
	}
}

func (p *TestPluginClient) CheckHealth(ctx context.Context, r *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	req := &pluginv2.CheckHealthRequest{
		PluginContext: backend.ToProto().PluginContext(r.PluginContext),
	}

	resp, err := p.DiagnosticsClient.CheckHealth(ctx, req)
	if err != nil {
		return nil, err
	}

	return backend.FromProto().CheckHealthResponse(resp), nil
}

func (p *TestPluginClient) CallResource(ctx context.Context, r *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	protoReq := backend.ToProto().CallResourceRequest(r)
	protoStream, err := p.ResourceClient.CallResource(ctx, protoReq)
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

		if err = sender.Send(backend.FromProto().CallResourceResponse(protoResp)); err != nil {
			return err
		}
	}
}

func (p *TestPluginClient) shutdown() error {
	return p.conn.Close()
}
