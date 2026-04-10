package httpclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// ResponseLimitMiddlewareName is the middleware name used by ResponseLimitMiddleware.
const ResponseLimitMiddlewareName = "response-limit"

// responseLimitEnvVar can be set directly on the plugin server to cap response sizes
// regardless of what Grafana's config system sends. This is useful when running a plugin
// on a separate server in Cloud environments where you want per-pod control.
const responseLimitEnvVar = "GF_DATAPROXY_RESPONSE_LIMIT"

type responseLimitContextKey struct{}

// WithResponseLimitContext injects a response limit into the context so that
// ResponseLimitMiddleware can apply it per-request. A value of 0 explicitly disables
// context-based limiting for that request.
//
// This is called by WithGrafanaConfig in the backend package — plugins do not need to
// call it directly. Note that WithGrafanaConfig only forwards limits > 0, so when
// GrafanaCfg carries no limit the middleware falls back to the limit argument or disables
// entirely.
func WithResponseLimitContext(ctx context.Context, limit int64) context.Context {
	return context.WithValue(ctx, responseLimitContextKey{}, &limit)
}

func responseLimitFromContext(ctx context.Context) (int64, bool) {
	v, ok := ctx.Value(responseLimitContextKey{}).(*int64)
	if !ok || v == nil {
		return 0, false
	}
	return *v, true
}

// ResponseLimitMiddleware limits the size of downstream response bodies.
// When the limit is exceeded the response body returns ErrResponseBodyTooLarge and a
// warning is logged with the datasource identifiers from opts.Labels.
//
// The limit is resolved per-request in the following priority order:
//  1. GF_DATAPROXY_RESPONSE_LIMIT environment variable — read once at client construction,
//     takes highest priority so plugin server operators can override Grafana's config
//  2. GrafanaCfg.ResponseLimit() injected via WithGrafanaConfig (sourced from Grafana's config)
//  3. The limit argument passed to this function, if > 0
//
// If none of the above are set, limiting is disabled for that request.
func ResponseLimitMiddleware(limit int64) Middleware {
	return NamedMiddlewareFunc(ResponseLimitMiddlewareName, func(opts Options, next http.RoundTripper) http.RoundTripper {
		envLimit := parseEnvResponseLimit()
		dsUID := opts.Labels["datasource_uid"]
		dsName := opts.Labels["datasource_name"]

		return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			effectiveLimit := resolveResponseLimit(envLimit, limit, req.Context())

			res, err := next.RoundTrip(req)
			if err != nil {
				return nil, err
			}

			if effectiveLimit <= 0 {
				return res, nil
			}

			if res != nil && res.StatusCode != http.StatusSwitchingProtocols {
				res.Body = &responseLimitBody{
					ReadCloser: MaxBytesReader(res.Body, effectiveLimit),
					ctx:        req.Context(),
					limit:      effectiveLimit,
					dsUID:      dsUID,
					dsName:     dsName,
				}
			}

			return res, nil
		})
	})
}

// parseEnvResponseLimit reads GF_DATAPROXY_RESPONSE_LIMIT once at client construction time.
// Changes to the env var after the client is built will not take effect until the client
// is recreated.
func parseEnvResponseLimit() int64 {
	v, err := strconv.ParseInt(os.Getenv(responseLimitEnvVar), 10, 64)
	if err == nil && v > 0 {
		return v
	}
	return 0
}

// resolveResponseLimit determines the effective limit for a request.
// envLimit wins if set, then the per-request context value from GrafanaCfg, then the
// static limit argument. Returns 0 if none are set, which disables limiting.
func resolveResponseLimit(envLimit, limit int64, ctx context.Context) int64 {
	if envLimit > 0 {
		return envLimit
	}
	if ctxLimit, ok := responseLimitFromContext(ctx); ok {
		return ctxLimit
	}
	return limit
}

type responseLimitBody struct {
	io.ReadCloser
	ctx    context.Context
	limit  int64
	dsUID  string
	dsName string
	once   sync.Once
}

func (b *responseLimitBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if err != nil && errors.Is(err, ErrResponseBodyTooLarge) {
		b.once.Do(func() {
			log.DefaultLogger.FromContext(b.ctx).Warn("downstream response body exceeded limit",
				"datasource_uid", b.dsUID,
				"datasource_name", b.dsName,
				"limit_bytes", b.limit,
			)
		})
	}
	return n, err
}
