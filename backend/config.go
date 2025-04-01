package backend

import (
	"context"

	"github.com/grafana/grafana-plugin-sdk-go/backend/useragent"
	"github.com/grafana/grafana-plugin-sdk-go/config"
)

const (
	AppURL                           = config.AppURL
	ConcurrentQueryCount             = config.ConcurrentQueryCount
	UserFacingDefaultError           = config.UserFacingDefaultError
	SQLRowLimit                      = config.SQLRowLimit
	SQLMaxOpenConnsDefault           = config.SQLMaxOpenConnsDefault
	SQLMaxIdleConnsDefault           = config.SQLMaxIdleConnsDefault
	SQLMaxConnLifetimeSecondsDefault = config.SQLMaxConnLifetimeSecondsDefault
	ResponseLimit                    = config.ResponseLimit
	AppClientSecret                  = config.AppClientSecret
)

// Deprecated: Use the config package instead.
type GrafanaCfg = config.GrafanaCfg

// Deprecated: Use the config package instead.
type FeatureToggles = config.FeatureToggles

// Deprecated: Use the config package instead.
func NewGrafanaCfg(m map[string]string) *config.GrafanaCfg {
	return config.NewGrafanaCfg(m)
}

// Deprecated: Use the config package instead.
func GrafanaConfigFromContext(ctx context.Context) *config.GrafanaCfg {
	return config.GrafanaConfigFromContext(ctx)
}

// Deprecated: Use the config package instead.
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
