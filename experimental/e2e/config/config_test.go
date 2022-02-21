package config_test

import (
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e/config"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	t.Run("should load config", func(t *testing.T) {
		cfg, err := config.LoadConfig("testdata/proxy.json")
		require.NoError(t, err)
		require.Equal(t, "127.0.0.1:9999", cfg.Address)
		require.Equal(t, "fixtures/e2e.har", cfg.Path)
		require.Equal(t, []string{"example.com", "example.org"}, cfg.Hosts)
	})
}
