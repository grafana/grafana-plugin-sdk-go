package backend

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend/proxy"
	"github.com/grafana/grafana-plugin-sdk-go/backend/useragent"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/featuretoggles"
)

func TestConfig(t *testing.T) {
	t.Run("GrafanaConfigFromContext", func(t *testing.T) {
		tcs := []struct {
			name                   string
			cfg                    *GrafanaCfg
			expectedFeatureToggles FeatureToggles
			expectedProxy          Proxy
		}{
			{
				name:                   "nil config",
				cfg:                    nil,
				expectedFeatureToggles: FeatureToggles{},
				expectedProxy:          Proxy{},
			},
			{
				name:                   "empty config",
				cfg:                    &GrafanaCfg{},
				expectedFeatureToggles: FeatureToggles{},
				expectedProxy:          Proxy{},
			},
			{
				name:                   "nil config map",
				cfg:                    NewGrafanaCfg(nil),
				expectedFeatureToggles: FeatureToggles{},
				expectedProxy:          Proxy{},
			},
			{
				name:                   "empty config map",
				cfg:                    NewGrafanaCfg(make(map[string]string)),
				expectedFeatureToggles: FeatureToggles{},
				expectedProxy:          Proxy{},
			},
			{
				name: "feature toggles and proxy enabled",
				cfg: NewGrafanaCfg(map[string]string{
					featuretoggles.EnabledFeatures:           "TestFeature",
					proxy.PluginSecureSocksProxyEnabled:      "true",
					proxy.PluginSecureSocksProxyProxyAddress: "localhost:1234",
					proxy.PluginSecureSocksProxyServerName:   "localhost",
					proxy.PluginSecureSocksProxyClientKey:    "clientKey",
					proxy.PluginSecureSocksProxyClientCert:   "clientCert",
					proxy.PluginSecureSocksProxyRootCACert:   "rootCACert",
				}),
				expectedFeatureToggles: FeatureToggles{
					enabled: map[string]struct{}{
						"TestFeature": {},
					},
				},
				expectedProxy: Proxy{
					clientCfg: &proxy.ClientCfg{
						ClientCert:   "clientCert",
						ClientKey:    "clientKey",
						RootCA:       "rootCACert",
						ProxyAddress: "localhost:1234",
						ServerName:   "localhost",
					},
				},
			},
			{
				name: "feature toggles enabled and proxy disabled",
				cfg: NewGrafanaCfg(map[string]string{
					featuretoggles.EnabledFeatures:           "TestFeature",
					proxy.PluginSecureSocksProxyEnabled:      "false",
					proxy.PluginSecureSocksProxyProxyAddress: "localhost:1234",
					proxy.PluginSecureSocksProxyServerName:   "localhost",
					proxy.PluginSecureSocksProxyClientKey:    "clientKey",
					proxy.PluginSecureSocksProxyClientCert:   "clientCert",
					proxy.PluginSecureSocksProxyRootCACert:   "rootCACert",
				}),
				expectedFeatureToggles: FeatureToggles{
					enabled: map[string]struct{}{
						"TestFeature": {},
					},
				},
				expectedProxy: Proxy{},
			},
			{
				name: "feature toggles disabled and proxy enabled",
				cfg: NewGrafanaCfg(map[string]string{
					featuretoggles.EnabledFeatures:           "",
					proxy.PluginSecureSocksProxyEnabled:      "true",
					proxy.PluginSecureSocksProxyProxyAddress: "localhost:1234",
					proxy.PluginSecureSocksProxyServerName:   "localhost",
					proxy.PluginSecureSocksProxyClientKey:    "clientKey",
					proxy.PluginSecureSocksProxyClientCert:   "clientCert",
					proxy.PluginSecureSocksProxyRootCACert:   "rootCACert",
				}),
				expectedFeatureToggles: FeatureToggles{},
				expectedProxy: Proxy{
					clientCfg: &proxy.ClientCfg{
						ClientCert:   "clientCert",
						ClientKey:    "clientKey",
						RootCA:       "rootCACert",
						ProxyAddress: "localhost:1234",
						ServerName:   "localhost",
					},
				},
			},
			{
				name: "feature toggles disabled and insecure proxy enabled",
				cfg: NewGrafanaCfg(map[string]string{
					featuretoggles.EnabledFeatures:            "",
					proxy.PluginSecureSocksProxyEnabled:       "true",
					proxy.PluginSecureSocksProxyProxyAddress:  "localhost:1234",
					proxy.PluginSecureSocksProxyServerName:    "localhost",
					proxy.PluginSecureSocksProxyClientKey:     "clientKey",
					proxy.PluginSecureSocksProxyClientCert:    "clientCert",
					proxy.PluginSecureSocksProxyRootCACert:    "rootCACert",
					proxy.PluginSecureSocksProxyAllowInsecure: "true",
				}),
				expectedFeatureToggles: FeatureToggles{},
				expectedProxy: Proxy{
					clientCfg: &proxy.ClientCfg{
						ClientCert:    "clientCert",
						ClientKey:     "clientKey",
						RootCA:        "rootCACert",
						ProxyAddress:  "localhost:1234",
						ServerName:    "localhost",
						AllowInsecure: true,
					},
				},
			},
		}

		for _, tc := range tcs {
			ctx := WithGrafanaConfig(context.Background(), tc.cfg)
			cfg := GrafanaConfigFromContext(ctx)

			require.Equal(t, tc.expectedFeatureToggles, cfg.FeatureToggles())
			proxy, err := cfg.proxy()
			assert.NoError(t, err)

			require.Equal(t, tc.expectedProxy, proxy)
		}
	})
}

func TestAppURL(t *testing.T) {
	t.Run("it should return the configured app URL", func(t *testing.T) {
		cfg := NewGrafanaCfg(map[string]string{
			AppURL: "http://localhost:3000",
		})
		url, err := cfg.AppURL()
		require.NoError(t, err)
		require.Equal(t, "http://localhost:3000", url)
	})

	t.Run("it should return an error if the app URL is missing", func(t *testing.T) {
		cfg := NewGrafanaCfg(map[string]string{})
		_, err := cfg.AppURL()
		require.Error(t, err)
	})
}

func TestConcurrentQueryCount(t *testing.T) {
	t.Run("it should return the configured concurrent query count", func(t *testing.T) {
		cfg := NewGrafanaCfg(map[string]string{
			ConcurrentQueryCount: "23",
		})
		concurrentQueryCount, err := cfg.ConcurrentQueryCount()
		require.NoError(t, err)
		require.Equal(t, 23, concurrentQueryCount)
	})

	t.Run("it should return an error if the concurrent query count is missing", func(t *testing.T) {
		cfg := NewGrafanaCfg(map[string]string{})
		concurrentQueryCount, err := cfg.ConcurrentQueryCount()
		require.Error(t, err)
		require.Equal(t, 0, concurrentQueryCount)
	})
}

func TestResponseLimit(t *testing.T) {
	t.Run("it should return the configured response limit", func(t *testing.T) {
		cfg := NewGrafanaCfg(map[string]string{
			ResponseLimit: "42",
		})
		responseLimit, isLimited := cfg.ResponseLimit()
		require.Equal(t, true, isLimited)
		require.Equal(t, int64(42), responseLimit)
	})

	t.Run("it should return false if response limit is not configured", func(t *testing.T) {
		cfg := NewGrafanaCfg(map[string]string{})
		responseLimit, isLimited := cfg.ResponseLimit()
		require.Equal(t, false, isLimited)
		require.Equal(t, int64(0), responseLimit)
	})

	t.Run("it should return false if response limit is badly configured", func(t *testing.T) {
		cfg := NewGrafanaCfg(map[string]string{
			ResponseLimit: "42a",
		})
		responseLimit, isLimited := cfg.ResponseLimit()
		require.Equal(t, false, isLimited)
		require.Equal(t, int64(0), responseLimit)
	})
}

func TestUserAgentFromContext(t *testing.T) {
	ua, err := useragent.New("10.0.0", "test", "test")
	require.NoError(t, err)

	ctx := WithUserAgent(context.Background(), ua)
	result := UserAgentFromContext(ctx)

	require.Equal(t, "10.0.0", result.GrafanaVersion())
	require.Equal(t, "Grafana/10.0.0 (test; test)", result.String())
}

func TestUserAgentFromContext_NoUserAgent(t *testing.T) {
	ctx := context.Background()

	result := UserAgentFromContext(ctx)
	require.Equal(t, "0.0.0", result.GrafanaVersion())
	require.Equal(t, "Grafana/0.0.0 (unknown; unknown)", result.String())
}
