package slo

import (
	"context"
	"net/http"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
)

// DurationMiddlewareName is the middleware name used by DurationMiddleware.
const DurationMiddlewareName = "Duration"

// DurationMiddleware applies the duration to the context.
func DurationMiddleware() httpclient.Middleware {
	return httpclient.NamedMiddlewareFunc(DurationMiddlewareName, DurationRoundTripper)
}

func AddDurationMiddleware(ctx context.Context, s *backend.DataSourceInstanceSettings) (httpclient.Options, error) {
	opts, err := s.HTTPClientOptions(ctx)
	if err != nil {
		return opts, err
	}
	opts.Middlewares = append(opts.Middlewares, DurationMiddleware())
	return opts, nil
}

// DurationRoundTripper captures the duration of the request in the context
func DurationRoundTripper(_ httpclient.Options, next http.RoundTripper) http.RoundTripper {
	return httpclient.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		var duration *Duration
		var httpErr error
		var source = SourceDownstream
		var statusCode int

		start := time.Now()

		ctx := req.Context()
		val := ctx.Value(DurationKey)
		if val == nil {
			// TODO: this doesn't seem to change the context upstream
			// so we always have to add the value to the context in the QueryData method
			duration = &Duration{Value: 0}
			ctx = context.WithValue(ctx, DurationKey, duration)
			req = req.WithContext(ctx)
		} else {
			duration = val.(*Duration)
		}

		defer func() {
			duration.Value += time.Since(start).Seconds()
			if duration.Status == "" {
				duration.Status = "ok"
			}
			if httpErr != nil {
				duration.Status = "error"
			}
			if statusCode >= 400 {
				duration.Status = "error"
			}

			// If the status code is now ok, but the previous status code was 401 or 403, mark it as ok
			// assuming a successful re-authentication ( token refresh, etc )
			if statusCode < 400 && (duration.StatusCode == 401 || duration.StatusCode == 403) {
				duration.Status = "ok"
			}

			duration.StatusCode = statusCode
			duration.Source = source
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
