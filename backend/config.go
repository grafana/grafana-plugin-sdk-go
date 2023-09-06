package backend

import (
	"github.com/grafana/grafana-plugin-sdk-go/experimental/featuretoggles"
	"strings"
)

const GrafanaVersion = "GF_VERSION"

type Cfg struct {
	config map[string]string
}

func (c Cfg) Get(key string) string {
	return c.config[strings.ToUpper(key)]
}

func (c Cfg) Equal(c2 Cfg) bool {
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

type FeatureToggles struct {
	// flags is a set-like map of feature flags that are enabled.
	enabled map[string]struct{}
}

// IsEnabled returns true if feature f is contained in ft.enabled.
func (ft FeatureToggles) IsEnabled(f string) bool {
	_, exists := ft.enabled[f]
	return exists
}
