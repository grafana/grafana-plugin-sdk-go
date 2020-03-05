package backend

import (
	"bytes"
	"context"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend/plugin"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

// sdkAdapter adapter between low level plugin protocol and SDK interfaces.
type sdkAdapter struct {
	CheckHealthHandler   CheckHealthHandler
	DataQueryHandler     DataQueryHandler
	CallResourceHandler  CallResourceHandler
	TransformDataHandler TransformDataHandler
}

func (a *sdkAdapter) CollectMetrics(ctx context.Context, protoReq *pluginv2.CollectMetrics_Request) (*pluginv2.CollectMetrics_Response, error) {
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	for _, mf := range mfs {
		_, err := expfmt.MetricFamilyToText(&buf, mf)
		if err != nil {
			return nil, err
		}
	}

	return &pluginv2.CollectMetrics_Response{
		Metrics: &pluginv2.CollectMetrics_Payload{
			Prometheus: buf.Bytes(),
		},
	}, nil
}

func (a *sdkAdapter) CheckHealth(ctx context.Context, protoReq *pluginv2.CheckHealth_Request) (*pluginv2.CheckHealth_Response, error) {
	if a.CheckHealthHandler != nil {
		res, err := a.CheckHealthHandler.CheckHealth(ctx, fromProto().HealthCheckRequest(protoReq))
		if err != nil {
			return nil, err
		}
		return toProto().CheckHealthResponse(res), nil
	}

	return &pluginv2.CheckHealth_Response{
		Status: pluginv2.CheckHealth_Response_OK,
	}, nil
}

func (a *sdkAdapter) DataQuery(ctx context.Context, req *pluginv2.DataQueryRequest) (*pluginv2.DataQueryResponse, error) {
	resp, err := a.DataQueryHandler.DataQuery(ctx, fromProto().DataQueryRequest(req))
	if err != nil {
		return nil, err
	}

	return toProto().DataQueryResponse(resp)
}

func (a *sdkAdapter) CallResource(ctx context.Context, protoReq *pluginv2.CallResource_Request) (*pluginv2.CallResource_Response, error) {
	if a.CallResourceHandler == nil {
		return &pluginv2.CallResource_Response{
			Code: http.StatusNotImplemented,
		}, nil
	}

	resp, err := a.CallResourceHandler.CallResource(ctx, fromProto().CallResourceRequest(protoReq))
	if err != nil {
		return nil, err
	}

	return toProto().CallResourceResponse(resp), nil
}

func (a *sdkAdapter) TransformData(ctx context.Context, req *pluginv2.DataQueryRequest, callBack plugin.TransformCallBack) (*pluginv2.DataQueryResponse, error) {
	resp, err := a.TransformDataHandler.TransformData(ctx, fromProto().DataQueryRequest(req), &transformCallBackWrapper{callBack})
	if err != nil {
		return nil, err
	}

	return toProto().DataQueryResponse(resp)
}

type transformCallBackWrapper struct {
	callBack plugin.TransformCallBack
}

func (tw *transformCallBackWrapper) DataQuery(ctx context.Context, req *DataQueryRequest) (*DataQueryResponse, error) {
	protoRes, err := tw.callBack.DataQuery(ctx, toProto().DataQueryRequest(req))
	if err != nil {
		return nil, err
	}

	return fromProto().DataQueryResponse(protoRes)
}
