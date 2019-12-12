package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/dataframe"
	bproto "github.com/grafana/grafana-plugin-sdk-go/genproto/go/backend_plugin"
	plugin "github.com/hashicorp/go-plugin"
)

type transformWrapper struct {
	plugin.NetRPCUnsupportedPlugin

	handlers TransformHandlers
}

func (t *transformWrapper) DataQuery(ctx context.Context, req *bproto.DataQueryRequest, callBack TransformCallBack) (*bproto.DataQueryResponse, error) {
	pc := pluginConfigFromProto(req.Config)

	queries := make([]DataQuery, len(req.Queries))
	for i, q := range req.Queries {
		queries[i] = *dataQueryFromProtobuf(q)
	}

	resp, err := t.handlers.DataQuery(ctx, pc, req.Headers, queries, &transformCallBackWrapper{callBack})
	if err != nil {
		return nil, err
	}

	encodedFrames := make([][]byte, len(resp.Frames))
	for i, frame := range resp.Frames {
		encodedFrames[i], err = dataframe.MarshalArrow(frame)
		if err != nil {
			return nil, err
		}
	}

	return &bproto.DataQueryResponse{
		Frames:   encodedFrames,
		Metadata: resp.Metadata,
	}, nil
}

type TransformHandlers struct {
	TransformDataQueryHandler
}

type TransformDataQueryHandler interface {
	DataQuery(ctx context.Context, pc PluginConfig, headers map[string]string, queries []DataQuery, callBack TransformCallBackHandler) (*DataQueryResponse, error)
}

// Callback

type TransformCallBackHandler interface {
	// TODO: Forget if I actually need PluginConfig on the callback or not.
	DataQuery(ctx context.Context, pc PluginConfig, headers map[string]string, queries []DataQuery) (*DataQueryResponse, error)
}

type transformCallBackWrapper struct {
	callBack TransformCallBack
}

func (tw *transformCallBackWrapper) DataQuery(ctx context.Context, pc PluginConfig, headers map[string]string, queries []DataQuery) (*DataQueryResponse, error) {
	protoQueries := make([]*bproto.DataQuery, len(queries))
	for i, q := range queries {
		protoQueries[i] = q.toProtobuf()
	}

	protoReq := &bproto.DataQueryRequest{
		// TODO: Plugin Config?
		Queries: protoQueries,
	}

	protoRes, err := tw.callBack.DataQuery(ctx, protoReq)
	if err != nil {
		return nil, err
	}

	frames := make([]*dataframe.Frame, len(protoRes.Frames))
	for i, encodedFrame := range protoRes.Frames {
		frames[i], err = dataframe.UnmarshalArrow(encodedFrame)
		if err != nil {
			return nil, err
		}
	}

	return &DataQueryResponse{
		//TODO: Metadata
		Frames: frames,
	}, nil
}
