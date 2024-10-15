package backend

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

// NewMetricsMiddleware creates a new HandlerMiddleware that will
// record metrics for requests.
func NewMetricsMiddleware(registerer prometheus.Registerer, namespace string) HandlerMiddleware {
	requestCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "plugin",
		Name:      "request_total",
		Help:      "The total amount of plugin requests",
	}, []string{"endpoint", "status", "status_source"})

	registerer.MustRegister(requestCounter)

	return HandlerMiddlewareFunc(func(next Handler) Handler {
		return &metricsMiddleware{
			BaseHandler:    NewBaseHandler(next),
			requestCounter: requestCounter,
		}
	})
}

type metricsMiddleware struct {
	BaseHandler
	requestCounter *prometheus.CounterVec
}

func (m *metricsMiddleware) instrumentRequest(ctx context.Context, fn func(context.Context) (RequestStatus, error)) error {
	status, err := fn(ctx)
	endpoint := EndpointFromContext(ctx)

	m.requestCounter.WithLabelValues(endpoint.String(), status.String(), string(ErrorSourceFromContext(ctx))).Inc()
	return err
}

func (m *metricsMiddleware) QueryData(ctx context.Context, req *QueryDataRequest) (*QueryDataResponse, error) {
	var resp *QueryDataResponse
	err := m.instrumentRequest(ctx, func(ctx context.Context) (RequestStatus, error) {
		var innerErr error
		resp, innerErr = m.BaseHandler.QueryData(ctx, req)
		return RequestStatusFromQueryDataResponse(resp, innerErr), innerErr
	})

	return resp, err
}

func (m *metricsMiddleware) CallResource(ctx context.Context, req *CallResourceRequest, sender CallResourceResponseSender) error {
	return m.instrumentRequest(ctx, func(ctx context.Context) (RequestStatus, error) {
		innerErr := m.BaseHandler.CallResource(ctx, req, sender)
		return RequestStatusFromError(innerErr), innerErr
	})
}

func (m *metricsMiddleware) CheckHealth(ctx context.Context, req *CheckHealthRequest) (*CheckHealthResult, error) {
	var resp *CheckHealthResult
	err := m.instrumentRequest(ctx, func(ctx context.Context) (RequestStatus, error) {
		var innerErr error
		resp, innerErr = m.BaseHandler.CheckHealth(ctx, req)
		return RequestStatusFromError(innerErr), innerErr
	})

	return resp, err
}

func (m *metricsMiddleware) CollectMetrics(ctx context.Context, req *CollectMetricsRequest) (*CollectMetricsResult, error) {
	var resp *CollectMetricsResult
	err := m.instrumentRequest(ctx, func(ctx context.Context) (RequestStatus, error) {
		var innerErr error
		resp, innerErr = m.BaseHandler.CollectMetrics(ctx, req)
		return RequestStatusFromError(innerErr), innerErr
	})
	return resp, err
}

func (m *metricsMiddleware) SubscribeStream(ctx context.Context, req *SubscribeStreamRequest) (*SubscribeStreamResponse, error) {
	var resp *SubscribeStreamResponse
	err := m.instrumentRequest(ctx, func(ctx context.Context) (RequestStatus, error) {
		var innerErr error
		resp, innerErr = m.BaseHandler.SubscribeStream(ctx, req)
		return RequestStatusFromError(innerErr), innerErr
	})
	return resp, err
}

func (m *metricsMiddleware) PublishStream(ctx context.Context, req *PublishStreamRequest) (*PublishStreamResponse, error) {
	var resp *PublishStreamResponse
	err := m.instrumentRequest(ctx, func(ctx context.Context) (RequestStatus, error) {
		var innerErr error
		resp, innerErr = m.BaseHandler.PublishStream(ctx, req)
		return RequestStatusFromError(innerErr), innerErr
	})
	return resp, err
}

func (m *metricsMiddleware) RunStream(ctx context.Context, req *RunStreamRequest, sender *StreamSender) error {
	err := m.instrumentRequest(ctx, func(ctx context.Context) (RequestStatus, error) {
		innerErr := m.BaseHandler.RunStream(ctx, req, sender)
		return RequestStatusFromError(innerErr), innerErr
	})
	return err
}

func (m *metricsMiddleware) ValidateAdmission(ctx context.Context, req *AdmissionRequest) (*ValidationResponse, error) {
	var resp *ValidationResponse
	err := m.instrumentRequest(ctx, func(ctx context.Context) (RequestStatus, error) {
		var innerErr error
		resp, innerErr = m.BaseHandler.ValidateAdmission(ctx, req)
		return RequestStatusFromError(innerErr), innerErr
	})

	return resp, err
}

func (m *metricsMiddleware) MutateAdmission(ctx context.Context, req *AdmissionRequest) (*MutationResponse, error) {
	var resp *MutationResponse
	err := m.instrumentRequest(ctx, func(ctx context.Context) (RequestStatus, error) {
		var innerErr error
		resp, innerErr = m.BaseHandler.MutateAdmission(ctx, req)
		return RequestStatusFromError(innerErr), innerErr
	})

	return resp, err
}

func (m *metricsMiddleware) ConvertObjects(ctx context.Context, req *ConversionRequest) (*ConversionResponse, error) {
	var resp *ConversionResponse
	err := m.instrumentRequest(ctx, func(ctx context.Context) (RequestStatus, error) {
		var innerErr error
		resp, innerErr = m.BaseHandler.ConvertObjects(ctx, req)
		return RequestStatusFromError(innerErr), innerErr
	})

	return resp, err
}
