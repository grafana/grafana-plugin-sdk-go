package config_test

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e/config"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	t.Run("should default to HAR config if config file is not found", func(t *testing.T) {
		cfg, err := config.LoadConfig("proxy.json")
		require.NoError(t, err)
		require.Equal(t, "127.0.0.1:9999", cfg.Address)
		require.Equal(t, config.StorageTypeHAR, cfg.Storage.Type)
		require.Equal(t, "fixtures/e2e.har", cfg.Storage.Path)
		require.Equal(t, []string{}, cfg.Hosts)
	})

	t.Run("should load HAR config", func(t *testing.T) {
		cfg, err := config.LoadConfig("testdata/har.json")
		require.NoError(t, err)
		require.Equal(t, "127.0.0.1:8888", cfg.Address)
		require.Equal(t, config.StorageTypeHAR, cfg.Storage.Type)
		require.Equal(t, "fixtures/test.har", cfg.Storage.Path)
		require.Equal(t, []string{"example.com", "example.org"}, cfg.Hosts)
	})
}
