package backend

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/backend/tracing"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type handlerWrapperFunc func(ctx context.Context) (RequestStatus, error)

func setupContext(ctx context.Context, endpoint Endpoint) context.Context {
	ctx = WithEndpoint(ctx, endpoint)
	ctx = propagateTenantIDIfPresent(ctx)

	return ctx
}

func wrapHandler(ctx context.Context, pluginCtx PluginContext, next handlerWrapperFunc) error {
	ctx = setupHandlerContext(ctx, pluginCtx)
	wrapper := logWrapper(next)
	wrapper = metricWrapper(wrapper)
	wrapper = tracingWrapper(wrapper)
	_, err := wrapper(ctx)
	return err
}

func setupHandlerContext(ctx context.Context, pluginCtx PluginContext) context.Context {
	ctx = WithGrafanaConfig(ctx, pluginCtx.GrafanaConfig)
	ctx = WithPluginContext(ctx, pluginCtx)
	ctx = WithUser(ctx, pluginCtx.User)
	ctx = withContextualLogAttributes(ctx, pluginCtx)
	ctx = WithUserAgent(ctx, pluginCtx.UserAgent)
	return ctx
}

var pluginRequestCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: "grafana_plugin",
	Name:      "request_total",
	Help:      "The total amount of plugin requests",
}, []string{"endpoint", "status"})

var once = sync.Once{}

func metricWrapper(next handlerWrapperFunc) handlerWrapperFunc {
	once.Do(func() {
		prometheus.MustRegister(pluginRequestCounter)
	})

	return func(ctx context.Context) (RequestStatus, error) {
		endpoint := EndpointFromContext(ctx)
		status, err := next(ctx)

		// TODO include error/status source
		pluginRequestCounter.WithLabelValues(endpoint.String(), status.String()).Inc()

		return status, err
	}
}

func tracingWrapper(next handlerWrapperFunc) handlerWrapperFunc {
	return func(ctx context.Context) (RequestStatus, error) {
		endpoint := EndpointFromContext(ctx)
		pluginCtx := PluginConfigFromContext(ctx)
		ctx, span := tracing.DefaultTracer().Start(ctx, fmt.Sprintf("sdk.%s", endpoint), trace.WithAttributes(
			attribute.String("plugin_id", pluginCtx.PluginID),
			attribute.Int64("org_id", pluginCtx.OrgID),
		))
		defer span.End()

		if pluginCtx.DataSourceInstanceSettings != nil {
			span.SetAttributes(
				attribute.String("datasource_name", pluginCtx.DataSourceInstanceSettings.Name),
				attribute.String("datasource_uid", pluginCtx.DataSourceInstanceSettings.UID),
			)
		}

		if u := pluginCtx.User; u != nil {
			span.SetAttributes(attribute.String("user", pluginCtx.User.Name))
		}

		status, err := next(ctx)

		span.SetAttributes(
			attribute.String("request_status", status.String()),
		)

		if err != nil {
			return status, tracing.Error(span, err)
		}

		return status, err
	}
}

func logWrapper(next handlerWrapperFunc) handlerWrapperFunc {
	return func(ctx context.Context) (RequestStatus, error) {
		start := time.Now()
		status, err := next(ctx)

		logParams := []any{
			"status", status.String(),
			"duration", time.Since(start),
		}

		if err != nil {
			logParams = append(logParams, "error", err)
		}

		// TODO status source
		// logParams = append(logParams, "statusSource", pluginrequestmeta.StatusSourceFromContext(ctx))

		ctxLogger := Logger.FromContext(ctx)
		logFunc := ctxLogger.Debug
		if status > RequestStatusOK {
			logFunc = ctxLogger.Error
		}

		logFunc("Plugin Request Completed", logParams...)

		return status, err
	}
}

func withHeaderMiddleware(ctx context.Context, headers http.Header) context.Context {
	if len(headers) > 0 {
		ctx = httpclient.WithContextualMiddleware(ctx,
			httpclient.MiddlewareFunc(func(opts httpclient.Options, next http.RoundTripper) http.RoundTripper {
				if !opts.ForwardHTTPHeaders {
					return next
				}

				return httpclient.RoundTripperFunc(func(qreq *http.Request) (*http.Response, error) {
					// Only set a header if it is not already set.
					for k, v := range headers {
						if qreq.Header.Get(k) == "" {
							for _, vv := range v {
								qreq.Header.Add(k, vv)
							}
						}
					}
					return next.RoundTrip(qreq)
				})
			}))
	}
	return ctx
}
