package config

import (
	"context"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/backend/proxy"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/featuretoggles"
)

const (
	AppURL                           = "GF_APP_URL"
	ConcurrentQueryCount             = "GF_CONCURRENT_QUERY_COUNT"
	UserFacingDefaultError           = "GF_USER_FACING_DEFAULT_ERROR"
	SQLRowLimit                      = "GF_SQL_ROW_LIMIT"
	SQLMaxOpenConnsDefault           = "GF_SQL_MAX_OPEN_CONNS_DEFAULT"
	SQLMaxIdleConnsDefault           = "GF_SQL_MAX_IDLE_CONNS_DEFAULT"
	SQLMaxConnLifetimeSecondsDefault = "GF_SQL_MAX_CONN_LIFETIME_SECONDS_DEFAULT"
	ResponseLimit                    = "GF_RESPONSE_LIMIT"
	AppClientSecret                  = "GF_PLUGIN_APP_CLIENT_SECRET" // nolint:gosec
)

// GrafanaCfg represents Grafana configuration
type GrafanaCfg struct {
	config map[string]string
}

type configKey struct{}

// GrafanaConfigFromContext returns Grafana config from context.
func GrafanaConfigFromContext(ctx context.Context) *GrafanaCfg {
	v := ctx.Value(configKey{})
	if v == nil {
		return NewGrafanaCfg(nil)
	}

	cfg := v.(*GrafanaCfg)
	if cfg == nil {
		return NewGrafanaCfg(nil)
	}

	return cfg
}

// WithGrafanaConfig injects supplied Grafana config into context.
func WithGrafanaConfig(ctx context.Context, cfg *GrafanaCfg) context.Context {
	ctx = context.WithValue(ctx, configKey{}, cfg)
	return ctx
}

// NewGrafanaCfg creates a new GrafanaCfg instance
func NewGrafanaCfg(cfg map[string]string) *GrafanaCfg {
	return &GrafanaCfg{config: cfg}
}

// Config returns the config map
func (c *GrafanaCfg) Config() map[string]string {
	return c.config
}

// Get returns a value from the config map
func (c *GrafanaCfg) Get(key string) string {
	return c.config[key]
}

func (c *GrafanaCfg) FeatureToggles() FeatureToggles {
	features, exists := c.config[featuretoggles.EnabledFeatures]
	if !exists || features == "" {
		return FeatureToggles{}
	}

	fs := strings.Split(features, ",")
	enabledFeatures := make(map[string]struct{}, len(fs))
	for _, f := range fs {
		enabledFeatures[f] = struct{}{}
	}

	return FeatureToggles{
		enabled: enabledFeatures,
	}
}

func (c *GrafanaCfg) Equal(c2 *GrafanaCfg) bool {
	if c == nil && c2 == nil {
		return true
	}
	if c == nil || c2 == nil {
		return false
	}

	if len(c.config) != len(c2.config) {
		return false
	}
	for k, v1 := range c.config {
		if v2, ok := c2.config[k]; !ok || v1 != v2 {
			return false
		}
	}
	return true
}

// ProxyHash returns the last four characters of the base64-encoded
// PDC client key contents, if present, for use in datasource instance
// caching. The contents should be PEM-encoded, so we try to PEM-decode
// them, and, if successful, return the base-64 encoding of the final three bytes,
// giving a four character hash.
func (c *GrafanaCfg) ProxyHash() string {
	if c == nil {
		return ""
	}
	contents := c.config[proxy.PluginSecureSocksProxyClientKeyContents]
	if contents == "" {
		return ""
	}
	block, _ := pem.Decode([]byte(contents))
	if block == nil {
		log.DefaultLogger.Warn("ProxyHash(): key contents are not PEM-encoded")
		return ""
	}
	if block.Type != "PRIVATE KEY" {
		log.DefaultLogger.Warn("ProxyHash(): key contents are not PEM-encoded private key")
		return ""
	}
	bl := len(block.Bytes)
	if bl < 3 {
		log.DefaultLogger.Warn("ProxyHash(): key contents too short")
		return ""
	}
	return base64.StdEncoding.EncodeToString(block.Bytes[bl-3:])
}

type FeatureToggles struct {
	// enabled is a set-like map of feature flags that are enabled.
	enabled map[string]struct{}
}

// IsEnabled returns true if feature f is contained in ft.enabled.
func (ft FeatureToggles) IsEnabled(f string) bool {
	_, exists := ft.enabled[f]
	return exists
}

type Proxy struct {
	ClientCfg *proxy.ClientCfg
}

func (c *GrafanaCfg) Proxy() (Proxy, error) {
	if v, exists := c.config[proxy.PluginSecureSocksProxyEnabled]; exists && v == strconv.FormatBool(true) {
		var (
			allowInsecure = false
			err           error
		)

		if v := c.Get(proxy.PluginSecureSocksProxyAllowInsecure); v != "" {
			allowInsecure, err = strconv.ParseBool(c.Get(proxy.PluginSecureSocksProxyAllowInsecure))
			if err != nil {
				return Proxy{}, fmt.Errorf("parsing %s, value must be a boolean: %w", proxy.PluginSecureSocksProxyAllowInsecure, err)
			}
		}

		var rootCaVals []string
		if v = c.Get(proxy.PluginSecureSocksProxyRootCAsContents); v != "" {
			rootCaVals = strings.Split(c.Get(proxy.PluginSecureSocksProxyRootCAsContents), ",")
		}

		return Proxy{
			ClientCfg: &proxy.ClientCfg{
				ClientCert:    c.Get(proxy.PluginSecureSocksProxyClientCert),
				ClientCertVal: c.Get(proxy.PluginSecureSocksProxyClientCertContents),
				ClientKey:     c.Get(proxy.PluginSecureSocksProxyClientKey),
				ClientKeyVal:  c.Get(proxy.PluginSecureSocksProxyClientKeyContents),
				RootCAs:       strings.Split(c.Get(proxy.PluginSecureSocksProxyRootCAs), " "),
				RootCAsVals:   rootCaVals,
				ProxyAddress:  c.Get(proxy.PluginSecureSocksProxyProxyAddress),
				ServerName:    c.Get(proxy.PluginSecureSocksProxyServerName),
				AllowInsecure: allowInsecure,
			},
		}, nil
	}

	return Proxy{}, nil
}

func (c *GrafanaCfg) AppURL() (string, error) {
	url, ok := c.config[AppURL]
	if !ok {
		// Fallback to environment variable for backwards compatibility
		url = os.Getenv(AppURL)
		if url == "" {
			return "", errors.New("app URL not set in config. A more recent version of Grafana may be required")
		}
	}
	return url, nil
}

func (c *GrafanaCfg) ConcurrentQueryCount() (int, error) {
	count, ok := c.config[ConcurrentQueryCount]
	if !ok {
		return 0, fmt.Errorf("ConcurrentQueryCount not set in config")
	}
	i, err := strconv.Atoi(count)
	if err != nil {
		return 0, fmt.Errorf("ConcurrentQueryCount cannot be converted to integer")
	}
	return i, nil
}

type SQLConfig struct {
	RowLimit                      int64
	DefaultMaxOpenConns           int
	DefaultMaxIdleConns           int
	DefaultMaxConnLifetimeSeconds int
}

func (c *GrafanaCfg) SQL() (SQLConfig, error) {
	// max open connections
	maxOpenString, ok := c.config[SQLMaxOpenConnsDefault]
	if !ok {
		return SQLConfig{}, errors.New("SQLDatasourceMaxOpenConnsDefault not set in config")
	}

	maxOpen, err := strconv.Atoi(maxOpenString)
	if err != nil {
		return SQLConfig{}, errors.New("SQLDatasourceMaxOpenConnsDefault config value is not a valid integer")
	}

	// max idle connections
	maxIdleString, ok := c.config[SQLMaxIdleConnsDefault]
	if !ok {
		return SQLConfig{}, errors.New("SQLDatasourceMaxIdleConnsDefault not set in config")
	}

	maxIdle, err := strconv.Atoi(maxIdleString)
	if err != nil {
		return SQLConfig{}, errors.New("SQLDatasourceMaxIdleConnsDefault config value is not a valid integer")
	}

	// max connection lifetime
	maxLifeString, ok := c.config[SQLMaxConnLifetimeSecondsDefault]
	if !ok {
		return SQLConfig{}, errors.New("SQLDatasourceMaxConnLifetimeDefault not set in config")
	}

	maxLife, err := strconv.Atoi(maxLifeString)
	if err != nil {
		return SQLConfig{}, errors.New("SQLDatasourceMaxConnLifetimeDefault config value is not a valid integer")
	}

	rowLimitString, ok := c.config[SQLRowLimit]
	if !ok {
		return SQLConfig{}, errors.New("RowLimit not set in config")
	}

	rowLimit, err := strconv.ParseInt(rowLimitString, 10, 64)
	if err != nil {
		return SQLConfig{}, errors.New("RowLimit in config is not a valid integer")
	}

	return SQLConfig{
		RowLimit:                      rowLimit,
		DefaultMaxOpenConns:           maxOpen,
		DefaultMaxIdleConns:           maxIdle,
		DefaultMaxConnLifetimeSeconds: maxLife,
	}, nil
}

func (c *GrafanaCfg) UserFacingDefaultError() (string, error) {
	value, ok := c.config[UserFacingDefaultError]
	if !ok {
		return "", errors.New("UserFacingDefaultError not set in config")
	}

	return value, nil
}

func (c *GrafanaCfg) ResponseLimit() int64 {
	count, ok := c.config[ResponseLimit]
	if !ok {
		return 0
	}
	i, err := strconv.ParseInt(count, 10, 64)
	if err != nil {
		return 0
	}
	return i
}

func (c *GrafanaCfg) PluginAppClientSecret() (string, error) {
	value, ok := c.config[AppClientSecret]
	if !ok {
		// Fallback to environment variable for backwards compatibility
		value = os.Getenv(AppClientSecret)
		if value == "" {
			return "", errors.New("PluginAppClientSecret not set in config")
		}
	}

	return value, nil
}
