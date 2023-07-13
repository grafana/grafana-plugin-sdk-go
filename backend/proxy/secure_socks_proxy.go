package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"golang.org/x/net/proxy"
)

var (

	// PluginSecureSocksProxyEnabled is a constant for the GF_SECURE_SOCKS_DATASOURCE_PROXY_SERVER_ENABLED
	// environment variable used to specify if a secure socks proxy is allowed to be used for datasource connections.
	PluginSecureSocksProxyEnabled = "GF_SECURE_SOCKS_DATASOURCE_PROXY_SERVER_ENABLED"
	// PluginSecureSocksProxyClientCert is a constant for the GF_SECURE_SOCKS_DATASOURCE_PROXY_CLIENT_CERT
	// environment variable used to specify the file location of the client cert for the secure socks proxy.
	PluginSecureSocksProxyClientCert = "GF_SECURE_SOCKS_DATASOURCE_PROXY_CLIENT_CERT"
	// PluginSecureSocksProxyClientKey is a constant for the GF_SECURE_SOCKS_DATASOURCE_PROXY_CLIENT_KEY
	// environment variable used to specify the file location of the client key for the secure socks proxy.
	PluginSecureSocksProxyClientKey = "GF_SECURE_SOCKS_DATASOURCE_PROXY_CLIENT_KEY"
	// PluginSecureSocksProxyRootCACert is a constant for the GF_SECURE_SOCKS_DATASOURCE_PROXY_ROOT_CA_CERT
	// environment variable used to specify the file location of the root ca for the secure socks proxy.
	PluginSecureSocksProxyRootCACert = "GF_SECURE_SOCKS_DATASOURCE_PROXY_ROOT_CA_CERT"
	// PluginSecureSocksProxyProxyAddress is a constant for the GF_SECURE_SOCKS_DATASOURCE_PROXY_PROXY_ADDRESS
	// environment variable used to specify the secure socks proxy server address to proxy the connections to.
	PluginSecureSocksProxyProxyAddress = "GF_SECURE_SOCKS_DATASOURCE_PROXY_PROXY_ADDRESS"
	// PluginSecureSocksProxyServerName is a constant for the GF_SECURE_SOCKS_DATASOURCE_PROXY_SERVER_NAME
	// environment variable used to specify the server name of the secure socks proxy.
	PluginSecureSocksProxyServerName = "GF_SECURE_SOCKS_DATASOURCE_PROXY_SERVER_NAME"
)

// SecureSocksProxyConfig contains the information needed to allow datasource connections to be
// proxied to a secure socks proxy. The proxy will automatically use the env variables for this information
type SecureSocksProxyConfig struct {
	Enabled      bool
	ClientCert   string
	ClientKey    string
	RootCA       string
	ProxyAddress string
	ServerName   string
}

// SecureSocksProxyEnabled checks if the Grafana instance allows the secure socks proxy to be used
// and the datasource options specify to use the proxy
func SecureSocksProxyEnabled(opts *Options) bool {
	if opts == nil {
		return false
	}

	if !opts.Enabled {
		return false
	}

	return (proxyConfig != nil) && proxyConfig.Enabled
}

// SecureSocksProxyEnabledOnDS checks the datasource json data for `enableSecureSocksProxy`
// to determine if the secure socks proxy should be enabled on it
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

// ConfigureSecureSocksHTTPProxy takes a http.DefaultTransport and wraps it in a socks5 proxy with TLS
// if it is enabled on the datasource and the grafana instance
func ConfigureSecureSocksHTTPProxy(transport *http.Transport, opts *Options) error {
	if !SecureSocksProxyEnabled(opts) {
		return nil
	}

	dialSocksProxy, err := NewSecureSocksProxyContextDialer(opts)
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

// NewSecureSocksProxyContextDialer returns a proxy context dialer that can be used to allow datasource connections to go through a secure socks proxy
func NewSecureSocksProxyContextDialer(opts *Options) (proxy.Dialer, error) {
	if !SecureSocksProxyEnabled(opts) {
		return nil, fmt.Errorf("proxy not enabled")
	}

	clientOpts := createOptions(opts)

	certPool := x509.NewCertPool()
	for _, rootCAFile := range strings.Split(proxyConfig.RootCA, " ") {
		// nolint:gosec
		// The gosec G304 warning can be ignored because `rootCAFile` comes from config ini
		// and we check below if it's the right file type
		pemBytes, err := os.ReadFile(rootCAFile)
		if err != nil {
			return nil, err
		}

		pemDecoded, _ := pem.Decode(pemBytes)
		if pemDecoded == nil || pemDecoded.Type != "CERTIFICATE" {
			return nil, errors.New("root ca is invalid")
		}

		if !certPool.AppendCertsFromPEM(pemBytes) {
			return nil, errors.New("failed to append CA certificate " + rootCAFile)
		}
	}

	cert, err := tls.LoadX509KeyPair(proxyConfig.ClientCert, proxyConfig.ClientKey)
	if err != nil {
		return nil, err
	}

	tlsDialer := &tls.Dialer{
		Config: &tls.Config{
			Certificates: []tls.Certificate{cert},
			ServerName:   proxyConfig.ServerName,
			RootCAs:      certPool,
			MinVersion:   tls.VersionTLS13,
		},
		NetDialer: &net.Dialer{
			Timeout:   clientOpts.Timeouts.Timeout,
			KeepAlive: clientOpts.Timeouts.KeepAlive,
		},
	}

	var auth *proxy.Auth
	if clientOpts.Auth != nil {
		auth = &proxy.Auth{
			User:     clientOpts.Auth.Username,
			Password: clientOpts.Auth.Password,
		}
	}

	dialSocksProxy, err := proxy.SOCKS5("tcp", proxyConfig.ProxyAddress, auth, tlsDialer)
	if err != nil {
		return nil, err
	}

	return dialSocksProxy, nil
}

// SetSecureSocksProxyConfig overrides the default proxy configuration set by the Grafana env variables
func SetSecureSocksProxyConfig(cfg *SecureSocksProxyConfig) {
	proxyConfig = cfg
}

// proxyConfig is automatically set through the env variables added to plugin contexts by grafana
var proxyConfig = getConfigFromEnv()

// getConfigFromEnv gets the needed proxy information from the env variables that Grafana set with the values from the config ini
func getConfigFromEnv() *SecureSocksProxyConfig {
	if value, ok := os.LookupEnv(PluginSecureSocksProxyEnabled); ok {
		enabled, err := strconv.ParseBool(value)
		if err != nil || !enabled {
			return nil
		}
	}

	clientCert := ""
	if value, ok := os.LookupEnv(PluginSecureSocksProxyClientCert); ok {
		clientCert = value
	} else {
		return nil
	}

	clientKey := ""
	if value, ok := os.LookupEnv(PluginSecureSocksProxyClientKey); ok {
		clientKey = value
	} else {
		return nil
	}

	rootCA := ""
	if value, ok := os.LookupEnv(PluginSecureSocksProxyRootCACert); ok {
		rootCA = value
	} else {
		return nil
	}

	proxyAddress := ""
	if value, ok := os.LookupEnv(PluginSecureSocksProxyProxyAddress); ok {
		proxyAddress = value
	} else {
		return nil
	}

	serverName := ""
	if value, ok := os.LookupEnv(PluginSecureSocksProxyServerName); ok {
		serverName = value
	} else {
		return nil
	}

	return &SecureSocksProxyConfig{
		Enabled:      true,
		ClientCert:   clientCert,
		ClientKey:    clientKey,
		RootCA:       rootCA,
		ProxyAddress: proxyAddress,
		ServerName:   serverName,
	}
}
