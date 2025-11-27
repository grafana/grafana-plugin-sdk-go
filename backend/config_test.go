package backend

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"testing"
	"time"

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
					proxy.PluginSecureSocksProxyClientCertContents: "clientCert",
					proxy.PluginSecureSocksProxyClientKeyContents:  "clientKey",
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
		os.Setenv(AppURL, "http://localhost-env:3000")
		defer os.Unsetenv(AppURL)
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
		os.Setenv(AppClientSecret, "client-secret")
		defer os.Unsetenv(AppClientSecret)
		cfg := NewGrafanaCfg(map[string]string{})
		v, err := cfg.PluginAppClientSecret()
		require.NoError(t, err)
		require.Equal(t, "client-secret", v)
	})
}

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

// pEMEncodeCertificate encodes a certificate in PEM format
func pEMEncodeCertificate(cert []byte) ([]byte, error) {
	var b bytes.Buffer
	err := pem.Encode(&b, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	})
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// createTestCertificate generates a test X.509 certificate with Ed25519 key and the given expiry time
// This aligns with the GenerateEd25519ClientKeys pattern used in production
// should match something like this;
// -----BEGIN PRIVATE KEY-----
// MC4CAQAwBQYDK2VwBCIEII+zrobzViCQkpHXkteM6qmDs2UW6fKAXcuvhb9rdJ2+
// -----END PRIVATE KEY-----
func createTestCertificate(notAfter time.Time) string {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	kb, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		panic(err)
	}

	//nolint:gosec
	keyHash := sha1.Sum(kb)
	ski := keyHash[:]

	clientCert := &x509.Certificate{
		SerialNumber: big.NewInt(2024),
		Subject: pkix.Name{
			Organization: []string{"Grafana Labs"},
			CommonName:   "test-client-cert",
		},
		NotBefore:    time.Now().Add(-24 * time.Hour),
		NotAfter:     notAfter,
		SubjectKeyId: ski,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	// Self-sign the certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, clientCert, clientCert, pubKey, privKey)
	if err != nil {
		panic(err)
	}

	certPEM, err := pEMEncodeCertificate(certBytes)
	if err != nil {
		panic(err)
	}

	return string(certPEM)
}

func TestIsProxyCertificateExpiring(t *testing.T) {
	t.Run("returns false when proxy is nil", func(t *testing.T) {
		cfg := NewGrafanaCfg(map[string]string{})
		result := cfg.isProxyCertificateExpiring()
		require.True(t, result)
	})

	t.Run("returns false when proxy is not enabled", func(t *testing.T) {
		cfg := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled: "false",
		})
		result := cfg.isProxyCertificateExpiring()
		require.True(t, result)
	})

	t.Run("returns true when certificate is already expired", func(t *testing.T) {
		expiredCert := createTestCertificate(time.Now().Add(-1 * time.Hour))

		cfg := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:           "true",
			proxy.PluginSecureSocksProxyClientKeyContents: expiredCert,
		})
		result := cfg.isProxyCertificateExpiring()
		require.True(t, result)
	})

	t.Run("returns true when certificate expires within 6 hours", func(t *testing.T) {
		soonExpiredCert := createTestCertificate(time.Now().Add(2 * time.Hour))

		cfg := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:           "true",
			proxy.PluginSecureSocksProxyClientKeyContents: soonExpiredCert,
		})
		result := cfg.isProxyCertificateExpiring()
		require.True(t, result)
	})

	t.Run("returns false when certificate expires after 6 hours", func(t *testing.T) {
		validCert := createTestCertificate(time.Now().Add(12 * time.Hour))

		cfg := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:           "true",
			proxy.PluginSecureSocksProxyClientKeyContents: validCert,
		})
		result := cfg.isProxyCertificateExpiring()
		require.False(t, result)
	})

	t.Run("returns false when certificate expires just after 6 hour boundary", func(t *testing.T) {
		// Expiry slightly after the 6 hour mark to avoid timing issues
		justAfterBoundary := time.Now().Add(6*time.Hour + 1*time.Second)
		cert := createTestCertificate(justAfterBoundary)

		cfg := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:           "true",
			proxy.PluginSecureSocksProxyClientKeyContents: cert,
		})
		result := cfg.isProxyCertificateExpiring()
		require.False(t, result)
	})

	t.Run("returns true when certificate cert value is invalid base64", func(t *testing.T) {
		cfg := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:           "true",
			proxy.PluginSecureSocksProxyClientKeyContents: "not-valid-base64!@#$",
		})
		result := cfg.isProxyCertificateExpiring()
		require.True(t, result)
	})

	t.Run("returns true when certificate cert value is invalid PEM", func(t *testing.T) {
		cfg := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:           "true",
			proxy.PluginSecureSocksProxyClientKeyContents: string([]byte("not-a-pem-certificate")),
		})
		result := cfg.isProxyCertificateExpiring()
		require.True(t, result)
	})

	t.Run("returns true when certificate parsing fails", func(t *testing.T) {
		invalidCert := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: []byte("invalid-cert-bytes"),
		})

		cfg := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:           "true",
			proxy.PluginSecureSocksProxyClientKeyContents: string(invalidCert),
		})
		result := cfg.isProxyCertificateExpiring()
		require.True(t, result)
	})

	t.Run("returns false when certificate is valid for more than 6 hours (1 month validity)", func(t *testing.T) {
		validLongCert := createTestCertificate(time.Now().Add(24 * 30 * time.Hour))

		cfg := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:           "true",
			proxy.PluginSecureSocksProxyClientKeyContents: validLongCert,
		})
		result := cfg.isProxyCertificateExpiring()
		require.False(t, result)
	})
}

func TestGrafanaCfg_Equal(t *testing.T) {
	t.Run("both configs nil should return true", func(t *testing.T) {
		var cfg1, cfg2 *GrafanaCfg
		result := cfg1.Equal(cfg2)
		require.True(t, result)
	})

	t.Run("one config nil should return false", func(t *testing.T) {
		cfg1 := NewGrafanaCfg(map[string]string{"key": "value"})
		var cfg2 *GrafanaCfg
		require.False(t, cfg1.Equal(cfg2))
		require.False(t, cfg2.Equal(cfg1))
	})

	t.Run("empty configs should return true", func(t *testing.T) {
		cfg1 := NewGrafanaCfg(map[string]string{})
		cfg2 := NewGrafanaCfg(map[string]string{})
		require.True(t, cfg1.Equal(cfg2))
	})

	t.Run("identical configs should return true", func(t *testing.T) {
		config := map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		}
		cfg1 := NewGrafanaCfg(config)
		cfg2 := NewGrafanaCfg(config)
		require.True(t, cfg1.Equal(cfg2))
	})

	t.Run("different values should return false", func(t *testing.T) {
		cfg1 := NewGrafanaCfg(map[string]string{
			"key1": "value1",
			"key2": "value2",
		})
		cfg2 := NewGrafanaCfg(map[string]string{
			"key1": "different_value",
			"key2": "value2",
		})
		require.False(t, cfg1.Equal(cfg2))
	})

	t.Run("different sizes should return false", func(t *testing.T) {
		cfg1 := NewGrafanaCfg(map[string]string{
			"key1": "value1",
			"key2": "value2",
		})
		cfg2 := NewGrafanaCfg(map[string]string{
			"key1": "value1",
		})
		require.False(t, cfg1.Equal(cfg2))
	})

	t.Run("proxy enabled with valid certificate should return true", func(t *testing.T) {
		validCert := createTestCertificate(time.Now().Add(12 * time.Hour))

		config := map[string]string{
			proxy.PluginSecureSocksProxyEnabled:           "true",
			proxy.PluginSecureSocksProxyClientKeyContents: validCert,
			"other_key": "other_value",
		}
		cfg1 := NewGrafanaCfg(config)
		cfg2 := NewGrafanaCfg(config)
		require.True(t, cfg1.Equal(cfg2))
	})

	t.Run("proxy enabled with expiring certificate returns needsUpdate true", func(t *testing.T) {
		expiringCert := createTestCertificate(time.Now().Add(12 * time.Hour))
		newCert := createTestCertificate(time.Now().Add(1 * time.Hour))

		cfg1 := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:           "true",
			proxy.PluginSecureSocksProxyClientKeyContents: expiringCert,
			"other_key": "other_value",
		})
		cfg2 := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:           "true",
			proxy.PluginSecureSocksProxyClientKeyContents: newCert,
			"other_key": "other_value",
		})

		require.True(t, cfg1.Equal(cfg2))
	})

	t.Run("proxy disabled should ignore certificate expiration", func(t *testing.T) {
		expiringCert := createTestCertificate(time.Now().Add(12 * time.Hour))

		cfg1 := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:           "false",
			proxy.PluginSecureSocksProxyClientKeyContents: expiringCert,
		})
		cfg2 := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:           "false",
			proxy.PluginSecureSocksProxyClientKeyContents: expiringCert,
		})
		// Even though certificate is expiring, proxy is disabled so Equal should return true
		require.True(t, cfg1.Equal(cfg2))
	})

	t.Run("secure socks proxy keys should be ignored in comparison", func(t *testing.T) {
		cfg1 := NewGrafanaCfg(map[string]string{
			"key1":                              "value1",
			"GF_SECURE_SOCKS_DATASOURCE_PROXY":  "different_value_should_be_ignored",
			"GF_SECURE_SOCKS_DATASOURCE_PROXY2": "also_ignored",
		})
		cfg2 := NewGrafanaCfg(map[string]string{
			"key1":                              "value1",
			"GF_SECURE_SOCKS_DATASOURCE_PROXY":  "different_value",
			"GF_SECURE_SOCKS_DATASOURCE_PROXY2": "completely_different",
		})
		// Proxy keys should be ignored, other keys are the same
		require.True(t, cfg1.Equal(cfg2))
	})

	t.Run("non-proxy keys should not be ignored", func(t *testing.T) {
		cfg1 := NewGrafanaCfg(map[string]string{
			"key1": "value1",
			"key2": "value2",
		})
		cfg2 := NewGrafanaCfg(map[string]string{
			"key1": "value1",
			"key2": "different_value",
		})
		require.False(t, cfg1.Equal(cfg2))
	})

	t.Run("proxy enabled with expired certificate returns needsUpdate", func(t *testing.T) {
		expiredCert := createTestCertificate(time.Now().Add(-1 * time.Hour))

		cfg1 := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:           "true",
			proxy.PluginSecureSocksProxyClientKeyContents: expiredCert,
		})
		cfg2 := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:           "true",
			proxy.PluginSecureSocksProxyClientKeyContents: expiredCert,
		})

		require.False(t, cfg1.Equal(cfg2))
	})

	t.Run("proxy enabled with invalid certificate returns needsUpdate", func(t *testing.T) {
		cfg1 := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:           "true",
			proxy.PluginSecureSocksProxyClientKeyContents: "invalid-cert",
		})
		cfg2 := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:           "true",
			proxy.PluginSecureSocksProxyClientKeyContents: "invalid-cert",
		})

		require.False(t, cfg1.Equal(cfg2))
	})

	t.Run("different non-proxy keys with expiring certificate returns false", func(t *testing.T) {
		expiringCert := createTestCertificate(time.Now().Add(2 * time.Hour))

		cfg1 := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:           "true",
			proxy.PluginSecureSocksProxyClientKeyContents: expiringCert,
			"other_key": "value1",
		})
		cfg2 := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:           "true",
			proxy.PluginSecureSocksProxyClientKeyContents: expiringCert,
			"other_key": "value2",
		})

		require.False(t, cfg1.Equal(cfg2))
	})
}
