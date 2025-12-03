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
					proxy.PluginSecureSocksProxyRootCAs:      "rootCACert",
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
						RootCAs:      []string{"rootCACert"},
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
					proxy.PluginSecureSocksProxyRootCAs:      "rootCACert",
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
					proxy.PluginSecureSocksProxyRootCAs:      "rootCACert",
				}),
				expectedFeatureToggles: FeatureToggles{},
				expectedProxy: Proxy{
					clientCfg: &proxy.ClientCfg{
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
				cfg: NewGrafanaCfg(map[string]string{
					featuretoggles.EnabledFeatures:            "",
					proxy.PluginSecureSocksProxyEnabled:       "true",
					proxy.PluginSecureSocksProxyProxyAddress:  "localhost:1234",
					proxy.PluginSecureSocksProxyServerName:    "localhost",
					proxy.PluginSecureSocksProxyClientKey:     "clientKey",
					proxy.PluginSecureSocksProxyClientCert:    "clientCert",
					proxy.PluginSecureSocksProxyRootCAs:       "rootCACert",
					proxy.PluginSecureSocksProxyAllowInsecure: "true",
				}),
				expectedFeatureToggles: FeatureToggles{},
				expectedProxy: Proxy{
					clientCfg: &proxy.ClientCfg{
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
				cfg: NewGrafanaCfg(map[string]string{
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
				expectedFeatureToggles: FeatureToggles{},
				expectedProxy: Proxy{
					clientCfg: &proxy.ClientCfg{
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

	t.Run("it should return the configured app URL from env", func(t *testing.T) {
		_ = os.Setenv(AppURL, "http://localhost-env:3000")
		defer func() { _ = os.Unsetenv(AppURL) }()
		cfg := NewGrafanaCfg(map[string]string{})
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
		cfg := NewGrafanaCfg(map[string]string{
			UserFacingDefaultError: "something failed",
		})
		v, err := cfg.UserFacingDefaultError()
		require.NoError(t, err)
		require.Equal(t, "something failed", v)
	})

	t.Run("it should return an error if the default error message is missing", func(t *testing.T) {
		cfg := NewGrafanaCfg(map[string]string{})
		_, err := cfg.UserFacingDefaultError()
		require.Error(t, err)
	})
}

func TestSql(t *testing.T) {
	t.Run("it should return the configured sql default values", func(t *testing.T) {
		cfg := NewGrafanaCfg(map[string]string{
			SQLRowLimit:                      "5",
			SQLMaxOpenConnsDefault:           "11",
			SQLMaxIdleConnsDefault:           "22",
			SQLMaxConnLifetimeSecondsDefault: "33",
		})
		v, err := cfg.SQL()
		require.NoError(t, err)
		require.Equal(t, int64(5), v.RowLimit)
		require.Equal(t, 11, v.DefaultMaxOpenConns)
		require.Equal(t, 22, v.DefaultMaxIdleConns)
		require.Equal(t, 33, v.DefaultMaxConnLifetimeSeconds)
	})

	t.Run("it should return an error if any of the defaults is missing", func(t *testing.T) {
		cfg1 := NewGrafanaCfg(map[string]string{
			SQLMaxOpenConnsDefault:           "11",
			SQLMaxIdleConnsDefault:           "22",
			SQLMaxConnLifetimeSecondsDefault: "33",
		})

		cfg2 := NewGrafanaCfg(map[string]string{
			SQLRowLimit:                      "5",
			SQLMaxIdleConnsDefault:           "22",
			SQLMaxConnLifetimeSecondsDefault: "33",
		})

		cfg3 := NewGrafanaCfg(map[string]string{
			SQLRowLimit:                      "5",
			SQLMaxOpenConnsDefault:           "11",
			SQLMaxConnLifetimeSecondsDefault: "33",
		})

		cfg4 := NewGrafanaCfg(map[string]string{
			SQLRowLimit:            "5",
			SQLMaxOpenConnsDefault: "11",
			SQLMaxIdleConnsDefault: "22",
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
		cfg1 := NewGrafanaCfg(map[string]string{
			SQLRowLimit:                      "not-an-integer",
			SQLMaxOpenConnsDefault:           "11",
			SQLMaxIdleConnsDefault:           "22",
			SQLMaxConnLifetimeSecondsDefault: "33",
		})

		cfg2 := NewGrafanaCfg(map[string]string{
			SQLRowLimit:                      "5",
			SQLMaxOpenConnsDefault:           "11",
			SQLMaxIdleConnsDefault:           "not-an-integer",
			SQLMaxConnLifetimeSecondsDefault: "33",
		})

		cfg3 := NewGrafanaCfg(map[string]string{
			SQLRowLimit:                      "5",
			SQLMaxOpenConnsDefault:           "11",
			SQLMaxIdleConnsDefault:           "not-an-integer",
			SQLMaxConnLifetimeSecondsDefault: "33",
		})

		cfg4 := NewGrafanaCfg(map[string]string{
			SQLRowLimit:                      "5",
			SQLMaxOpenConnsDefault:           "11",
			SQLMaxIdleConnsDefault:           "22",
			SQLMaxConnLifetimeSecondsDefault: "not-an-integer",
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
		cfg := NewGrafanaCfg(map[string]string{
			AppClientSecret: "client-secret",
		})
		v, err := cfg.PluginAppClientSecret()
		require.NoError(t, err)
		require.Equal(t, "client-secret", v)
	})

	t.Run("it should return the configured PluginAppClientSecret from env", func(t *testing.T) {
		_ = os.Setenv(AppClientSecret, "client-secret")
		defer func() { _ = os.Unsetenv(AppClientSecret) }()
		cfg := NewGrafanaCfg(map[string]string{})
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

func TestGrafanaCfg_Diff(t *testing.T) {
	t.Run("both configs nil should return empty slice", func(t *testing.T) {
		var cfg1, cfg2 *GrafanaCfg
		diff := cfg1.Diff(cfg2)
		require.Empty(t, diff)
		require.NotNil(t, diff)
	})

	t.Run("one config nil should return keys from other config", func(t *testing.T) {
		cfg1 := NewGrafanaCfg(map[string]string{"key1": "value1", "key2": "value2"})
		var cfg2 *GrafanaCfg

		diff1 := cfg1.Diff(cfg2) // cfg1 has keys, cfg2 is nil
		diff2 := cfg2.Diff(cfg1) // cfg2 is nil, cfg1 has keys

		require.Len(t, diff1, 2)
		require.Contains(t, diff1, "key1")
		require.Contains(t, diff1, "key2")

		require.Len(t, diff2, 2)
		require.Contains(t, diff2, "key1")
		require.Contains(t, diff2, "key2")
	})

	t.Run("one config empty should return keys from other config", func(t *testing.T) {
		cfg1 := NewGrafanaCfg(map[string]string{"key1": "value1", "key2": "value2"})
		cfg2 := NewGrafanaCfg(map[string]string{})

		diff1 := cfg1.Diff(cfg2) // cfg1 has keys, cfg2 is empty
		diff2 := cfg2.Diff(cfg1) // cfg2 is empty, cfg1 has keys

		require.Len(t, diff1, 2)
		require.Contains(t, diff1, "key1")
		require.Contains(t, diff1, "key2")

		require.Len(t, diff2, 2)
		require.Contains(t, diff2, "key1")
		require.Contains(t, diff2, "key2")
	})

	t.Run("empty configs should return empty slice", func(t *testing.T) {
		cfg1 := NewGrafanaCfg(map[string]string{})
		cfg2 := NewGrafanaCfg(map[string]string{})

		diff := cfg1.Diff(cfg2)
		require.Empty(t, diff)
	})

	t.Run("identical configs should return empty slice", func(t *testing.T) {
		config := map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		}
		cfg1 := NewGrafanaCfg(config)
		cfg2 := NewGrafanaCfg(config)

		diff := cfg1.Diff(cfg2)
		require.Empty(t, diff)
	})

	t.Run("different values should return changed keys", func(t *testing.T) {
		cfg1 := NewGrafanaCfg(map[string]string{
			"key1": "value1",
			"key2": "value2",
		})
		cfg2 := NewGrafanaCfg(map[string]string{
			"key1": "different_value",
			"key2": "value2",
		})

		diff := cfg1.Diff(cfg2)
		require.Len(t, diff, 1)
		require.Contains(t, diff, "key1")
	})

	t.Run("added keys should be detected", func(t *testing.T) {
		cfg1 := NewGrafanaCfg(map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		})
		cfg2 := NewGrafanaCfg(map[string]string{
			"key1": "value1",
			"key2": "value2",
		})

		diff := cfg1.Diff(cfg2)
		require.Len(t, diff, 1)
		require.Contains(t, diff, "key3")
	})

	t.Run("removed keys should be detected", func(t *testing.T) {
		cfg1 := NewGrafanaCfg(map[string]string{
			"key1": "value1",
		})
		cfg2 := NewGrafanaCfg(map[string]string{
			"key1": "value1",
			"key2": "value2",
		})

		diff := cfg1.Diff(cfg2)
		require.Len(t, diff, 1)
		require.Contains(t, diff, "key2")
	})

	t.Run("multiple changes should be detected", func(t *testing.T) {
		cfg1 := NewGrafanaCfg(map[string]string{
			"unchanged": "same_value",
			"modified":  "old_value",
			"removed":   "will_be_removed",
		})
		cfg2 := NewGrafanaCfg(map[string]string{
			"unchanged": "same_value",
			"modified":  "new_value",
			"added":     "new_key",
		})

		diff := cfg1.Diff(cfg2)
		require.Len(t, diff, 3)
		require.Contains(t, diff, "modified")
		require.Contains(t, diff, "removed")
		require.Contains(t, diff, "added")
		require.NotContains(t, diff, "unchanged")
	})
}

func BenchmarkProxyHash(b *testing.B) {
	count := 0
	kBytes := randomProxyContents()
	cm := map[string]string{
		proxy.PluginSecureSocksProxyClientKeyContents: string(kBytes),
	}
	for i := 0; i < b.N; i++ {
		kBytes[88] = b64chars[mathrand.Intn(64)] //nolint:gosec
		cm[proxy.PluginSecureSocksProxyClientKeyContents] = string(kBytes)
		cfg := NewGrafanaCfg(cm)
		hash := cfg.ProxyHash()
		if hash[0] == 'a' {
			count++
		}
	}
	fmt.Printf("This should be about one in 64: %f\n", float64(count)/float64(b.N))
}
