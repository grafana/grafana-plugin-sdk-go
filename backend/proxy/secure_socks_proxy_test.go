package proxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io/fs"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSecureSocksProxy(t *testing.T) {
	settings := setupTestSecureSocksProxySettings(t)

	// create empty file for testing invalid configs
	tempDir := t.TempDir()
	tempEmptyFile := filepath.Join(tempDir, "emptyfile.txt")
	// nolint:gosec
	// The gosec G304 warning can be ignored because all values come from the test
	_, err := os.Create(tempEmptyFile)
	require.NoError(t, err)

	t.Run("New socks proxy should be properly configured when all settings are valid", func(t *testing.T) {
		require.NoError(t, NewSecureSocksHTTPProxy(settings, &http.Transport{}, "uid"))
	})

	t.Run("Client cert must be valid", func(t *testing.T) {
		clientCertBefore := settings.ClientCert
		settings.ClientCert = tempEmptyFile
		t.Cleanup(func() {
			settings.ClientCert = clientCertBefore
		})
		require.Error(t, NewSecureSocksHTTPProxy(settings, &http.Transport{}, "uid"))
	})

	t.Run("Client key must be valid", func(t *testing.T) {
		clientKeyBefore := settings.ClientKey
		settings.ClientKey = tempEmptyFile
		t.Cleanup(func() {
			settings.ClientKey = clientKeyBefore
		})
		require.Error(t, NewSecureSocksHTTPProxy(settings, &http.Transport{}, "uid"))
	})

	t.Run("Root CA must be valid", func(t *testing.T) {
		rootCABefore := settings.RootCA
		settings.RootCA = tempEmptyFile
		t.Cleanup(func() {
			settings.RootCA = rootCABefore
		})
		require.Error(t, NewSecureSocksHTTPProxy(settings, &http.Transport{}, "uid"))
	})
}

func TestSecureSocksProxyEnabled(t *testing.T) {
	t.Run("Secure socks proxy should be enabled if the config says so", func(t *testing.T) {
		assert.Equal(t, true, SecureSocksProxyEnabled(&SecureSocksProxyConfig{Enabled: true}))
	})
	t.Run("Secure socks proxy should be enabled if the env variables say so", func(t *testing.T) {
		os.Setenv(PluginSecureSocksProxyEnabled, "true")
		assert.Equal(t, true, SecureSocksProxyEnabled(nil))
	})
}

func TestSecureSocksProxyConfigEnv(t *testing.T) {
	expected := SecureSocksProxyConfig{
		ClientCert:   "client.crt",
		ClientKey:    "client.key",
		RootCA:       "ca.crt",
		ProxyAddress: "localhost:8080",
		ServerName:   "testServer",
	}
	os.Setenv(PluginSecureSocksProxyClientCert, expected.ClientCert)
	os.Setenv(PluginSecureSocksProxyClientKey, expected.ClientKey)
	os.Setenv(PluginSecureSocksProxyRootCACert, expected.RootCA)
	os.Setenv(PluginSecureSocksProxyProxyAddress, expected.ProxyAddress)
	os.Setenv(PluginSecureSocksProxyServerName, expected.ServerName)

	actual, err := getConfigFromEnv()
	assert.NoError(t, err)
	assert.Equal(t, &expected, actual)
}

func TestSecureSocksProxyEnabledOnDS(t *testing.T) {
	t.Run("Secure socks proxy should only be enabled when the json data contains enableSecureSocksProxy=true", func(t *testing.T) {
		tests := []struct {
			jsonData map[string]interface{}
			enabled  bool
		}{
			{
				jsonData: map[string]interface{}{},
				enabled:  false,
			},
			{
				jsonData: map[string]interface{}{"enableSecureSocksProxy": "nonbool"},
				enabled:  false,
			},
			{
				jsonData: map[string]interface{}{"enableSecureSocksProxy": false},
				enabled:  false,
			},
			{
				jsonData: map[string]interface{}{"enableSecureSocksProxy": true},
				enabled:  true,
			},
		}

		for _, tt := range tests {
			assert.Equal(t, tt.enabled, SecureSocksProxyEnabledOnDS(tt.jsonData))
		}
	})
}

func TestPreventInvalidRootCA(t *testing.T) {
	tempDir := t.TempDir()
	t.Run("root ca must be of the type CERTIFICATE", func(t *testing.T) {
		rootCACert := filepath.Join(tempDir, "ca.cert")
		caCertFile, err := os.Create(rootCACert)
		require.NoError(t, err)
		err = pem.Encode(caCertFile, &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: []byte("testing"),
		})
		require.NoError(t, err)
		_, err = NewSecureSocksProxyContextDialer(&SecureSocksProxyConfig{RootCA: rootCACert}, "test")
		require.Contains(t, err.Error(), "root ca is invalid")
	})
	t.Run("root ca has to have valid content", func(t *testing.T) {
		rootCACert := filepath.Join(tempDir, "ca.cert")
		err := os.WriteFile(rootCACert, []byte("this is not a pem encoded file"), fs.ModeAppend)
		require.NoError(t, err)
		_, err = NewSecureSocksProxyContextDialer(&SecureSocksProxyConfig{RootCA: rootCACert}, "test")
		require.Contains(t, err.Error(), "root ca is invalid")
	})
}

func setupTestSecureSocksProxySettings(t *testing.T) *SecureSocksProxyConfig {
	proxyAddress := "localhost:3000"
	serverName := "localhost"
	tempDir := t.TempDir()

	// generate test rootCA
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization: []string{"Grafana Labs"},
			CommonName:   "Grafana",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	require.NoError(t, err)
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	require.NoError(t, err)
	rootCACert := filepath.Join(tempDir, "ca.cert")
	// nolint:gosec
	// The gosec G304 warning can be ignored because all values come from the test
	caCertFile, err := os.Create(rootCACert)
	require.NoError(t, err)
	err = pem.Encode(caCertFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})
	require.NoError(t, err)

	// generate test client cert & key
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization: []string{"Grafana Labs"},
			CommonName:   "Grafana",
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	certPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	require.NoError(t, err)
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &certPrivKey.PublicKey, caPrivKey)
	require.NoError(t, err)
	clientCert := filepath.Join(tempDir, "client.cert")
	// nolint:gosec
	// The gosec G304 warning can be ignored because all values come from the test
	certFile, err := os.Create(clientCert)
	require.NoError(t, err)
	err = pem.Encode(certFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	require.NoError(t, err)
	clientKey := filepath.Join(tempDir, "client.key")
	// nolint:gosec
	// The gosec G304 warning can be ignored because all values come from the test
	keyFile, err := os.Create(clientKey)
	require.NoError(t, err)
	err = pem.Encode(keyFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	})
	require.NoError(t, err)

	return &SecureSocksProxyConfig{
		ClientCert:   clientCert,
		ClientKey:    clientKey,
		RootCA:       rootCACert,
		ServerName:   serverName,
		ProxyAddress: proxyAddress,
	}
}
