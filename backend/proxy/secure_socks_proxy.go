package proxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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

var (
	socksUnknownError              = regexp.MustCompile(`unknown code: (\d+)`)
	secureSocksConnectionsDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "grafana",
		Name:      "secure_socks_connections_duration",
		Help:      "Duration of establishing a connection to a secure socks proxy",
	}, []string{"code"})
)

// Client is the main Proxy Client interface.
type Client interface {
	SecureSocksProxyEnabled(opts *Options) bool
	ConfigureSecureSocksHTTPProxy(transport *http.Transport, opts *Options) error
	NewSecureSocksProxyContextDialer(opts *Options) (proxy.Dialer, error)
}

// ClientCfg contains the information needed to allow datasource connections to be
// proxied to a secure socks proxy.
type ClientCfg struct {
	Enabled      bool
	ClientCert   string
	ClientKey    string
	RootCA       string
	ProxyAddress string
	ServerName   string
}

// Cli is the default Proxy Client.
var Cli = New()

// New creates a new proxy client from the environment variables set by the grafana-server in the plugin.
func New() Client {
	return NewWithCfg(getConfigFromEnv())
}

// NewWithCfg creates a new proxy client from a given config.
func NewWithCfg(cfg *ClientCfg) Client {
	return &cfgProxyWrapper{
		cfg: cfg,
	}
}

type cfgProxyWrapper struct {
	cfg *ClientCfg
}

// SecureSocksProxyEnabled checks if the Grafana instance allows the secure socks proxy to be used
// and the datasource options specify to use the proxy
func (p *cfgProxyWrapper) SecureSocksProxyEnabled(opts *Options) bool {
	// it cannot be enabled if it's not enabled on Grafana
	if p.cfg == nil || !p.cfg.Enabled {
		return false
	}

	// if it's enabled on Grafana, check if the datasource is using it
	return (opts != nil) && opts.Enabled
}

// ConfigureSecureSocksHTTPProxy takes a http.DefaultTransport and wraps it in a socks5 proxy with TLS
// if it is enabled on the datasource and the grafana instance
func (p *cfgProxyWrapper) ConfigureSecureSocksHTTPProxy(transport *http.Transport, opts *Options) error {
	if !p.SecureSocksProxyEnabled(opts) {
		return nil
	}

	dialSocksProxy, err := p.NewSecureSocksProxyContextDialer(opts)
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
func (p *cfgProxyWrapper) NewSecureSocksProxyContextDialer(opts *Options) (proxy.Dialer, error) {
	if !p.SecureSocksProxyEnabled(opts) {
		return nil, fmt.Errorf("proxy not enabled")
	}

	clientOpts := setDefaults(opts)

	certPool := x509.NewCertPool()
	for _, rootCAFile := range strings.Split(p.cfg.RootCA, " ") {
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

	cert, err := tls.LoadX509KeyPair(p.cfg.ClientCert, p.cfg.ClientKey)
	if err != nil {
		return nil, err
	}

	tlsDialer := &tls.Dialer{
		Config: &tls.Config{
			Certificates: []tls.Certificate{cert},
			ServerName:   p.cfg.ServerName,
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

	dialSocksProxy, err := proxy.SOCKS5("tcp", p.cfg.ProxyAddress, auth, tlsDialer)
	if err != nil {
		return nil, err
	}

	instrumentedSocksDialer := NewInstrumentedSocksDialer(dialSocksProxy)

	return instrumentedSocksDialer, nil
}

// getConfigFromEnv gets the needed proxy information from the env variables that Grafana set with the values from the config ini
func getConfigFromEnv() *ClientCfg {
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

	return &ClientCfg{
		Enabled:      true,
		ClientCert:   clientCert,
		ClientKey:    clientKey,
		RootCA:       rootCA,
		ProxyAddress: proxyAddress,
		ServerName:   serverName,
	}
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

// instrumentedSocksDialer  is a wrapper around the proxy.Dialer and proxy.DialContext
// that records relevant socks secure socks proxy.
type instrumentedSocksDialer struct {
	dialer proxy.Dialer
}

// NewInstrumenSockstedDialer creates a new instrumented dialer
func NewInstrumentedSocksDialer(dialer proxy.Dialer) proxy.Dialer {
	return &instrumentedSocksDialer{
		dialer: dialer,
	}
}

// Dial -
func (d *instrumentedSocksDialer) Dial(network, addr string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, addr)
}

// DialContext -
func (d *instrumentedSocksDialer) DialContext(ctx context.Context, n, addr string) (net.Conn, error) {
	start := time.Now()
	dialer, ok := d.dialer.(proxy.ContextDialer)
	if !ok {
		return nil, errors.New("unable to cast socks proxy dialer to context proxy dialer")
	}
	c, err := dialer.DialContext(ctx, n, addr)

	var code string
	if err == nil {
		code = "0"
	} else if socksErr, ok := err.(*net.OpError); ok && strings.Contains(socksErr.Op, "socks") {
		// Socks errors defined here: https://cs.opensource.google/go/x/net/+/refs/tags/v0.15.0:internal/socks/socks.go;l=40-63
		if strings.Contains(socksErr.Error(), "general SOCKS server failure") {
			code = "1"
		} else if strings.Contains(socksErr.Error(), "connection not allowed by ruleset") {
			code = "2"
		} else if strings.Contains(socksErr.Error(), "network unreachable") {
			code = "3"
		} else if strings.Contains(socksErr.Error(), "host unreachable") {
			code = "4"
		} else if strings.Contains(socksErr.Error(), "connection refused") {
			code = "5"
		} else if strings.Contains(socksErr.Error(), "TTL expired") {
			code = "6"
		} else if strings.Contains(socksErr.Error(), "command not supported") {
			code = "7"
		} else if strings.Contains(socksErr.Error(), "address type not supported") {
			code = "8"
		} else if match := socksUnknownError.FindStringSubmatch(socksErr.Error()); len(match) > 1 {
			code = match[1]
		} else {
			code = "socks_unknown_error"
		}
	} else {
		code = "dial_error"
	}

	secureSocksConnectionsDuration.WithLabelValues(code).Observe(time.Since(start).Seconds())
	return c, err
}
