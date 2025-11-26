package backend

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
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

// pEMEncodeEd25519PrivateKey encodes an Ed25519 private key in PEM format
func pEMEncodeEd25519PrivateKey(key crypto.PrivateKey) ([]byte, error) {
	var b bytes.Buffer
	kb, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, err
	}
	err = pem.Encode(&b, &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: kb,
	})
	if err != nil {
		return nil, err
	}
	return b.Bytes(), err
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
// Returns: (certPEM []byte, keyPEM []byte, error)
func createTestCertificate(notAfter time.Time) ([]byte, []byte, error) {
	// Generate Ed25519 key pair
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	// Generate SubjectKeyId by taking the SHA-1 hash of the ASN.1 encoding of the public key
	// This follows RFC 5280 section 4.2.1.2
	kb, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return nil, nil, err
	}

	//nolint:gosec
	keyHash := sha1.Sum(kb)
	ski := keyHash[:]

	// Create certificate with realistic fields matching production pattern
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
		return nil, nil, err
	}

	// Encode both certificate and private key in PEM format using utility functions
	certPEM, err := pEMEncodeCertificate(certBytes)
	if err != nil {
		return nil, nil, err
	}

	keyPEM, err := pEMEncodeEd25519PrivateKey(privKey)
	if err != nil {
		return nil, nil, err
	}

	return certPEM, keyPEM, nil
}

func TestshouldRefreshProxyClientCert(t *testing.T) {
	t.Run("returns false when proxy is nil", func(t *testing.T) {
		cfg := NewGrafanaCfg(map[string]string{})
		result := cfg.shouldRefreshProxyClientCert()
		require.True(t, result)
	})

	t.Run("returns false when proxy is not enabled", func(t *testing.T) {
		cfg := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled: "false",
		})
		result := cfg.shouldRefreshProxyClientCert()
		require.True(t, result)
	})

	t.Run("returns true when certificate is already expired", func(t *testing.T) {
		expiredCert, _, err := createTestCertificate(time.Now().Add(-1 * time.Hour))
		require.NoError(t, err)

		cfg := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:            "true",
			proxy.PluginSecureSocksProxyClientCertContents: base64.StdEncoding.EncodeToString(expiredCert),
		})
		result := cfg.shouldRefreshProxyClientCert()
		require.True(t, result)
	})

	t.Run("returns true when certificate expires within 6 hours", func(t *testing.T) {
		soonExpiredCert, _, err := createTestCertificate(time.Now().Add(2 * time.Hour))
		require.NoError(t, err)

		cfg := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:            "true",
			proxy.PluginSecureSocksProxyClientCertContents: base64.StdEncoding.EncodeToString(soonExpiredCert),
		})
		result := cfg.shouldRefreshProxyClientCert()
		require.True(t, result)
	})

	t.Run("returns false when certificate expires after 6 hours", func(t *testing.T) {
		validCert, _, err := createTestCertificate(time.Now().Add(12 * time.Hour))
		require.NoError(t, err)

		cfg := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:            "true",
			proxy.PluginSecureSocksProxyClientCertContents: base64.StdEncoding.EncodeToString(validCert),
		})
		result := cfg.shouldRefreshProxyClientCert()
		require.False(t, result)
	})

	t.Run("returns false when certificate expires just after 6 hour boundary", func(t *testing.T) {
		// Expiry slightly after the 6 hour mark to avoid timing issues
		justAfterBoundary := time.Now().Add(6*time.Hour + 1*time.Second)
		cert, _, err := createTestCertificate(justAfterBoundary)
		require.NoError(t, err)

		cfg := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:            "true",
			proxy.PluginSecureSocksProxyClientCertContents: base64.StdEncoding.EncodeToString(cert),
		})
		result := cfg.shouldRefreshProxyClientCert()
		// After the 6 hour boundary, should return false
		require.False(t, result)
	})

	t.Run("returns true when certificate cert value is invalid base64", func(t *testing.T) {
		cfg := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:            "true",
			proxy.PluginSecureSocksProxyClientCertContents: "not-valid-base64!@#$",
		})
		result := cfg.shouldRefreshProxyClientCert()
		require.True(t, result)
	})

	t.Run("returns true when certificate cert value is invalid PEM", func(t *testing.T) {
		cfg := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:            "true",
			proxy.PluginSecureSocksProxyClientCertContents: base64.StdEncoding.EncodeToString([]byte("not-a-pem-certificate")),
		})
		result := cfg.shouldRefreshProxyClientCert()
		require.True(t, result)
	})

	t.Run("returns true when certificate parsing fails", func(t *testing.T) {
		invalidCert := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: []byte("invalid-cert-bytes"),
		})

		cfg := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:            "true",
			proxy.PluginSecureSocksProxyClientCertContents: base64.StdEncoding.EncodeToString(invalidCert),
		})
		result := cfg.shouldRefreshProxyClientCert()
		require.True(t, result)
	})

	t.Run("returns false when certificate is valid for more than 6 hours (1 month validity)", func(t *testing.T) {
		validLongCert, _, err := createTestCertificate(time.Now().Add(24 * 30 * time.Hour))
		require.NoError(t, err)

		cfg := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:            "true",
			proxy.PluginSecureSocksProxyClientCertContents: base64.StdEncoding.EncodeToString(validLongCert),
		})
		result := cfg.shouldRefreshProxyClientCert()
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
		validCert, _, err := createTestCertificate(time.Now().Add(12 * time.Hour))
		require.NoError(t, err)

		config := map[string]string{
			proxy.PluginSecureSocksProxyEnabled:            "true",
			proxy.PluginSecureSocksProxyClientCertContents: base64.StdEncoding.EncodeToString(validCert),
			"other_key": "other_value",
		}
		cfg1 := NewGrafanaCfg(config)
		cfg2 := NewGrafanaCfg(config)
		require.True(t, cfg1.Equal(cfg2))
	})

	t.Run("proxy enabled with expiring certificate returns needsUpdate true", func(t *testing.T) {
		expiringCert, _, err := createTestCertificate(time.Now().Add(2 * time.Hour))
		require.NoError(t, err)

		cfg1 := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:            "true",
			proxy.PluginSecureSocksProxyClientCertContents: base64.StdEncoding.EncodeToString(expiringCert),
			"other_key": "other_value",
		})
		cfg2 := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:            "true",
			proxy.PluginSecureSocksProxyClientCertContents: base64.StdEncoding.EncodeToString(expiringCert),
			"other_key": "other_value",
		})
		// Since certificate is expiring, shouldRefreshProxyClientCert returns true
		// When proxy is enabled AND expiring, Equal returns needsUpdate value
		// Other keys are identical, so needsUpdate = true, Equal returns true
		require.True(t, cfg1.Equal(cfg2))
	})

	t.Run("proxy enabled with expiring certificate returns needsUpdate true", func(t *testing.T) {
		expiringCert, _, err := createTestCertificate(time.Now().Add(2 * time.Hour))
		require.NoError(t, err)
		newCert, _, err := createTestCertificate(time.Now().Add(1 * time.Hour))
		require.NoError(t, err)

		cfg1 := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:            "true",
			proxy.PluginSecureSocksProxyClientCertContents: base64.StdEncoding.EncodeToString(expiringCert),
			"other_key": "other_value",
		})
		cfg2 := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:            "true",
			proxy.PluginSecureSocksProxyClientCertContents: base64.StdEncoding.EncodeToString(newCert),
			"other_key": "other_value",
		})
		// Since certificate is expiring, shouldRefreshProxyClientCert returns true
		// When proxy is enabled AND expiring, Equal returns needsUpdate value
		// Other keys are identical, so needsUpdate = true, Equal returns true
		require.True(t, cfg1.Equal(cfg2))
	})

	t.Run("proxy disabled should ignore certificate expiration", func(t *testing.T) {
		expiringCert, _, err := createTestCertificate(time.Now().Add(2 * time.Hour))
		require.NoError(t, err)

		cfg1 := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:            "false",
			proxy.PluginSecureSocksProxyClientCertContents: base64.StdEncoding.EncodeToString(expiringCert),
		})
		cfg2 := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:            "false",
			proxy.PluginSecureSocksProxyClientCertContents: base64.StdEncoding.EncodeToString(expiringCert),
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
		expiredCert, _, err := createTestCertificate(time.Now().Add(-1 * time.Hour))
		require.NoError(t, err)

		cfg1 := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:            "true",
			proxy.PluginSecureSocksProxyClientCertContents: base64.StdEncoding.EncodeToString(expiredCert),
		})
		cfg2 := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:            "true",
			proxy.PluginSecureSocksProxyClientCertContents: base64.StdEncoding.EncodeToString(expiredCert),
		})
		// Certificate is expired, shouldRefreshProxyClientCert returns true
		// All other keys are identical, needsUpdate = true
		// Equal returns needsUpdate = true
		require.True(t, cfg1.Equal(cfg2))
	})

	t.Run("proxy enabled with invalid certificate returns needsUpdate", func(t *testing.T) {
		cfg1 := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:            "true",
			proxy.PluginSecureSocksProxyClientCertContents: "invalid-cert",
		})
		cfg2 := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:            "true",
			proxy.PluginSecureSocksProxyClientCertContents: "invalid-cert",
		})
		// Invalid certificate causes shouldRefreshProxyClientCert to return true
		// All other keys are identical, needsUpdate = true
		// Equal returns needsUpdate = true
		require.True(t, cfg1.Equal(cfg2))
	})

	t.Run("different non-proxy keys with expiring certificate returns false", func(t *testing.T) {
		expiringCert, _, err := createTestCertificate(time.Now().Add(2 * time.Hour))
		require.NoError(t, err)

		cfg1 := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:            "true",
			proxy.PluginSecureSocksProxyClientCertContents: base64.StdEncoding.EncodeToString(expiringCert),
			"other_key": "value1",
		})
		cfg2 := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:            "true",
			proxy.PluginSecureSocksProxyClientCertContents: base64.StdEncoding.EncodeToString(expiringCert),
			"other_key": "value2",
		})
		// Different non-proxy keys, so needsUpdate = false
		// Certificate is expiring, so Equal returns needsUpdate = false
		require.False(t, cfg1.Equal(cfg2))
	})

	t.Run("different non-proxy keys with valid certificate returns true", func(t *testing.T) {
		validCert, _, err := createTestCertificate(time.Now().Add(12 * time.Hour))
		require.NoError(t, err)

		cfg1 := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:            "true",
			proxy.PluginSecureSocksProxyClientCertContents: base64.StdEncoding.EncodeToString(validCert),
			"other_key": "value1",
		})
		cfg2 := NewGrafanaCfg(map[string]string{
			proxy.PluginSecureSocksProxyEnabled:            "true",
			proxy.PluginSecureSocksProxyClientCertContents: base64.StdEncoding.EncodeToString(validCert),
			"other_key": "value2",
		})
		// Different non-proxy keys, so needsUpdate = false
		// But certificate is valid and proxy is enabled
		// Equal returns true immediately (early return for valid certificate)
		// This ignores the fact that other keys differ!
		require.True(t, cfg1.Equal(cfg2))
	})
}
