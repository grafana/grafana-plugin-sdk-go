package backend

import (
	"context"
)

type TransformHandlers interface {
	TransformDataHandler
}

// TransformDataHandler is a type that can transform data.
type TransformDataHandler interface {
	TransformData(ctx context.Context, req *QueryDataRequest, callBack TransformDataCallBackHandler) (*QueryDataResponse, error)
}

// TransformDataCallBackHandler is a type that can handle callbacks from TransformDataHandler.
type TransformDataCallBackHandler interface {
	// QueryData is a data transformation callback for querying for data.
	// TODO: Forget if I actually need PluginConfig on the callback or not.
	QueryData(ctx context.Context, req *QueryDataRequest) (*QueryDataResponse, error)
}
