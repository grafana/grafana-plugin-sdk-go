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

const (
	responseLimitEnvVar  = "GF_RESPONSE_LIMIT"
	defaultResponseLimit = 200 * 1024 * 1024 // 200MB
)

type responseLimitContextKey struct{}

// WithResponseLimitContext stores a response limit in the context, to be picked up by
// ResponseLimitMiddleware on each request. The backend package calls this from
// WithGrafanaConfig so that GrafanaCfg.ResponseLimit() takes priority over the env var.
func WithResponseLimitContext(ctx context.Context, limit int64) context.Context {
	return context.WithValue(ctx, responseLimitContextKey{}, limit)
}

func responseLimitFromContext(ctx context.Context) int64 {
	v, _ := ctx.Value(responseLimitContextKey{}).(int64)
	return v
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
			if ctxLimit := responseLimitFromContext(req.Context()); ctxLimit > 0 {
				effectiveLimit = ctxLimit
			}

			res, err := next.RoundTrip(req)
			if err != nil {
				return nil, err
			}

			if res != nil && res.StatusCode != http.StatusSwitchingProtocols {
				res.Body = &responseLimitBody{
					ReadCloser: MaxBytesReader(res.Body, effectiveLimit),
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
	if v, err := strconv.ParseInt(os.Getenv(responseLimitEnvVar), 10, 64); err == nil && v > 0 {
		return v
	}
	return defaultResponseLimit
}

// responseLimitBody wraps MaxBytesReader to log when the response limit is exceeded.
type responseLimitBody struct {
	io.ReadCloser
	limit  int64
	dsUID  string
	dsName string
	once   sync.Once
}

func (b *responseLimitBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if err != nil && errors.Is(err, ErrResponseBodyTooLarge) {
		b.once.Do(func() {
			log.DefaultLogger.Warn("downstream response body exceeded limit",
				"datasource_uid", b.dsUID,
				"datasource_name", b.dsName,
				"limit_bytes", b.limit,
			)
		})
	}
	return n, err
}
