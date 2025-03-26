package config

import (
	"context"
	"strconv"
)

const (
	ResponseLimit = "GF_RESPONSE_LIMIT"
)

// GrafanaCfg represents Grafana configuration
type GrafanaCfg struct {
	config map[string]string
}

// NewGrafanaCfg creates a new GrafanaCfg instance
func NewGrafanaCfg(cfg map[string]string) *GrafanaCfg {
	return &GrafanaCfg{config: cfg}
}

// Get returns a value from the config map
func (c *GrafanaCfg) Get(key string) string {
	return c.config[key]
}

// ResponseLimit returns the response limit value
func (c *GrafanaCfg) ResponseLimit() int64 {
	if v, exists := c.config[ResponseLimit]; exists {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return 0
}

type configKey struct{}

// GrafanaConfigFromContext returns Grafana config from context
func GrafanaConfigFromContext(ctx context.Context) *GrafanaCfg {
	v := ctx.Value(configKey{})
	if v == nil {
		return NewGrafanaCfg(nil)
	}

	cfg := v.(*GrafanaCfg)
	if cfg == nil {
		return NewGrafanaCfg(nil)
	}

	return cfg
}

// WithGrafanaConfig injects supplied Grafana config into context
func WithGrafanaConfig(ctx context.Context, cfg *GrafanaCfg) context.Context {
	ctx = context.WithValue(ctx, configKey{}, cfg)
	return ctx
}
