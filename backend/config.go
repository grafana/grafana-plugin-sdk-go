package backend

import (
	"context"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/featuretoggles"
)

type configKey struct{}

// ConfigFromContext returns config from context.
func ConfigFromContext(ctx context.Context) *Cfg {
	v := ctx.Value(configKey{})
	if v == nil {
		return NewCfg(nil)
	}

	return v.(*Cfg)
}

// contextWithConfig injects supplied config into context.
func contextWithConfig(ctx context.Context, cfg *Cfg) context.Context {
	ctx = context.WithValue(ctx, configKey{}, cfg)
	return ctx
}

type Cfg struct {
	config map[string]string
}

func NewCfg(cfg map[string]string) *Cfg {
	return &Cfg{config: cfg}
}

func (c *Cfg) Get(key string) string {
	return c.config[key]
}

func (c *Cfg) FeatureToggles() FeatureToggles {
	features, exists := c.config[featuretoggles.EnabledFeatures]
	if !exists {
		return FeatureToggles{}
	}

	fs := strings.Split(features, ",")
	enabledFeatures := make(map[string]struct{}, len(fs))
	for _, f := range fs {
		enabledFeatures[f] = struct{}{}
	}

	// TODO fallback to legacy env var

	return FeatureToggles{
		enabled: enabledFeatures,
	}
}

func (c *Cfg) Equal(c2 *Cfg) bool {
	if c == nil && c2 == nil {
		return true
	}
	if c == nil || c2 == nil {
		return false
	}

	if len(c.config) != len(c2.config) {
		return false
	}
	for k, v1 := range c.config {
		if v2, ok := c2.config[k]; !ok || v1 != v2 {
			return false
		}
	}
	return true
}

type FeatureToggles struct {
	// enabled is a set-like map of feature flags that are enabled.
	enabled map[string]struct{}
}

// IsEnabled returns true if feature f is contained in ft.enabled.
func (ft FeatureToggles) IsEnabled(f string) bool {
	_, exists := ft.enabled[f]
	return exists
}
