package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"time"
)

const (
	// OpenFeatureProviderURL is the base URL of the OFREP-compatible endpoint
	// exposed by the Grafana instance hosting the plugin. OpenFeature OFREP
	// providers append /ofrep/v1/evaluate/flags[/{key}] to this URL when
	// evaluating flags.
	OpenFeatureProviderURL = "GF_INSTANCE_OPENFEATURE_PROVIDER_URL"
	// OpenFeatureProviderType is the type of OpenFeature provider the URL
	// points at, as configured on the host: "static" (the host Grafana's
	// built-in provider serving its own feature toggle configuration),
	// "features-service" or "ofrep" (a remote provider).
	OpenFeatureProviderType = "GF_INSTANCE_OPENFEATURE_PROVIDER_TYPE"
	// OpenFeatureCacheTTL is the host's advisory TTL for caching flag
	// evaluation results, expressed as an integer number of seconds so that
	// plugins in any language can parse it. A value of 0, like an absent
	// variable, means the host offers no caching advice and plugins should
	// apply their own defaults; hosts must not use 0 to demand that caching
	// be disabled.
	OpenFeatureCacheTTL = "GF_INSTANCE_OPENFEATURE_CACHE_TTL"
	// OpenFeatureContext is a JSON object of host-owned evaluation context
	// attributes (for example stackId and slug on Grafana Cloud). It is only
	// distributed through the per-request config, never as an environment
	// variable, because one plugin process may serve many tenants.
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
// exposed by the Grafana instance hosting the plugin.
//
// Each value is resolved in the following priority order:
//  1. The per-request config from the request context, set by WithGrafanaConfig
//  2. The corresponding environment variable, set at plugin process start
//
// A key present in the per-request config is authoritative even when its
// value is empty: the environment is only consulted for keys the host did
// not send at all. This keeps a multi-tenant host's per-request answer from
// being overridden by process-wide state.
//
// ContextAttrs is only distributed through the per-request config; use
// PluginContext.Namespace together with ContextAttrs to build the evaluation
// context. Before any request has arrived (for example while constructing a
// provider at plugin start-up), use OpenFeatureConfigFromEnv instead.
//
// It returns an error wrapping ErrOpenFeatureNotConfigured when the host has
// not exposed a provider URL. A more recent version of Grafana may be
// required.
func (c *GrafanaCfg) OpenFeature() (OpenFeatureConfig, error) {
	url, ok := c.config[OpenFeatureProviderURL]
	if !ok {
		// Fallback to environment variable for hosts that only provide
		// discovery at process start.
		url = os.Getenv(OpenFeatureProviderURL)
	}
	if url == "" {
		return OpenFeatureConfig{}, fmt.Errorf("%w: %s is empty or not set. A more recent version of Grafana may be required", ErrOpenFeatureNotConfigured, OpenFeatureProviderURL)
	}

	providerType, ok := c.config[OpenFeatureProviderType]
	if !ok {
		providerType = os.Getenv(OpenFeatureProviderType)
	}

	ttlString, ok := c.config[OpenFeatureCacheTTL]
	if !ok {
		ttlString = os.Getenv(OpenFeatureCacheTTL)
	}
	ttl, err := parseOpenFeatureCacheTTL(ttlString)
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
		ProviderType: providerType,
		URL:          url,
		CacheTTL:     ttl,
		ContextAttrs: contextAttrs,
	}, nil
}

// OpenFeatureConfigFromEnv returns the OpenFeature provider discovery
// configuration from environment variables. It is intended for use at plugin
// process start, before any request (and therefore any per-request config)
// has arrived. ContextAttrs is always nil since the evaluation context is
// only distributed through the per-request config.
//
// It returns an error wrapping ErrOpenFeatureNotConfigured when the host has
// not exposed a provider URL. A more recent version of Grafana may be
// required.
func OpenFeatureConfigFromEnv() (OpenFeatureConfig, error) {
	url := os.Getenv(OpenFeatureProviderURL)
	if url == "" {
		return OpenFeatureConfig{}, fmt.Errorf("%w: %s not set in environment. A more recent version of Grafana may be required", ErrOpenFeatureNotConfigured, OpenFeatureProviderURL)
	}

	ttl, err := parseOpenFeatureCacheTTL(os.Getenv(OpenFeatureCacheTTL))
	if err != nil {
		return OpenFeatureConfig{}, err
	}

	return OpenFeatureConfig{
		ProviderType: os.Getenv(OpenFeatureProviderType),
		URL:          url,
		CacheTTL:     ttl,
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
