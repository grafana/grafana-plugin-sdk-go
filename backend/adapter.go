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

func (a *sdkAdapter) CollectMetrics(ctx context.Context, protoReq *pluginv2.CollectMetricsRequest) (*pluginv2.CollectMetricsResponse, error) {
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

	return &pluginv2.CollectMetricsResponse{
		Metrics: &pluginv2.CollectMetricsResponse_Payload{
			Prometheus: buf.Bytes(),
		},
	}, nil
}

func (a *sdkAdapter) CheckHealth(ctx context.Context, protoReq *pluginv2.CheckHealthRequest) (*pluginv2.CheckHealthResponse, error) {
	if a.CheckHealthHandler != nil {
		res, err := a.CheckHealthHandler.CheckHealth(ctx, fromProto().HealthCheckRequest(protoReq))
		if err != nil {
			return nil, err
		}
		return toProto().CheckHealthResponse(res), nil
	}

	return &pluginv2.CheckHealthResponse{
		Status: pluginv2.CheckHealthResponse_OK,
	}, nil
}

func (a *sdkAdapter) QueryData(ctx context.Context, req *pluginv2.QueryDataRequest) (*pluginv2.QueryDataResponse, error) {
	resp, err := a.DataQueryHandler.DataQuery(ctx, fromProto().DataQueryRequest(req))
	if err != nil {
		return nil, err
	}

	return toProto().DataQueryResponse(resp)
}

type callResourceResponseSenderFunc func(resp *CallResourceResponse) error

func (fn callResourceResponseSenderFunc) Send(resp *CallResourceResponse) error {
	return fn(resp)
}

func (a *sdkAdapter) CallResource(protoReq *pluginv2.CallResourceRequest, protoSrv pluginv2.Resource_CallResourceServer) error {
	if a.CallResourceHandler == nil {
		return protoSrv.Send(&pluginv2.CallResourceResponse{
			Code: http.StatusNotImplemented,
		})
	}

	fn := callResourceResponseSenderFunc(func(resp *CallResourceResponse) error {
		return protoSrv.Send(toProto().CallResourceResponse(resp))
	})

	return a.CallResourceHandler.CallResource(protoSrv.Context(), fromProto().CallResourceRequest(protoReq), fn)
}

func (a *sdkAdapter) TransformData(ctx context.Context, req *pluginv2.QueryDataRequest, callBack plugin.TransformDataCallBack) (*pluginv2.QueryDataResponse, error) {
	resp, err := a.TransformDataHandler.TransformData(ctx, fromProto().DataQueryRequest(req), &transformDataCallBackWrapper{callBack})
	if err != nil {
		return nil, err
	}

	return toProto().DataQueryResponse(resp)
}

type transformDataCallBackWrapper struct {
	callBack plugin.TransformDataCallBack
}

func (tw *transformDataCallBackWrapper) QueryData(ctx context.Context, req *DataQueryRequest) (*DataQueryResponse, error) {
	protoRes, err := tw.callBack.QueryData(ctx, toProto().DataQueryRequest(req))
	if err != nil {
		return nil, err
	}

	return fromProto().DataQueryResponse(protoRes)
}
