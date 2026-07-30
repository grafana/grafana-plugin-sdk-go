package config

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOpenFeature(t *testing.T) {
	t.Run("it should return the configured OpenFeature provider", func(t *testing.T) {
		cfg := NewGrafanaCfg(map[string]string{
			OpenFeatureProviderType: "ofrep",
			OpenFeatureProviderURL:  "http://localhost:3000",
			OpenFeatureCacheTTL:     "30",
			OpenFeatureContext:      `{"grafana_version":"12.3.0","namespace":"stacks-123"}`,
		})
		v, err := cfg.OpenFeature()
		require.NoError(t, err)
		require.Equal(t, "ofrep", v.ProviderType)
		require.Equal(t, "http://localhost:3000", v.URL)
		require.Equal(t, 30*time.Second, v.CacheTTL)
		require.Equal(t, map[string]string{"grafana_version": "12.3.0", "namespace": "stacks-123"}, v.ContextAttrs)
	})

	t.Run("it should not require the cache TTL and context attributes", func(t *testing.T) {
		cfg := NewGrafanaCfg(map[string]string{
			OpenFeatureProviderType: "static",
			OpenFeatureProviderURL:  "http://localhost:3000",
		})
		v, err := cfg.OpenFeature()
		require.NoError(t, err)
		require.Equal(t, "static", v.ProviderType)
		require.Equal(t, "http://localhost:3000", v.URL)
		require.Zero(t, v.CacheTTL)
		require.Nil(t, v.ContextAttrs)
	})

	t.Run("it should fall back to environment variables when not set in config", func(t *testing.T) {
		t.Setenv(OpenFeatureProviderType, "features-service")
		t.Setenv(OpenFeatureProviderURL, "http://localhost-env:3000")
		t.Setenv(OpenFeatureCacheTTL, "60")
		cfg := NewGrafanaCfg(map[string]string{})
		v, err := cfg.OpenFeature()
		require.NoError(t, err)
		require.Equal(t, "features-service", v.ProviderType)
		require.Equal(t, "http://localhost-env:3000", v.URL)
		require.Equal(t, time.Minute, v.CacheTTL)
		require.Nil(t, v.ContextAttrs)
	})

	t.Run("it should prefer config values over environment variables", func(t *testing.T) {
		t.Setenv(OpenFeatureProviderType, "static")
		t.Setenv(OpenFeatureProviderURL, "http://localhost-env:3000")
		t.Setenv(OpenFeatureCacheTTL, "60")
		cfg := NewGrafanaCfg(map[string]string{
			OpenFeatureProviderType: "ofrep",
			OpenFeatureProviderURL:  "http://localhost:3000",
			OpenFeatureCacheTTL:     "30",
		})
		v, err := cfg.OpenFeature()
		require.NoError(t, err)
		require.Equal(t, "ofrep", v.ProviderType)
		require.Equal(t, "http://localhost:3000", v.URL)
		require.Equal(t, 30*time.Second, v.CacheTTL)
	})

	t.Run("it should return ErrOpenFeatureNotConfigured if the provider URL is missing", func(t *testing.T) {
		cfg := NewGrafanaCfg(map[string]string{})
		_, err := cfg.OpenFeature()
		require.Error(t, err)
		require.True(t, errors.Is(err, ErrOpenFeatureNotConfigured))
	})

	t.Run("it should return ErrOpenFeatureNotConfigured if the provider URL is empty", func(t *testing.T) {
		cfg := NewGrafanaCfg(map[string]string{
			OpenFeatureProviderURL: "",
		})
		_, err := cfg.OpenFeature()
		require.Error(t, err)
		require.True(t, errors.Is(err, ErrOpenFeatureNotConfigured))
	})

	t.Run("it should return an error if the cache TTL is not an integer", func(t *testing.T) {
		cfg := NewGrafanaCfg(map[string]string{
			OpenFeatureProviderURL: "http://localhost:3000",
			OpenFeatureCacheTTL:    "30s",
		})
		_, err := cfg.OpenFeature()
		require.ErrorContains(t, err, "integer number of seconds")
	})

	t.Run("it should return an error if the cache TTL is negative", func(t *testing.T) {
		cfg := NewGrafanaCfg(map[string]string{
			OpenFeatureProviderURL: "http://localhost:3000",
			OpenFeatureCacheTTL:    "-1",
		})
		_, err := cfg.OpenFeature()
		require.ErrorContains(t, err, "non-negative")
	})

	t.Run("it should return an error if the cache TTL does not fit in a duration", func(t *testing.T) {
		cfg := NewGrafanaCfg(map[string]string{
			OpenFeatureProviderURL: "http://localhost:3000",
			OpenFeatureCacheTTL:    "10000000000",
		})
		_, err := cfg.OpenFeature()
		require.ErrorContains(t, err, "does not fit in a time.Duration")
	})

	t.Run("it should not consult the environment for keys present but empty in config", func(t *testing.T) {
		t.Setenv(OpenFeatureProviderURL, "http://localhost-env:3000")
		cfg := NewGrafanaCfg(map[string]string{
			OpenFeatureProviderURL: "",
		})
		_, err := cfg.OpenFeature()
		require.Error(t, err)
		require.True(t, errors.Is(err, ErrOpenFeatureNotConfigured))
	})

	t.Run("it should return an error if the context is not a JSON object of strings", func(t *testing.T) {
		for _, v := range []string{"not-json", `{"stackId":123}`, `["a","b"]`} {
			cfg := NewGrafanaCfg(map[string]string{
				OpenFeatureProviderURL: "http://localhost:3000",
				OpenFeatureContext:     v,
			})
			_, err := cfg.OpenFeature()
			require.ErrorContains(t, err, "JSON object")
		}
	})

	t.Run("it should not read context attributes from the environment", func(t *testing.T) {
		t.Setenv(OpenFeatureContext, `{"stackId":"123"}`)
		cfg := NewGrafanaCfg(map[string]string{
			OpenFeatureProviderURL: "http://localhost:3000",
		})
		v, err := cfg.OpenFeature()
		require.NoError(t, err)
		require.Nil(t, v.ContextAttrs)
	})
}

func TestOpenFeatureConfigFromEnv(t *testing.T) {
	t.Run("it should return the provider configured in environment variables", func(t *testing.T) {
		t.Setenv(OpenFeatureProviderType, "static")
		t.Setenv(OpenFeatureProviderURL, "http://localhost:3000")
		t.Setenv(OpenFeatureCacheTTL, "30")
		v, err := OpenFeatureConfigFromEnv()
		require.NoError(t, err)
		require.Equal(t, "static", v.ProviderType)
		require.Equal(t, "http://localhost:3000", v.URL)
		require.Equal(t, 30*time.Second, v.CacheTTL)
		require.Nil(t, v.ContextAttrs)
	})

	t.Run("it should return ErrOpenFeatureNotConfigured if the provider URL is missing", func(t *testing.T) {
		t.Setenv(OpenFeatureProviderURL, "")
		_, err := OpenFeatureConfigFromEnv()
		require.Error(t, err)
		require.True(t, errors.Is(err, ErrOpenFeatureNotConfigured))
	})

	t.Run("it should return an error if the cache TTL is not an integer", func(t *testing.T) {
		t.Setenv(OpenFeatureProviderURL, "http://localhost:3000")
		t.Setenv(OpenFeatureCacheTTL, "1m")
		_, err := OpenFeatureConfigFromEnv()
		require.ErrorContains(t, err, "integer number of seconds")
	})
}
