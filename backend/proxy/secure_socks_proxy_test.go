package proxy

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io/fs"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

func TestNewSecureSocksProxyContextDialer_SupportsFilePathAndContents(t *testing.T) {
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
	rootCACertFilePath := filepath.Join(tempDir, "ca.cert")
	// nolint:gosec
	// The gosec G304 warning can be ignored because all values come from the test
	caCertFile, err := os.Create(rootCACertFilePath)
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
	clientCertFilePath := filepath.Join(tempDir, "client.cert")
	// nolint:gosec
	// The gosec G304 warning can be ignored because all values come from the test
	certFile, err := os.Create(clientCertFilePath)
	require.NoError(t, err)
	err = pem.Encode(certFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	require.NoError(t, err)
	clientKeyFilePath := filepath.Join(tempDir, "client.key")
	// nolint:gosec
	// The gosec G304 warning can be ignored because all values come from the test
	keyFile, err := os.Create(clientKeyFilePath)
	require.NoError(t, err)
	err = pem.Encode(keyFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	})
	require.NoError(t, err)

	t.Run("Works with file paths", func(t *testing.T) {
		cli := New(&Options{
			Enabled:  true,
			Timeouts: &TimeoutOptions{Timeout: time.Duration(30), KeepAlive: time.Duration(15)},
			Auth:     &AuthOptions{Username: "user1"},
			ClientCfg: &ClientCfg{
				AllowInsecure: false,
				ClientCert:    clientCertFilePath,
				ClientKey:     clientKeyFilePath,
				RootCAs:       []string{rootCACertFilePath},
				ServerName:    "localhost",
				ProxyAddress:  "localhost:3000",
			},
		})

		dialer, err := cli.NewSecureSocksProxyContextDialer()
		assert.NotNil(t, dialer)
		assert.NoError(t, err)
	})

	t.Run("Works with file contents", func(t *testing.T) {
		clientCert, err := os.ReadFile(clientCertFilePath)
		require.NoError(t, err)
		clientKey, err := os.ReadFile(clientKeyFilePath)
		require.NoError(t, err)
		rootCA, err := os.ReadFile(rootCACertFilePath)
		require.NoError(t, err)

		cli := New(&Options{
			Enabled:  true,
			Timeouts: &TimeoutOptions{Timeout: time.Duration(30), KeepAlive: time.Duration(15)},
			Auth:     &AuthOptions{Username: "user1"},
			// No need to include the TLS config since the proxy won't use it.
			ClientCfg: &ClientCfg{
				AllowInsecure: false,
				ClientCertVal: string(clientCert),
				ClientKeyVal:  string(clientKey),
				RootCAsVals:   []string{string(rootCA)},
				ServerName:    "localhost",
				ProxyAddress:  "localhost:3000",
			},
		})

		dialer, err := cli.NewSecureSocksProxyContextDialer()
		assert.NotNil(t, dialer)
		assert.NoError(t, err)
	})
}

func TestNewSecureSocksProxy(t *testing.T) {
	opts := &Options{
		Enabled:   true,
		Timeouts:  &TimeoutOptions{Timeout: time.Duration(30), KeepAlive: time.Duration(15)},
		Auth:      &AuthOptions{Username: "user1"},
		ClientCfg: setupTestSecureSocksProxySettings(t),
	}
	cli := New(opts)

	t.Run("New socks proxy should be properly configured when all settings are valid", func(t *testing.T) {
		require.NoError(t, cli.ConfigureSecureSocksHTTPProxy(&http.Transport{}))
	})

	t.Run("Client cert must be valid", func(t *testing.T) {
		clientCertBefore := opts.ClientCfg.ClientCertVal
		opts.ClientCfg.ClientCertVal = ""
		cli = New(opts)
		t.Cleanup(func() {
			opts.ClientCfg.ClientCertVal = clientCertBefore
			cli = New(opts)
		})
		require.Error(t, cli.ConfigureSecureSocksHTTPProxy(&http.Transport{}))
	})

	t.Run("Client key must be valid", func(t *testing.T) {
		clientKeyBefore := opts.ClientCfg.ClientKeyVal
		opts.ClientCfg.ClientKeyVal = ""
		cli = New(opts)
		t.Cleanup(func() {
			opts.ClientCfg.ClientKeyVal = clientKeyBefore
			cli = New(opts)
		})
		require.Error(t, cli.ConfigureSecureSocksHTTPProxy(&http.Transport{}))
	})

	t.Run("Root CA must be not empty", func(t *testing.T) {
		rootCABefore := opts.ClientCfg.RootCAsVals
		opts.ClientCfg.RootCAsVals = []string{}
		cli = New(opts)
		t.Cleanup(func() {
			opts.ClientCfg.RootCAsVals = rootCABefore
			cli = New(opts)
		})
		require.Error(t, cli.ConfigureSecureSocksHTTPProxy(&http.Transport{}))
	})

	t.Run("Root CA must be valid", func(t *testing.T) {
		rootCABefore := opts.ClientCfg.RootCAsVals
		opts.ClientCfg.RootCAsVals = []string{""}
		cli = New(opts)
		t.Cleanup(func() {
			opts.ClientCfg.RootCAsVals = rootCABefore
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
			ProxyAddress: "localhost:8080",
			ServerName:   "testServer",
		},
	}

	t.Run("root ca must be of the type CERTIFICATE", func(t *testing.T) {
		pemStr := new(strings.Builder)
		err := pem.Encode(pemStr, &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: []byte("testing"),
		})
		require.NoError(t, err)
		opts.ClientCfg.RootCAsVals = []string{pemStr.String()}
		cli := New(opts)
		_, err = cli.NewSecureSocksProxyContextDialer()
		require.Contains(t, err.Error(), "root ca is invalid")
	})
	t.Run("root ca has to have valid content", func(t *testing.T) {
		opts.ClientCfg.RootCAsVals = []string{"this is not a pem encoded file"}
		cli := New(opts)
		_, err := cli.NewSecureSocksProxyContextDialer()
		require.Contains(t, err.Error(), "root ca is invalid")
	})

	t.Run("root ca must be of the type CERTIFICATE", func(t *testing.T) {
		rootCACert := filepath.Join(tempDir, "ca.cert")
		caCertFile, err := os.Create(rootCACert)
		require.NoError(t, err)
		err = pem.Encode(caCertFile, &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: []byte("testing"),
		})
		require.NoError(t, err)
		opts.ClientCfg.RootCAs = []string{rootCACert}
		cli := New(opts)
		_, err = cli.NewSecureSocksProxyContextDialer()
		require.Contains(t, err.Error(), "root ca is invalid")
	})

	t.Run("root ca has to have valid content", func(t *testing.T) {
		rootCACert := filepath.Join(tempDir, "ca.cert")
		err := os.WriteFile(rootCACert, []byte("this is not a pem encoded file"), fs.ModeAppend)
		require.NoError(t, err)
		opts.ClientCfg.RootCAs = []string{rootCACert}
		cli := New(opts)
		_, err = cli.NewSecureSocksProxyContextDialer()
		require.Contains(t, err.Error(), "root ca is invalid")
	})
}

func setupTestSecureSocksProxySettings(t *testing.T) *ClientCfg {
	t.Helper()
	proxyAddress := "localhost:3000"
	serverName := "localhost"

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

	caCert := new(strings.Builder)
	err = pem.Encode(caCert, &pem.Block{
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
	clientCert := new(strings.Builder)
	require.NoError(t, err)
	err = pem.Encode(clientCert, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	require.NoError(t, err)
	clientKey := new(strings.Builder)
	require.NoError(t, err)
	err = pem.Encode(clientKey, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	})
	require.NoError(t, err)

	cfg := &ClientCfg{
		ClientCertVal: clientCert.String(),
		ClientKeyVal:  clientKey.String(),
		RootCAsVals:   []string{caCert.String()},
		ServerName:    serverName,
		ProxyAddress:  proxyAddress,
	}

	return cfg
}

// fakeConn implements proxy.Dialer and proxy.ContextDialer
type fakeConn struct {
	net.Conn
	err error
}

func (fc fakeConn) Dial(_, _ string) (c net.Conn, err error) {
	if fc.err != nil {
		return nil, fc.err
	}
	return fc, nil
}

func (fc fakeConn) DialContext(_ context.Context, _, _ string) (c net.Conn, err error) {
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
