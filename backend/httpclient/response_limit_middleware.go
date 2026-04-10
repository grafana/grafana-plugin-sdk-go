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

const responseLimitEnvVar = "GF_RESPONSE_LIMIT"

type responseLimitContextKey struct{}

// WithResponseLimitContext stores a response limit in the context, to be picked up by
// ResponseLimitMiddleware on each request. It is called by the backend package from
// WithGrafanaConfig so that GrafanaCfg.ResponseLimit() takes priority over the env var.
// A limit of 0 explicitly disables limiting for the request, regardless of any fallback.
// Note: WithGrafanaConfig only calls this when the cfg limit is > 0, so a zero cfg value
// falls through to the env var / 200MB default rather than disabling limiting entirely.
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

// ResponseLimitMiddleware creates a middleware that limits the size of the response body.
// The effective limit is resolved per-request in priority order:
//  1. GrafanaCfg.ResponseLimit() from context (set via WithResponseLimitContext)
//  2. limit argument, if > 0
//  3. GF_RESPONSE_LIMIT environment variable
//  4. 200MB default
func ResponseLimitMiddleware(limit int64) Middleware {
	return NamedMiddlewareFunc(ResponseLimitMiddlewareName, func(opts Options, next http.RoundTripper) http.RoundTripper {
		fallbackLimit := resolveResponseLimit(limit)
		dsUID := opts.Labels["datasource_uid"]
		dsName := opts.Labels["datasource_name"]

		return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			effectiveLimit := fallbackLimit
			if ctxLimit, ok := responseLimitFromContext(req.Context()); ok {
				effectiveLimit = ctxLimit
			}

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

func resolveResponseLimit(limit int64) int64 {
	if limit > 0 {
		return limit
	}
	// GF_RESPONSE_LIMIT is read once at client construction time. Changes to the env var
	// after the client is built will not take effect until the client is recreated.
	if v, err := strconv.ParseInt(os.Getenv(responseLimitEnvVar), 10, 64); err == nil && v > 0 {
		return v
	}
	return 0
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
