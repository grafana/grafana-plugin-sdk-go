package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	sdkhttpclient "github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"golang.org/x/net/proxy"
)

var (
	PluginSecureSocksProxyClientCert   = "GF_SECURE_SOCKS_DATASOURCE_PROXY_CLIENT_CERT"
	PluginSecureSocksProxyClientKey    = "GF_SECURE_SOCKS_DATASOURCE_PROXY_CLIENT_KEY"
	PluginSecureSocksProxyRootCACert   = "GF_SECURE_SOCKS_DATASOURCE_PROXY_ROOT_CA_CERT"
	PluginSecureSocksProxyProxyAddress = "GF_SECURE_SOCKS_DATASOURCE_PROXY_PROXY_ADDRESS"
	PluginSecureSocksProxyServerName   = "GF_SECURE_SOCKS_DATASOURCE_PROXY_SERVER_NAME"
	PluginSecureSocksProxyEnabled      = "GF_SECURE_SOCKS_DATASOURCE_PROXY_SERVER_ENABLED"
)

type secureSocksConfig struct {
	clientCert   string
	clientKey    string
	rootCA       string
	proxyAddress string
	serverName   string
}

func SecureSocksProxyEnabled() bool {
	if value, ok := os.LookupEnv(PluginSecureSocksProxyEnabled); ok {
		res, err := strconv.ParseBool(value)
		if err != nil {
			return false
		}

		return res
	}

	return false
}

// NewSecureSocksHTTPProxy takes a http.DefaultTransport and wraps it in a socks5 proxy with TLS
func NewSecureSocksHTTPProxy(transport *http.Transport) error {
	dialSocksProxy, err := NewSecureSocksProxyContextDialer()
	if err != nil {
		return err
	}

	contextDialer, ok := dialSocksProxy.(proxy.ContextDialer)
	if !ok {
		return errors.New("unable to cast socks proxy dialer to context proxy dialer")
	}

	transport.DialContext = contextDialer.DialContext
	return nil
}

// NewSecureSocksProxyContextDialer returns a proxy context dialer that will wrap connections in a secure socks proxy
func NewSecureSocksProxyContextDialer() (proxy.Dialer, error) {
	cfg, err := getConfigFromEnv()
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	for _, rootCAFile := range strings.Split(cfg.rootCA, " ") {
		// nolint:gosec
		// The gosec G304 warning can be ignored because `rootCAFile` comes from config ini.
		pem, err := os.ReadFile(rootCAFile)
		if err != nil {
			return nil, err
		}
		if !certPool.AppendCertsFromPEM(pem) {
			return nil, errors.New("failed to append CA certificate " + rootCAFile)
		}
	}

	cert, err := tls.LoadX509KeyPair(cfg.clientCert, cfg.clientKey)
	if err != nil {
		return nil, err
	}

	tlsDialer := &tls.Dialer{
		Config: &tls.Config{
			Certificates: []tls.Certificate{cert},
			ServerName:   cfg.serverName,
			RootCAs:      certPool,
		},
	}
	dialSocksProxy, err := proxy.SOCKS5("tcp", cfg.proxyAddress, nil, tlsDialer)
	if err != nil {
		return nil, err
	}

	return dialSocksProxy, nil
}

// SecureSocksProxyEnabledOnDS checks the datasource json data to see if the secure socks proxy is enabled on it
func SecureSocksProxyEnabledOnDS(opts sdkhttpclient.Options) bool {
	jsonData := backend.JSONDataFromHTTPClientOptions(opts)
	res, enabled := jsonData["enableSecureSocksProxy"]
	if !enabled {
		return false
	}

	if val, ok := res.(bool); ok {
		return val
	}

	return false
}

func getConfigFromEnv() (*secureSocksConfig, error) {
	clientCert := ""
	if value, ok := os.LookupEnv(PluginSecureSocksProxyClientCert); ok {
		clientCert = value
	} else {
		return nil, fmt.Errorf("missing client cert")
	}

	clientKey := ""
	if value, ok := os.LookupEnv(PluginSecureSocksProxyClientKey); ok {
		clientKey = value
	} else {
		return nil, fmt.Errorf("missing client key")
	}

	rootCA := ""
	if value, ok := os.LookupEnv(PluginSecureSocksProxyRootCACert); ok {
		rootCA = value
	} else {
		return nil, fmt.Errorf("missing root ca")
	}

	proxyAddress := ""
	if value, ok := os.LookupEnv(PluginSecureSocksProxyProxyAddress); ok {
		proxyAddress = value
	} else {
		return nil, fmt.Errorf("missing proxy address")
	}

	serverName := ""
	if value, ok := os.LookupEnv(PluginSecureSocksProxyServerName); ok {
		serverName = value
	} else {
		return nil, fmt.Errorf("missing server name")
	}

	return &secureSocksConfig{
		clientCert:   clientCert,
		clientKey:    clientKey,
		rootCA:       rootCA,
		proxyAddress: proxyAddress,
		serverName:   serverName,
	}, nil
}
