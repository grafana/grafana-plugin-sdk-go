package shared

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

type TransformCallBack interface {
	DataQuery(ctx context.Context, req *pluginv2.DataQueryRequest) (*pluginv2.DataQueryResponse, error)
}

type TransformServer interface {
	TransformData(ctx context.Context, req *pluginv2.DataQueryRequest, callback TransformCallBack) (*pluginv2.DataQueryResponse, error)
}
