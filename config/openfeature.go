package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"
)

const (
	// OpenFeatureProviderURL is the per-request config key carrying the base
	// URL of the OFREP-compatible endpoint exposed by the Grafana instance
	// hosting the plugin. OpenFeature OFREP providers append
	// /ofrep/v1/evaluate/flags[/{key}] to this URL when evaluating flags.
	OpenFeatureProviderURL = "GF_INSTANCE_OPENFEATURE_PROVIDER_URL"
	// OpenFeatureProviderType is the per-request config key carrying the type
	// of OpenFeature provider the URL points at, as configured on the host:
	// "static" (the host Grafana's built-in provider serving its own feature
	// toggle configuration), "features-service" or "ofrep" (a remote
	// provider).
	OpenFeatureProviderType = "GF_INSTANCE_OPENFEATURE_PROVIDER_TYPE"
	// OpenFeatureCacheTTL is the per-request config key carrying the host's
	// advisory TTL for caching flag evaluation results, expressed as an
	// integer number of seconds so that plugins in any language can parse it.
	// A value of 0, like an absent key, means the host offers no caching
	// advice and plugins should apply their own defaults; hosts must not use
	// 0 to demand that caching be disabled.
	OpenFeatureCacheTTL = "GF_INSTANCE_OPENFEATURE_CACHE_TTL"
	// OpenFeatureContext is the per-request config key carrying a JSON object
	// of host-owned evaluation context attributes (for example stackId and
	// slug on Grafana Cloud).
	OpenFeatureContext = "GF_INSTANCE_OPENFEATURE_CONTEXT"
)

// ErrOpenFeatureNotConfigured is returned when the Grafana instance hosting
// the plugin has not exposed an OpenFeature provider URL. Plugins should
// treat this as "discovery unavailable" and fall back to their own defaults.
var ErrOpenFeatureNotConfigured = errors.New("OpenFeature provider discovery not configured")

// OpenFeatureConfig is the OpenFeature provider discovery information exposed
// by the Grafana instance hosting the plugin. The SDK only exposes discovery:
// plugins instantiate their own OpenFeature provider and client, exactly as
// they do on the frontend.
type OpenFeatureConfig struct {
	// ProviderType is the kind of provider URL points at: "static",
	// "features-service" or "ofrep".
	ProviderType string
	// URL is the OFREP base URL. OFREP clients append
	// /ofrep/v1/evaluate/flags[/{key}] to it when evaluating flags.
	URL string
	// CacheTTL is the host's advisory TTL for caching flag evaluation
	// results. Zero means the host gave no caching advice.
	CacheTTL time.Duration
	// ContextAttrs are host-owned evaluation context attributes. Plugins
	// should merge them into their evaluation context verbatim and must not
	// override host-asserted keys.
	ContextAttrs map[string]string
}

// OpenFeature returns the OpenFeature provider discovery configuration
// exposed by the Grafana instance hosting the plugin, resolved from the
// per-request config set by WithGrafanaConfig. Discovery is therefore only
// available once a request has arrived: plugins should construct their
// OpenFeature provider lazily on first use rather than at process start.
//
// Use PluginContext.Namespace together with ContextAttrs to build the
// evaluation context.
//
// It returns an error wrapping ErrOpenFeatureNotConfigured when the host has
// not exposed a provider URL. A more recent version of Grafana may be
// required.
func (c *GrafanaCfg) OpenFeature() (OpenFeatureConfig, error) {
	url := c.config[OpenFeatureProviderURL]
	if url == "" {
		return OpenFeatureConfig{}, fmt.Errorf("%w: %s is empty or not set. A more recent version of Grafana may be required", ErrOpenFeatureNotConfigured, OpenFeatureProviderURL)
	}

	ttl, err := parseOpenFeatureCacheTTL(c.config[OpenFeatureCacheTTL])
	if err != nil {
		return OpenFeatureConfig{}, err
	}

	var contextAttrs map[string]string
	if v := c.config[OpenFeatureContext]; v != "" {
		if err := json.Unmarshal([]byte(v), &contextAttrs); err != nil {
			return OpenFeatureConfig{}, fmt.Errorf("parsing %s, value must be a JSON object of string attributes: %w", OpenFeatureContext, err)
		}
	}

	return OpenFeatureConfig{
		ProviderType: c.config[OpenFeatureProviderType],
		URL:          url,
		CacheTTL:     ttl,
		ContextAttrs: contextAttrs,
	}, nil
}

func parseOpenFeatureCacheTTL(v string) (time.Duration, error) {
	if v == "" {
		return 0, nil
	}
	secs, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing %s, value must be an integer number of seconds: %w", OpenFeatureCacheTTL, err)
	}
	if secs < 0 {
		return 0, fmt.Errorf("parsing %s, value must be a non-negative integer number of seconds", OpenFeatureCacheTTL)
	}
	if secs > math.MaxInt64/int64(time.Second) {
		return 0, fmt.Errorf("parsing %s, value %d seconds does not fit in a time.Duration", OpenFeatureCacheTTL, secs)
	}
	return time.Duration(secs) * time.Second, nil
}
