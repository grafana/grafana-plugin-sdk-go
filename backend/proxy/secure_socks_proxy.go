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

	"golang.org/x/net/proxy"
)

var (
	PluginSecureSocksProxyEnabled      = "GF_SECURE_SOCKS_DATASOURCE_PROXY_SERVER_ENABLED"
	PluginSecureSocksProxyClientCert   = "GF_SECURE_SOCKS_DATASOURCE_PROXY_CLIENT_CERT"
	PluginSecureSocksProxyClientKey    = "GF_SECURE_SOCKS_DATASOURCE_PROXY_CLIENT_KEY"
	PluginSecureSocksProxyRootCACert   = "GF_SECURE_SOCKS_DATASOURCE_PROXY_ROOT_CA_CERT"
	PluginSecureSocksProxyProxyAddress = "GF_SECURE_SOCKS_DATASOURCE_PROXY_PROXY_ADDRESS"
	PluginSecureSocksProxyServerName   = "GF_SECURE_SOCKS_DATASOURCE_PROXY_SERVER_NAME"
)

type SecureSocksProxyConfig struct {
	Enabled      bool
	ClientCert   string
	ClientKey    string
	RootCA       string
	ProxyAddress string
	ServerName   string
}

// SecureSocksProxyEnabled checks if the Grafana instance allows the secure socks proxy to be used
func SecureSocksProxyEnabled(cfg *SecureSocksProxyConfig) bool {
	// if passed in, use the config, otherwise attempt to find it as an env variable
	if cfg != nil {
		return cfg.Enabled
	}

	if value, ok := os.LookupEnv(PluginSecureSocksProxyEnabled); ok {
		res, err := strconv.ParseBool(value)
		if err != nil {
			return false
		}

		return res
	}

	return false
}

// SecureSocksProxyEnabledOnDS checks the datasource json data to see if the secure socks proxy is enabled on it
func SecureSocksProxyEnabledOnDS(jsonData map[string]interface{}) bool {
	res, enabled := jsonData["enableSecureSocksProxy"]
	if !enabled {
		return false
	}

	if val, ok := res.(bool); ok {
		return val
	}

	return false
}

// NewSecureSocksHTTPProxy takes a http.DefaultTransport and wraps it in a socks5 proxy with TLS
func NewSecureSocksHTTPProxy(cfg *SecureSocksProxyConfig, transport *http.Transport, dsUID string) error {
	dialSocksProxy, err := NewSecureSocksProxyContextDialer(cfg, dsUID)
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
func NewSecureSocksProxyContextDialer(cfg *SecureSocksProxyConfig, dsUID string) (proxy.Dialer, error) {
	var err error
	// use the config, if passed in, otherwise attempt to get the values from the env
	if cfg == nil {
		cfg, err = getConfigFromEnv()
		if err != nil {
			return nil, err
		}
	}

	certPool := x509.NewCertPool()
	for _, rootCAFile := range strings.Split(cfg.RootCA, " ") {
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

	cert, err := tls.LoadX509KeyPair(cfg.ClientCert, cfg.ClientKey)
	if err != nil {
		return nil, err
	}

	tlsDialer := &tls.Dialer{
		Config: &tls.Config{
			Certificates: []tls.Certificate{cert},
			ServerName:   cfg.ServerName,
			RootCAs:      certPool,
		},
	}

	dsInfo := proxy.Auth{
		User: dsUID,
	}

	dialSocksProxy, err := proxy.SOCKS5("tcp", cfg.ProxyAddress, &dsInfo, tlsDialer)
	if err != nil {
		return nil, err
	}

	return dialSocksProxy, nil
}

// getConfigFromEnv gets the proxy information via env variables that were set by Grafana with values from the config ini
func getConfigFromEnv() (*SecureSocksProxyConfig, error) {
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

	return &SecureSocksProxyConfig{
		ClientCert:   clientCert,
		ClientKey:    clientKey,
		RootCA:       rootCA,
		ProxyAddress: proxyAddress,
		ServerName:   serverName,
	}, nil
}
