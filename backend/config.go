package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend/useragent"
	"github.com/grafana/grafana-plugin-sdk-go/config"
)

// GrafanaConfigFromContext returns Grafana config from context.
func GrafanaConfigFromContext(ctx context.Context) *config.GrafanaCfg {
	return config.GrafanaConfigFromContext(ctx)
}

// WithGrafanaConfig injects supplied Grafana config into context.
func WithGrafanaConfig(ctx context.Context, cfg *config.GrafanaCfg) context.Context {
	return config.WithGrafanaConfig(ctx, cfg)
}

type userAgentKey struct{}

// UserAgentFromContext returns user agent from context.
func UserAgentFromContext(ctx context.Context) *useragent.UserAgent {
	v := ctx.Value(userAgentKey{})
	if v == nil {
		return useragent.Empty()
	}

	ua := v.(*useragent.UserAgent)
	if ua == nil {
		return useragent.Empty()
	}

	return ua
}

// WithUserAgent injects supplied user agent into context.
func WithUserAgent(ctx context.Context, ua *useragent.UserAgent) context.Context {
	ctx = context.WithValue(ctx, userAgentKey{}, ua)
	return ctx
}
