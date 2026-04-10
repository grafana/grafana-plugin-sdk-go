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

// responseLimitEnvVar is read at client construction and takes priority over all other
// limit sources. It is intended for plugin server operators who need per-pod control
// independent of Grafana's config (e.g. when running plugins on separate servers in Cloud).
const responseLimitEnvVar = "GF_DATAPROXY_RESPONSE_LIMIT"

type responseLimitContextKey struct{}

// WithResponseLimitContext stores a response size limit in the context for
// ResponseLimitMiddleware to apply per-request. Called by WithGrafanaConfig in the
// backend package to propagate GrafanaCfg.ResponseLimit() — plugins do not need to call
// this directly.
//
// Note: GF_DATAPROXY_RESPONSE_LIMIT takes priority over the context value. A context
// value of 0 is only effective when the env var is unset — it disables the cfg-based
// limit and falls through to the limit argument.
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
//  1. GF_DATAPROXY_RESPONSE_LIMIT env var — read once at client construction
//  2. GrafanaCfg.ResponseLimit() from the request context, set by WithGrafanaConfig
//  3. The limit argument, if > 0
//
// If none are set, limiting is disabled.
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
