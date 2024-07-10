package backend

import (
	"context"
	"net/http"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
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
