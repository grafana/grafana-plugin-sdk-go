package backend

import (
	"context"
)

type TransformHandlers interface {
	TransformDataHandler
}

type TransformDataHandler interface {
	TransformData(ctx context.Context, req *DataQueryRequest, callBack TransformDataCallBackHandler) (*DataQueryResponse, error)
}

// Callback

type TransformDataCallBackHandler interface {
	// TODO: Forget if I actually need PluginConfig on the callback or not.
	QueryData(ctx context.Context, req *DataQueryRequest) (*DataQueryResponse, error)
}
