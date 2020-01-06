package backend

import (
	"bytes"
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/dataframe"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

// sdkAdapter adapter between protobuf and SDK interfaces.
type sdkAdapter struct {
	PluginHandlers
	TransformHandlers
	handlers          PluginHandlers
	transformHandlers TransformHandlers
}

func (a *sdkAdapter) CollectMetrics(ctx context.Context, protoReq *pluginv2.CollectMetrics_Request) (*pluginv2.CollectMetrics_Response, error) {
	metrics, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	for _, m := range metrics {
		_, err := expfmt.MetricFamilyToText(&buf, m)
		if err != nil {
			continue
		}
	}

	resp := &pluginv2.CollectMetrics_Response{
		Metrics: &pluginv2.CollectMetrics_Payload{
			Prometheus: buf.Bytes(),
		},
	}

	return resp, nil
}

func (a *sdkAdapter) CheckHealth(ctx context.Context, protoReq *pluginv2.CheckHealth_Request) (*pluginv2.CheckHealth_Response, error) {
	return &pluginv2.CheckHealth_Response{
		Status: pluginv2.CheckHealth_Response_OK,
	}, nil
}

func (a *sdkAdapter) DataQuery(ctx context.Context, req *pluginv2.DataQueryRequest) (*pluginv2.DataQueryResponse, error) {
	pc := pluginConfigFromProto(req.Config)

	queries := make([]DataQuery, len(req.Queries))
	for i, q := range req.Queries {
		queries[i] = *dataQueryFromProtobuf(q)
	}

	resp, err := a.handlers.DataQuery(ctx, pc, req.Headers, queries)
	if err != nil {
		return nil, err
	}

	return resp.toProtobuf()
}

func (a *sdkAdapter) Resource(ctx context.Context, req *pluginv2.ResourceRequest) (*pluginv2.ResourceResponse, error) {
	pc := pluginConfigFromProto(req.Config)
	resourceReq := resourceRequestFromProtobuf(req)
	res, err := a.handlers.Resource(ctx, pc, resourceReq)
	if err != nil {
		return nil, err
	}
	return res.toProtobuf(), nil
}

func (a *sdkAdapter) TransformData(ctx context.Context, req *pluginv2.DataQueryRequest, callBack TransformCallBack) (*pluginv2.DataQueryResponse, error) {
	pc := pluginConfigFromProto(req.Config)
	queries := make([]DataQuery, len(req.Queries))
	for i, q := range req.Queries {
		queries[i] = *dataQueryFromProtobuf(q)
	}

	resp, err := a.transformHandlers.DataQuery(ctx, pc, req.Headers, queries, &transformCallBackWrapper{callBack})
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

	return &pluginv2.DataQueryResponse{
		Frames:   encodedFrames,
		Metadata: resp.Metadata,
	}, nil
}
