package backend

import (
	"context"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	mathrand "math/rand"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/backend/proxy"
	"github.com/grafana/grafana-plugin-sdk-go/backend/useragent"
	"github.com/grafana/grafana-plugin-sdk-go/config"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/featuretoggles"
)

func TestConfig(t *testing.T) {
	t.Run("GrafanaConfigFromContext", func(t *testing.T) {
		tcs := []struct {
			name                   string
			cfg                    *config.GrafanaCfg
			expectedFeatureToggles config.FeatureToggles
			expectedProxy          config.Proxy
		}{
			{
				name:                   "nil config",
				cfg:                    nil,
				expectedFeatureToggles: config.NewGrafanaCfg(nil).FeatureToggles(),
				expectedProxy:          config.Proxy{},
			},
			{
				name:                   "empty config",
				cfg:                    &config.GrafanaCfg{},
				expectedFeatureToggles: config.NewGrafanaCfg(nil).FeatureToggles(),
				expectedProxy:          config.Proxy{},
			},
			{
				name:                   "nil config map",
				cfg:                    config.NewGrafanaCfg(nil),
				expectedFeatureToggles: config.NewGrafanaCfg(nil).FeatureToggles(),
				expectedProxy:          config.Proxy{},
			},
			{
				name:                   "empty config map",
				cfg:                    config.NewGrafanaCfg(make(map[string]string)),
				expectedFeatureToggles: config.NewGrafanaCfg(nil).FeatureToggles(),
				expectedProxy:          config.Proxy{},
			},
			{
				name: "feature toggles and proxy enabled",
				cfg: config.NewGrafanaCfg(map[string]string{
					featuretoggles.EnabledFeatures:           "TestFeature",
					proxy.PluginSecureSocksProxyEnabled:      "true",
					proxy.PluginSecureSocksProxyProxyAddress: "localhost:1234",
					proxy.PluginSecureSocksProxyServerName:   "localhost",
					proxy.PluginSecureSocksProxyClientKey:    "clientKey",
					proxy.PluginSecureSocksProxyClientCert:   "clientCert",
					proxy.PluginSecureSocksProxyRootCAs:      "rootCACert",
				}),
				expectedFeatureToggles: config.NewGrafanaCfg(map[string]string{
					featuretoggles.EnabledFeatures: "TestFeature",
				}).FeatureToggles(),
				expectedProxy: config.Proxy{
					ClientCfg: &proxy.ClientCfg{
						ClientCert:   "clientCert",
						ClientKey:    "clientKey",
						RootCAs:      []string{"rootCACert"},
						ProxyAddress: "localhost:1234",
						ServerName:   "localhost",
					},
				},
			},
			{
				name: "feature toggles enabled and proxy disabled",
				cfg: config.NewGrafanaCfg(map[string]string{
					featuretoggles.EnabledFeatures:           "TestFeature",
					proxy.PluginSecureSocksProxyEnabled:      "false",
					proxy.PluginSecureSocksProxyProxyAddress: "localhost:1234",
					proxy.PluginSecureSocksProxyServerName:   "localhost",
					proxy.PluginSecureSocksProxyClientKey:    "clientKey",
					proxy.PluginSecureSocksProxyClientCert:   "clientCert",
					proxy.PluginSecureSocksProxyRootCAs:      "rootCACert",
				}),
				expectedFeatureToggles: config.NewGrafanaCfg(map[string]string{
					featuretoggles.EnabledFeatures: "TestFeature",
				}).FeatureToggles(),
				expectedProxy: config.Proxy{},
			},
			{
				name: "feature toggles disabled and proxy enabled",
				cfg: config.NewGrafanaCfg(map[string]string{
					featuretoggles.EnabledFeatures:           "",
					proxy.PluginSecureSocksProxyEnabled:      "true",
					proxy.PluginSecureSocksProxyProxyAddress: "localhost:1234",
					proxy.PluginSecureSocksProxyServerName:   "localhost",
					proxy.PluginSecureSocksProxyClientKey:    "clientKey",
					proxy.PluginSecureSocksProxyClientCert:   "clientCert",
					proxy.PluginSecureSocksProxyRootCAs:      "rootCACert",
				}),
				expectedFeatureToggles: config.NewGrafanaCfg(map[string]string{
					featuretoggles.EnabledFeatures: "",
				}).FeatureToggles(),
				expectedProxy: config.Proxy{
					ClientCfg: &proxy.ClientCfg{
						ClientCert:   "clientCert",
						ClientKey:    "clientKey",
						RootCAs:      []string{"rootCACert"},
						ProxyAddress: "localhost:1234",
						ServerName:   "localhost",
					},
				},
			},
			{
				name: "feature toggles disabled and insecure proxy enabled",
				cfg: config.NewGrafanaCfg(map[string]string{
					featuretoggles.EnabledFeatures:            "",
					proxy.PluginSecureSocksProxyEnabled:       "true",
					proxy.PluginSecureSocksProxyProxyAddress:  "localhost:1234",
					proxy.PluginSecureSocksProxyServerName:    "localhost",
					proxy.PluginSecureSocksProxyClientKey:     "clientKey",
					proxy.PluginSecureSocksProxyClientCert:    "clientCert",
					proxy.PluginSecureSocksProxyRootCAs:       "rootCACert",
					proxy.PluginSecureSocksProxyAllowInsecure: "true",
				}),
				expectedFeatureToggles: config.NewGrafanaCfg(map[string]string{
					featuretoggles.EnabledFeatures: "",
				}).FeatureToggles(),
				expectedProxy: config.Proxy{
					ClientCfg: &proxy.ClientCfg{
						ClientCert:    "clientCert",
						ClientKey:     "clientKey",
						RootCAs:       []string{"rootCACert"},
						ProxyAddress:  "localhost:1234",
						ServerName:    "localhost",
						AllowInsecure: true,
					},
				},
			},
			{
				name: "feature toggles disabled and secure proxy enabled with file contents",
				cfg: config.NewGrafanaCfg(map[string]string{
					featuretoggles.EnabledFeatures:                 "",
					proxy.PluginSecureSocksProxyEnabled:            "true",
					proxy.PluginSecureSocksProxyProxyAddress:       "localhost:1234",
					proxy.PluginSecureSocksProxyServerName:         "localhost",
					proxy.PluginSecureSocksProxyClientKey:          "./clientKey",
					proxy.PluginSecureSocksProxyClientCert:         "./clientCert",
					proxy.PluginSecureSocksProxyRootCAs:            "./rootCACert ./rootCACert2",
					proxy.PluginSecureSocksProxyClientKeyContents:  "clientKey",
					proxy.PluginSecureSocksProxyClientCertContents: "clientCert",
					proxy.PluginSecureSocksProxyRootCAsContents:    "rootCACert,rootCACert2",
					proxy.PluginSecureSocksProxyAllowInsecure:      "true",
				}),
				expectedFeatureToggles: config.NewGrafanaCfg(map[string]string{
					featuretoggles.EnabledFeatures: "",
				}).FeatureToggles(),
				expectedProxy: config.Proxy{
					ClientCfg: &proxy.ClientCfg{
						ClientCert:    "./clientCert",
						ClientCertVal: "clientCert",
						ClientKey:     "./clientKey",
						ClientKeyVal:  "clientKey",
						RootCAs:       []string{"./rootCACert", "./rootCACert2"},
						RootCAsVals:   []string{"rootCACert", "rootCACert2"},
						ProxyAddress:  "localhost:1234",
						ServerName:    "localhost",
						AllowInsecure: true,
					},
				},
			},
		}

		for _, tc := range tcs {
			ctx := config.WithGrafanaConfig(context.Background(), tc.cfg)
			cfg := config.GrafanaConfigFromContext(ctx)

			require.Equal(t, tc.expectedFeatureToggles, cfg.FeatureToggles())
			proxy, err := cfg.Proxy()
			assert.NoError(t, err)

			require.Equal(t, tc.expectedProxy, proxy)
		}
	})
}

func TestAppURL(t *testing.T) {
	t.Run("it should return the configured app URL", func(t *testing.T) {
		cfg := config.NewGrafanaCfg(map[string]string{
			config.AppURL: "http://localhost:3000",
		})
		url, err := cfg.AppURL()
		require.NoError(t, err)
		require.Equal(t, "http://localhost:3000", url)
	})

	t.Run("it should return an error if the app URL is missing", func(t *testing.T) {
		cfg := config.NewGrafanaCfg(map[string]string{})
		_, err := cfg.AppURL()
		require.Error(t, err)
	})

	t.Run("it should return the configured app URL from env", func(t *testing.T) {
		os.Setenv(config.AppURL, "http://localhost-env:3000")
		defer os.Unsetenv(config.AppURL)
		cfg := config.NewGrafanaCfg(map[string]string{})
		v, err := cfg.AppURL()
		require.NoError(t, err)
		require.Equal(t, "http://localhost-env:3000", v)
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

func TestUserFacingDefaultError(t *testing.T) {
	t.Run("it should return the configured default error message", func(t *testing.T) {
		cfg := config.NewGrafanaCfg(map[string]string{
			config.UserFacingDefaultError: "something failed",
		})
		v, err := cfg.UserFacingDefaultError()
		require.NoError(t, err)
		require.Equal(t, "something failed", v)
	})

	t.Run("it should return an error if the default error message is missing", func(t *testing.T) {
		cfg := config.NewGrafanaCfg(map[string]string{})
		_, err := cfg.UserFacingDefaultError()
		require.Error(t, err)
	})
}

func TestSql(t *testing.T) {
	t.Run("it should return the configured sql default values", func(t *testing.T) {
		cfg := config.NewGrafanaCfg(map[string]string{
			config.SQLRowLimit:                      "5",
			config.SQLMaxOpenConnsDefault:           "11",
			config.SQLMaxIdleConnsDefault:           "22",
			config.SQLMaxConnLifetimeSecondsDefault: "33",
		})
		v, err := cfg.SQL()
		require.NoError(t, err)
		require.Equal(t, int64(5), v.RowLimit)
		require.Equal(t, 11, v.DefaultMaxOpenConns)
		require.Equal(t, 22, v.DefaultMaxIdleConns)
		require.Equal(t, 33, v.DefaultMaxConnLifetimeSeconds)
	})

	t.Run("it should return an error if any of the defaults is missing", func(t *testing.T) {
		cfg1 := config.NewGrafanaCfg(map[string]string{
			config.SQLMaxOpenConnsDefault:           "11",
			config.SQLMaxIdleConnsDefault:           "22",
			config.SQLMaxConnLifetimeSecondsDefault: "33",
		})

		cfg2 := config.NewGrafanaCfg(map[string]string{
			config.SQLRowLimit:                      "5",
			config.SQLMaxIdleConnsDefault:           "22",
			config.SQLMaxConnLifetimeSecondsDefault: "33",
		})

		cfg3 := config.NewGrafanaCfg(map[string]string{
			config.SQLRowLimit:                      "5",
			config.SQLMaxOpenConnsDefault:           "11",
			config.SQLMaxConnLifetimeSecondsDefault: "33",
		})

		cfg4 := config.NewGrafanaCfg(map[string]string{
			config.SQLRowLimit:            "5",
			config.SQLMaxOpenConnsDefault: "11",
			config.SQLMaxIdleConnsDefault: "22",
		})

		_, err := cfg1.SQL()
		require.ErrorContains(t, err, "not set")

		_, err = cfg2.SQL()
		require.ErrorContains(t, err, "not set")

		_, err = cfg3.SQL()
		require.ErrorContains(t, err, "not set")

		_, err = cfg4.SQL()
		require.ErrorContains(t, err, "not set")
	})

	t.Run("it should return an error if any of the defaults is not an integer", func(t *testing.T) {
		cfg1 := config.NewGrafanaCfg(map[string]string{
			config.SQLRowLimit:                      "not-an-integer",
			config.SQLMaxOpenConnsDefault:           "11",
			config.SQLMaxIdleConnsDefault:           "22",
			config.SQLMaxConnLifetimeSecondsDefault: "33",
		})

		cfg2 := config.NewGrafanaCfg(map[string]string{
			config.SQLRowLimit:                      "5",
			config.SQLMaxOpenConnsDefault:           "11",
			config.SQLMaxIdleConnsDefault:           "not-an-integer",
			config.SQLMaxConnLifetimeSecondsDefault: "33",
		})

		cfg3 := config.NewGrafanaCfg(map[string]string{
			config.SQLRowLimit:                      "5",
			config.SQLMaxOpenConnsDefault:           "11",
			config.SQLMaxIdleConnsDefault:           "not-an-integer",
			config.SQLMaxConnLifetimeSecondsDefault: "33",
		})

		cfg4 := config.NewGrafanaCfg(map[string]string{
			config.SQLRowLimit:                      "5",
			config.SQLMaxOpenConnsDefault:           "11",
			config.SQLMaxIdleConnsDefault:           "22",
			config.SQLMaxConnLifetimeSecondsDefault: "not-an-integer",
		})

		_, err := cfg1.SQL()
		require.ErrorContains(t, err, "not a valid integer")

		_, err = cfg2.SQL()
		require.ErrorContains(t, err, "not a valid integer")

		_, err = cfg3.SQL()
		require.ErrorContains(t, err, "not a valid integer")

		_, err = cfg4.SQL()
		require.ErrorContains(t, err, "not a valid integer")
	})
}

func TestPluginAppClientSecret(t *testing.T) {
	t.Run("it should return the configured PluginAppClientSecret", func(t *testing.T) {
		cfg := config.NewGrafanaCfg(map[string]string{
			config.AppClientSecret: "client-secret",
		})
		v, err := cfg.PluginAppClientSecret()
		require.NoError(t, err)
		require.Equal(t, "client-secret", v)
	})

	t.Run("it should return the configured PluginAppClientSecret from env", func(t *testing.T) {
		os.Setenv(config.AppClientSecret, "client-secret")
		defer os.Unsetenv(config.AppClientSecret)
		cfg := config.NewGrafanaCfg(map[string]string{})
		v, err := cfg.PluginAppClientSecret()
		require.NoError(t, err)
		require.Equal(t, "client-secret", v)
	})
}

func randomProxyContents() []byte {
	key := make([]byte, 48)
	_, _ = rand.Read(key)
	pb := pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: key,
	}
	return pem.EncodeToMemory(&pb)
}

var b64chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789/+"

func BenchmarkProxyHash(b *testing.B) {
	count := 0
	kBytes := randomProxyContents()
	cm := map[string]string{
		proxy.PluginSecureSocksProxyClientKeyContents: string(kBytes),
	}
	for i := 0; i < b.N; i++ {
		kBytes[88] = b64chars[mathrand.Intn(64)] //nolint:gosec
		cm[proxy.PluginSecureSocksProxyClientKeyContents] = string(kBytes)
		cfg := config.NewGrafanaCfg(cm)
		hash := cfg.ProxyHash()
		if hash[0] == 'a' {
			count++
		}
	}
	fmt.Printf("This should be about one in 64: %f\n", float64(count)/float64(b.N))
}
