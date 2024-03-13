package proxy

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/proxy"
)

func TestNewSecureSocksProxyContextDialerInsecureProxy(t *testing.T) {
	opts := &Options{
		Enabled:  true,
		Timeouts: &TimeoutOptions{Timeout: time.Duration(30), KeepAlive: time.Duration(15)},
		Auth:     &AuthOptions{Username: "user1"},
		// No need to include the TLS config since the proxy won't use it.
		ClientCfg: &ClientCfg{
			AllowInsecure: true,
		},
	}
	cli := New(opts)

	// No errors are expected even though the TLS config was not provided
	// because the socks proxcy won't use TLS.
	dialer, err := cli.NewSecureSocksProxyContextDialer()
	assert.NotNil(t, dialer)
	assert.NoError(t, err)
}

func TestNewSecureSocksProxy(t *testing.T) {
	opts := &Options{
		Enabled:   true,
		Timeouts:  &TimeoutOptions{Timeout: time.Duration(30), KeepAlive: time.Duration(15)},
		Auth:      &AuthOptions{Username: "user1"},
		ClientCfg: setupTestSecureSocksProxySettings(t),
	}
	cli := New(opts)

	// create empty file for testing invalid configs
	tempDir := t.TempDir()
	tempEmptyFile := filepath.Join(tempDir, "emptyfile.txt")
	// nolint:gosec
	// The gosec G304 warning can be ignored because all values come from the test
	_, err := os.Create(tempEmptyFile)
	require.NoError(t, err)

	t.Run("New socks proxy should be properly configured when all settings are valid", func(t *testing.T) {
		require.NoError(t, cli.ConfigureSecureSocksHTTPProxy(&http.Transport{}))
	})

	t.Run("Client cert must be valid", func(t *testing.T) {
		clientCertBefore := opts.ClientCfg.ClientCert
		opts.ClientCfg.ClientCert = tempEmptyFile
		cli = New(opts)
		t.Cleanup(func() {
			opts.ClientCfg.ClientCert = clientCertBefore
			cli = New(opts)
		})
		require.Error(t, cli.ConfigureSecureSocksHTTPProxy(&http.Transport{}))
	})

	t.Run("Client key must be valid", func(t *testing.T) {
		clientKeyBefore := opts.ClientCfg.ClientKey
		opts.ClientCfg.ClientKey = tempEmptyFile
		cli = New(opts)
		t.Cleanup(func() {
			opts.ClientCfg.ClientKey = clientKeyBefore
			cli = New(opts)
		})
		require.Error(t, cli.ConfigureSecureSocksHTTPProxy(&http.Transport{}))
	})

	t.Run("Root CA must be valid", func(t *testing.T) {
		rootCABefore := opts.ClientCfg.RootCA
		opts.ClientCfg.RootCA = tempEmptyFile
		cli = New(opts)
		t.Cleanup(func() {
			opts.ClientCfg.RootCA = rootCABefore
			cli = New(opts)
		})
		require.Error(t, cli.ConfigureSecureSocksHTTPProxy(&http.Transport{}))
	})
}

func TestSecureSocksProxyEnabled(t *testing.T) {
	t.Run("not enabled if Enabled field is not true", func(t *testing.T) {
		cli := New(&Options{Enabled: false})
		assert.Equal(t, false, cli.SecureSocksProxyEnabled())
	})
	t.Run("not enabled if opts is nil", func(t *testing.T) {
		cli := New(nil)
		assert.Equal(t, false, cli.SecureSocksProxyEnabled())
	})
	t.Run("enabled, if Enabled field is true", func(t *testing.T) {
		cli := New(&Options{Enabled: true})
		assert.Equal(t, true, cli.SecureSocksProxyEnabled())
	})
}

func TestGetConfigFromEnv(t *testing.T) {
	cases := []struct {
		description string
		envVars     map[string]string
		expected    *ClientCfg
	}{
		{
			description: "socks proxy not enabled, should return nil",
			envVars: map[string]string{
				PluginSecureSocksProxyEnabled:      "false",
				PluginSecureSocksProxyProxyAddress: "localhost",
				PluginSecureSocksProxyClientCert:   "cert",
				PluginSecureSocksProxyClientKey:    "key",
				PluginSecureSocksProxyRootCACert:   "root_ca",
				PluginSecureSocksProxyServerName:   "server_name",
			},
			expected: nil,
		},
		{
			description: "allowInsecure=true, should return config without tls fields filled",
			envVars: map[string]string{
				PluginSecureSocksProxyEnabled:       "true",
				PluginSecureSocksProxyProxyAddress:  "localhost",
				PluginSecureSocksProxyAllowInsecure: "true",
			},
			expected: &ClientCfg{
				ProxyAddress:  "localhost",
				AllowInsecure: true,
			},
		},
		{
			description: "allowInsecure=false, client cert is required, should return nil",
			envVars: map[string]string{
				PluginSecureSocksProxyEnabled:       "true",
				PluginSecureSocksProxyProxyAddress:  "localhost",
				PluginSecureSocksProxyAllowInsecure: "false",
			},
			expected: nil,
		},
		{
			description: "allowInsecure=false, client key is required, should return nil",
			envVars: map[string]string{
				PluginSecureSocksProxyEnabled:       "true",
				PluginSecureSocksProxyProxyAddress:  "localhost",
				PluginSecureSocksProxyAllowInsecure: "false",
				PluginSecureSocksProxyClientCert:    "cert",
			},
			expected: nil,
		},
		{
			description: "allowInsecure=false, root ca is required, should return nil",
			envVars: map[string]string{
				PluginSecureSocksProxyEnabled:       "true",
				PluginSecureSocksProxyProxyAddress:  "localhost",
				PluginSecureSocksProxyAllowInsecure: "false",
				PluginSecureSocksProxyClientCert:    "cert",
				PluginSecureSocksProxyClientKey:     "key",
			},
			expected: nil,
		},
		{
			description: "allowInsecure=false, server name is required, should return nil",
			envVars: map[string]string{
				PluginSecureSocksProxyEnabled:       "true",
				PluginSecureSocksProxyProxyAddress:  "localhost",
				PluginSecureSocksProxyAllowInsecure: "false",
				PluginSecureSocksProxyClientCert:    "cert",
				PluginSecureSocksProxyClientKey:     "key",
				PluginSecureSocksProxyRootCACert:    "root",
			},
			expected: nil,
		},
		{
			description: "allowInsecure=false, should return config with tls fields filled",
			envVars: map[string]string{
				PluginSecureSocksProxyEnabled:       "true",
				PluginSecureSocksProxyProxyAddress:  "localhost",
				PluginSecureSocksProxyAllowInsecure: "false",
				PluginSecureSocksProxyClientCert:    "cert",
				PluginSecureSocksProxyClientKey:     "key",
				PluginSecureSocksProxyRootCACert:    "root",
				PluginSecureSocksProxyServerName:    "name",
			},
			expected: &ClientCfg{
				ProxyAddress:  "localhost",
				ClientCert:    "cert",
				ClientKey:     "key",
				RootCA:        "root",
				ServerName:    "name",
				AllowInsecure: false,
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.description, func(t *testing.T) {
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}
			assert.Equal(t, tt.expected, getConfigFromEnv())
		})
	}
}

func TestSecureSocksProxyConfig(t *testing.T) {
	expected := ClientCfg{
		ProxyAddress:  "localhost:8080",
		AllowInsecure: true,
	}
	t.Setenv(PluginSecureSocksProxyEnabled, "true")
	t.Setenv(PluginSecureSocksProxyProxyAddress, expected.ProxyAddress)
	t.Setenv(PluginSecureSocksProxyAllowInsecure, fmt.Sprint(expected.AllowInsecure))

	t.Run("test env variables", func(t *testing.T) {
		assert.Equal(t, &expected, getConfigFromEnv())
	})

	t.Run("test overriding env variables", func(t *testing.T) {
		expected.ProxyAddress = "localhost:8082"
		t.Setenv(PluginSecureSocksProxyProxyAddress, expected.ProxyAddress)
		assert.Equal(t, &expected, getConfigFromEnv())
	})
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
	opts := &Options{
		Enabled:  true,
		Auth:     nil,
		Timeouts: nil,
		ClientCfg: &ClientCfg{
			ClientCert:   "client.crt",
			ClientKey:    "client.key",
			ProxyAddress: "localhost:8080",
			ServerName:   "testServer",
		},
	}

	t.Run("root ca must be of the type CERTIFICATE", func(t *testing.T) {
		rootCACert := filepath.Join(tempDir, "ca.cert")
		caCertFile, err := os.Create(rootCACert)
		require.NoError(t, err)
		err = pem.Encode(caCertFile, &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: []byte("testing"),
		})
		require.NoError(t, err)
		opts.ClientCfg.RootCA = rootCACert
		cli := New(opts)
		_, err = cli.NewSecureSocksProxyContextDialer()
		require.Contains(t, err.Error(), "root ca is invalid")
	})
	t.Run("root ca has to have valid content", func(t *testing.T) {
		rootCACert := filepath.Join(tempDir, "ca.cert")
		err := os.WriteFile(rootCACert, []byte("this is not a pem encoded file"), fs.ModeAppend)
		require.NoError(t, err)
		opts.ClientCfg.RootCA = rootCACert
		cli := New(opts)
		_, err = cli.NewSecureSocksProxyContextDialer()
		require.Contains(t, err.Error(), "root ca is invalid")
	})
}

func setupTestSecureSocksProxySettings(t *testing.T) *ClientCfg {
	t.Helper()
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

	cfg := &ClientCfg{
		ClientCert:   clientCert,
		ClientKey:    clientKey,
		RootCA:       rootCACert,
		ServerName:   serverName,
		ProxyAddress: proxyAddress,
	}

	return cfg
}

// fakeConn implements proxy.Dialer and proxy.ContextDialer
type fakeConn struct {
	net.Conn
	err error
}

func (fc fakeConn) Dial(network, addr string) (c net.Conn, err error) {
	if fc.err != nil {
		return nil, fc.err
	}
	return fc, nil
}

func (fc fakeConn) DialContext(ctx context.Context, network, addr string) (c net.Conn, err error) {
	if fc.err != nil {
		return nil, fc.err
	}
	return fc, nil
}

func (fc *fakeConn) withError(e error) {
	fc.err = e
}

func TestInstrumentedSocksDialer(t *testing.T) {
	t.Parallel()
	t.Run("returns context err if context is done", func(t *testing.T) {
		t.Parallel()
		md := fakeConn{}
		d := newInstrumentedSocksDialer(md, "name", "type")

		ctx, cancel := context.WithCancelCause(context.Background())
		cancel(errors.New("my custom error"))

		cd, ok := d.(proxy.ContextDialer)
		require.True(t, ok)

		c, err := cd.DialContext(ctx, "n", "addr")
		assert.Nil(t, c)
		assert.NotNil(t, err)
		assert.Equal(t, "context canceled", err.Error())
		assert.Equal(t, "my custom error", context.Cause(ctx).Error())
	})

	t.Run("returns conn if no error given", func(t *testing.T) {
		t.Parallel()
		md := fakeConn{}
		d := newInstrumentedSocksDialer(md, "name", "type")

		cd, ok := d.(proxy.ContextDialer)
		require.True(t, ok)

		c, err := cd.DialContext(context.Background(), "n", "addr")
		assert.Nil(t, err)
		assert.NotNil(t, c)
	})

	t.Run("returns error if dialer errors", func(t *testing.T) {
		t.Parallel()
		md := fakeConn{}
		md.withError(errors.New("custom error"))
		d := newInstrumentedSocksDialer(md, "name", "type")

		cd, ok := d.(proxy.ContextDialer)
		require.True(t, ok)

		c, err := cd.DialContext(context.Background(), "n", "addr")
		assert.Nil(t, c)
		assert.NotNil(t, err)
		assert.Equal(t, "custom error", err.Error())
	})
}
