package slo

import (
	"context"
	"net/http"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

var Logger = log.DefaultLogger

// DataSourceSLOMiddlewareName is the middleware name used by Middleware.
const DataSourceSLOMiddlewareName = "slo"

// Middleware applies the duration to the context.
func Middleware() httpclient.Middleware {
	return httpclient.NamedMiddlewareFunc(DataSourceSLOMiddlewareName, RoundTripper)
}

// AddMiddleware adds the middleware to the http client options.
func AddMiddleware(ctx context.Context, s *backend.DataSourceInstanceSettings) (httpclient.Options, error) {
	opts, err := s.HTTPClientOptions(ctx)
	if err != nil {
		Logger.Error("failed to get datasource info", "error", err)
		return opts, err
	}
	opts.Middlewares = append(opts.Middlewares, Middleware())
	return opts, nil
}

// RoundTripper captures the duration of the request in the context
func RoundTripper(_ httpclient.Options, next http.RoundTripper) http.RoundTripper {
	return httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		var duration *Duration
		var httpErr error
		var source = SourceDownstream
		var statusCode int

		start := time.Now()

		ctx := req.Context()
		val := ctx.Value(DurationKey{})
		if val == nil {
			duration = &Duration{value: 0}
			ctx = context.WithValue(ctx, DurationKey{}, duration)
			*req = *req.WithContext(ctx)
		} else {
			duration = val.(*Duration)
		}

		defer func() {
			duration.Add(time.Since(start).Seconds(), source, statusCode, httpErr)
		}()

		res, err := next.RoundTrip(req)
		if err != nil {
			httpErr = err
		}
		if res != nil {
			statusCode = res.StatusCode
			source = Source(FromStatus(backend.Status(res.StatusCode)))
		}
		return res, err
	})
}

// FromStatus returns the error source from backend status
func FromStatus(status backend.Status) backend.ErrorSource {
	return backend.ErrorSourceFromHTTPStatus(int(status))
}

// NewClient wraps the existing http client constructor and adds the duration middleware
func NewClient(opts ...httpclient.Options) (*http.Client, error) {
	if len(opts) == 0 {
		opts = append(opts, httpclient.Options{
			Middlewares: httpclient.DefaultMiddlewares(),
		})
	}
	if len(opts[0].Middlewares) == 0 {
		opts[0].Middlewares = httpclient.DefaultMiddlewares()
	}
	opts[0].Middlewares = append(opts[0].Middlewares, Middleware())
	return httpclient.New(opts...)
}
