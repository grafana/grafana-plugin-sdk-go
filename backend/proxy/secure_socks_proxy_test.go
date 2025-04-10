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
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	require.NoError(t, err)

	rootCACert, rootCACertFilePath, err := createRootCA(t, caPrivKey)
	require.NoError(t, err)
	rootCaCertValue, err := os.ReadFile(rootCACertFilePath)
	require.NoError(t, err)
	clientCertContents, clientCertFilePath, err := createClientCert(t, rootCACert, caPrivKey)
	require.NoError(t, err)
	clientKeyContents, clientKeyFilePath, err := createClientKey(t, caPrivKey)
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
		cli := New(&Options{
			Enabled:  true,
			Timeouts: &TimeoutOptions{Timeout: time.Duration(30), KeepAlive: time.Duration(15)},
			Auth:     &AuthOptions{Username: "user1"},
			ClientCfg: &ClientCfg{
				AllowInsecure: false,
				ClientCertVal: clientCertContents,
				ClientKeyVal:  clientKeyContents,
				RootCAsVals:   []string{string(rootCaCertValue)},
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

	t.Run("New socks proxy should fail with expect error with using http.DefaultTransport", func(t *testing.T) {
		defaultHTTPTransport, ok := http.DefaultTransport.(*http.Transport)
		require.True(t, ok)
		err := cli.ConfigureSecureSocksHTTPProxy(defaultHTTPTransport)
		require.Equal(t, errUseOfHTTPDefaultTransport, err)
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
	t.Run("not enabled if opts.ClientCfg is nil", func(t *testing.T) {
		cli := New(&Options{Enabled: true})
		assert.Equal(t, false, cli.SecureSocksProxyEnabled())
	})
	t.Run("enabled, if Enabled field is true", func(t *testing.T) {
		cli := New(&Options{Enabled: true, ClientCfg: &ClientCfg{}})
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

	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	require.NoError(t, err)

	rootCACert, rootCACertFilePath, err := createRootCA(t, caPrivKey)
	require.NoError(t, err)
	rootCaCertValue, err := os.ReadFile(rootCACertFilePath)
	require.NoError(t, err)

	clientCertValue, _, err := createClientCert(t, rootCACert, caPrivKey)
	require.NoError(t, err)
	clientKeyValue, _, err := createClientKey(t, caPrivKey)
	require.NoError(t, err)

	return &ClientCfg{
		ClientCertVal: clientCertValue,
		ClientKeyVal:  clientKeyValue,
		RootCAsVals:   []string{string(rootCaCertValue)},
		ServerName:    "localhost",
		ProxyAddress:  "localhost:3000",
	}
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

func Test_getTLSDialer(t *testing.T) {
	t.Run("Prefer client config certificate raw value fields over filepath fields", func(t *testing.T) {
		caPrivKey1, err := rsa.GenerateKey(rand.Reader, 4096)
		require.NoError(t, err)

		rootCACert1, rootCACertFilePath1, err := createRootCA(t, caPrivKey1)
		require.NoError(t, err)
		rootCaCertValue1, err := os.ReadFile(rootCACertFilePath1)
		require.NoError(t, err)

		clientCertValue1, clientCertFilePath1, err := createClientCert(t, rootCACert1, caPrivKey1)
		require.NoError(t, err)
		clientKeyValue1, clientKeyFilePath1, err := createClientKey(t, caPrivKey1)
		require.NoError(t, err)

		caPrivKey2, err := rsa.GenerateKey(rand.Reader, 4096)
		require.NoError(t, err)

		rootCACert2, rootCACertFilePath2, err := createRootCA(t, caPrivKey2)
		require.NoError(t, err)
		rootCaCertValue2, err := os.ReadFile(rootCACertFilePath2)
		require.NoError(t, err)

		clientCertValue2, _, err := createClientCert(t, rootCACert2, caPrivKey2)
		require.NoError(t, err)
		clientKeyValue2, _, err := createClientKey(t, caPrivKey2)
		require.NoError(t, err)

		require.NotEqual(t, rootCaCertValue1, rootCaCertValue2)
		require.NotEqual(t, clientCertValue1, clientCertValue2)
		require.NotEqual(t, clientKeyValue1, clientKeyValue2)

		clientCfg := &ClientCfg{
			ClientCert: clientCertFilePath1,
			ClientKey:  clientKeyFilePath1,
			RootCAs:    []string{rootCACertFilePath1},

			ClientCertVal: clientCertValue2,
			ClientKeyVal:  clientKeyValue2,
			RootCAsVals:   []string{string(rootCaCertValue2)},
		}

		p := cfgProxyWrapper{opts: &Options{ClientCfg: clientCfg, Timeouts: &TimeoutOptions{}}}

		dialer, err := p.getTLSDialer()
		require.NoError(t, err)
		require.NotNil(t, dialer)

		// check that the rootCaCert2 was used instead of rootCaCert1
		certPool := x509.NewCertPool()
		ok := certPool.AppendCertsFromPEM(rootCaCertValue2)
		require.True(t, ok)
		require.True(t, dialer.Config.RootCAs.Equal(certPool))

		certPool = x509.NewCertPool()
		ok = certPool.AppendCertsFromPEM(rootCaCertValue1)
		require.True(t, ok)
		require.False(t, dialer.Config.RootCAs.Equal(certPool))
	})
}

func createRootCA(t *testing.T, pvtKey *rsa.PrivateKey) (*x509.Certificate, string, error) {
	t.Helper()
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
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &pvtKey.PublicKey, pvtKey)
	if err != nil {
		return nil, "", err
	}
	caCertFile, err := os.CreateTemp(t.TempDir(), "rootCA-*.cert")
	if err != nil {
		return nil, "", err
	}
	err = pem.Encode(caCertFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})
	if err != nil {
		return nil, "", err
	}

	err = caCertFile.Close()
	if err != nil {
		return nil, "", err
	}
	return ca, caCertFile.Name(), nil
}

func createClientCert(t *testing.T, rootCaCert *x509.Certificate, pvtKey *rsa.PrivateKey) (string, string, error) {
	t.Helper()

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
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, rootCaCert, &pvtKey.PublicKey, pvtKey)
	if err != nil {
		return "", "", err
	}
	certFile, err := os.CreateTemp(t.TempDir(), "client-*.cert")
	if err != nil {
		return "", "", err
	}
	err = pem.Encode(certFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	if err != nil {
		return "", "", err
	}

	err = certFile.Close()
	if err != nil {
		return "", "", err
	}

	clientCertContents, err := os.ReadFile(certFile.Name())
	if err != nil {
		return "", "", err
	}

	return string(clientCertContents), certFile.Name(), nil
}

func createClientKey(t *testing.T, pvtKey *rsa.PrivateKey) (string, string, error) {
	t.Helper()

	keyFile, err := os.CreateTemp(t.TempDir(), "client-*.key")
	if err != nil {
		return "", "", err
	}
	err = pem.Encode(keyFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(pvtKey),
	})
	if err != nil {
		return "", "", err
	}

	err = keyFile.Close()
	if err != nil {
		return "", "", err
	}

	clientKeyContents, err := os.ReadFile(keyFile.Name())
	if err != nil {
		return "", "", err
	}
	return string(clientKeyContents), keyFile.Name(), nil
}
