package backend

import (
	"bytes"
	"context"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	"github.com/prometheus/common/expfmt"
)

type CollectMetricsResponse struct {
	Metrics []byte
}

func (res *CollectMetricsResponse) toProtobuf() *pluginv2.CollectMetrics_Response {
	return &pluginv2.CollectMetrics_Response{
		Metrics: &pluginv2.CollectMetrics_Payload{
			Prometheus: res.Metrics,
		},
	}
}

// HealthStatus is the status of the plugin.
type HealthStatus int

const (
	// HealthStatusUnknown means the status of the plugin is unknown.
	HealthStatusUnknown HealthStatus = iota
	// HealthStatusOk means the status of the plugin is good.
	HealthStatusOk
	// HealthStatusError means the plugin is in an error state.
	HealthStatusError
)

func (ps HealthStatus) toProtobuf() pluginv2.CheckHealth_Response_HealthStatus {
	switch ps {
	case HealthStatusUnknown:
		return pluginv2.CheckHealth_Response_UNKNOWN
	case HealthStatusOk:
		return pluginv2.CheckHealth_Response_OK
	case HealthStatusError:
		return pluginv2.CheckHealth_Response_ERROR
	}
	panic("unsupported protobuf health status type in sdk")
}

type CheckHealthResult struct {
	Status HealthStatus
	Info   string
}

func (res *CheckHealthResult) toProtobuf() *pluginv2.CheckHealth_Response {
	return &pluginv2.CheckHealth_Response{
		Status: res.Status.toProtobuf(),
		Info:   res.Info,
	}
}

type CollectMetricsHandler interface {
	CollectMetrics(ctx context.Context) (*CollectMetricsResponse, error)
}

type CheckHealthHandler interface {
	CheckHealth(ctx context.Context) (*CheckHealthResult, error)
}

type DiagnosticsHandler interface {
	CollectMetricsHandler
	CheckHealthHandler
}

func (w *coreWrapper) CollectMetrics(ctx context.Context, req *pluginv2.CollectMetrics_Request) (*pluginv2.CollectMetrics_Response, error) {
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
