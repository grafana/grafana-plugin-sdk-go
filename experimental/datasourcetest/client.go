package datasourcetest

import (
	"context"
	"errors"
	"io"

	"github.com/grafana/grafana-plugin-sdk-go/data"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

type TestPluginClient struct {
	DataClient        pluginv2.DataClient
	DiagnosticsClient pluginv2.DiagnosticsClient
	ResourceClient    pluginv2.ResourceClient
	InformationClient pluginv2.InformationClient

	conn *grpc.ClientConn
}

func newTestPluginClient(addr string) (*TestPluginClient, error) {
	c, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(16*1024*1024)))
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

	// A refID identifies a response, while frameID identifies frames within it.
	// Frames with identical frameIDs represent chunked data and should be merged.
	// Frames with unique frameIDs represent distinct data and should be appended.

	type refState struct {
		dr      *backend.DataResponse
		frameID string
		frame   *data.Frame
	}

	responses := backend.Responses{}
	states := make(map[string]*refState)

	for {
		sr, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return &backend.QueryDataResponse{Responses: responses}, nil
			}
			return nil, err
		}

		f, err := data.UnmarshalArrowFrame(sr.Frame)
		if err != nil {
			return nil, err
		}

		// First time we see this response?
		st, ok := states[sr.RefId]
		if !ok {
			dr := &backend.DataResponse{
				Frames: data.Frames{f},
				Status: backend.Status(sr.Status),
			}
			if sr.Error != "" {
				dr.Error = errors.New(sr.Error)
				dr.ErrorSource = backend.ErrorSource(sr.ErrorSource)
			}

			st = &refState{
				dr:      dr,
				frameID: sr.FrameId,
				frame:   f,
			}
			states[sr.RefId] = st

			// Store a value copy for the final response map.
			responses[sr.RefId] = *dr
			continue
		}

		// Frames with identical frameIDs represent chunked data and should be merged.
		if sr.FrameId == st.frameID {
			if len(f.Fields) != len(st.frame.Fields) {
				return nil, errors.New("received chunked frame with mismatched field count")
			}
			for i, field := range f.Fields {
				st.frame.Fields[i].AppendAll(field)
			}
			continue
		}

		// Frames with unique frameIDs represent distinct data and should be appended.
		st.dr.Frames = append(st.dr.Frames, f)
		st.frameID = sr.FrameId
		st.frame = f

		// Store a value copy for the final response map.
		responses[sr.RefId] = *st.dr
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

func (p *TestPluginClient) Tables(ctx context.Context, r *backend.TableInformationRequest) (*backend.TableInformationResponse, error) {
	req := &pluginv2.TableInformationRequest{
		PluginContext: backend.ToProto().PluginContext(r.PluginContext),
	}

	resp, err := p.InformationClient.Tables(ctx, req)
	if err != nil {
		return nil, err
	}

	return backend.FromProto().TableInformationResponse(resp), nil
}

func (p *TestPluginClient) shutdown() error {
	return p.conn.Close()
}
