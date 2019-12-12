package backend

import (
	"context"

	bproto "github.com/grafana/grafana-plugin-sdk-go/genproto/go/backend_plugin"
	plugin "github.com/hashicorp/go-plugin"
)

type transformWrapper struct {
	plugin.NetRPCUnsupportedPlugin

	Handlers TransformHandlers
}

func (t *transformWrapper) DataQuery(ctx context.Context, req *bproto.DataQueryRequest, callBack TransformCallBack) (*bproto.DataQueryResponse, error) {
	return nil, nil
}

type TransformHandlers struct {
	TransformHandler
}

type TransformHandler interface {
	DataQuery(ctx context.Context, queries []DataQuery, callBack TransformCallBackHandler) (DataQueryResponse, error)
}

// Callback

type TransformCallBackHandler interface {
	DataQuery(ctx context.Context, queries []DataQuery) (DataQueryResponse, error)
}

type transformCallBackWrapper struct {
	callBack TransformCallBack
}

func (tw *transformCallBackWrapper) DataQuery(ctx context.Context, queries []DataQuery) (DataQueryResponse, error) {
	return DataQueryResponse{}, nil
}
