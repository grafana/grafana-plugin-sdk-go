package config

import (
	"context"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"hash/maphash"
	"os"
	"strconv"
	"strings"
	"unique"

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
	LiveClientQueueMaxSize           = "GF_LIVE_CLIENT_QUEUE_MAX_SIZE"
)

// GrafanaCfg represents Grafana configuration
type GrafanaCfg struct {
	config map[string]string
	// hashes is non-nil only in a Digest.
	hashes map[unique.Handle[string]]uint64
}

var valueSeed = maphash.MakeSeed()

// Digest returns a GrafanaCfg that is Equal to c but keeps a hash of each value in place of the value.
// Its getters return zero values. Diff still names the keys that differ.
func (c *GrafanaCfg) Digest() *GrafanaCfg {
	if c == nil || c.hashes != nil {
		return c
	}
	hashes := make(map[unique.Handle[string]]uint64, len(c.config))
	for k, v := range c.config {
		hashes[unique.Make(k)] = maphash.String(valueSeed, v)
	}
	return &GrafanaCfg{hashes: hashes}
}

func (c *GrafanaCfg) valueHashes() map[string]uint64 {
	if c == nil {
		return nil
	}
	hashes := make(map[string]uint64, max(len(c.config), len(c.hashes)))
	for k, v := range c.config {
		hashes[k] = maphash.String(valueSeed, v)
	}
	for k, h := range c.hashes {
		hashes[k.Value()] = h
	}
	return hashes
}

var emptyValueHash = maphash.String(valueSeed, "")

func (c *GrafanaCfg) hasValue(key string) bool {
	if c.hashes == nil {
		return c.config[key] != ""
	}
	h, ok := c.hashes[unique.Make(key)]
	return ok && h != emptyValueHash
}

func (c *GrafanaCfg) valueHash(key string, handle unique.Handle[string]) (uint64, bool) {
	if c.hashes != nil {
		h, ok := c.hashes[handle]
		return h, ok
	}
	v, ok := c.config[key]
	return maphash.String(valueSeed, v), ok
}

func (c *GrafanaCfg) keptKeys(ignored map[string]struct{}) int {
	if len(ignored) == 0 {
		return len(c.config) + len(c.hashes)
	}
	n := 0
	for k := range c.config {
		if _, skip := ignored[k]; !skip {
			n++
		}
	}
	for k := range c.hashes {
		if _, skip := ignored[k.Value()]; !skip {
			n++
		}
	}
	return n
}

// digestEqual compares the counts of kept keys, so finding each kept key of d in c proves that the key sets are the same.
func digestEqual(d, c *GrafanaCfg, ignored map[string]struct{}) bool {
	kept := 0
	for k, h := range d.hashes {
		key := k.Value()
		if _, skip := ignored[key]; skip {
			continue
		}
		kept++
		if h2, ok := c.valueHash(key, k); !ok || h2 != h {
			return false
		}
	}
	return kept == c.keptKeys(ignored)
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

// Equal reports whether c and c2 hold the same configuration, ignoring keys that neither config can use.
func (c *GrafanaCfg) Equal(c2 *GrafanaCfg) bool {
	if c == nil && c2 == nil {
		return true
	}
	if c == nil || c2 == nil {
		return false
	}

	ignored := unusedKeys(c, c2)
	if c.hashes != nil {
		return digestEqual(c, c2, ignored)
	}
	if c2.hashes != nil {
		return digestEqual(c2, c, ignored)
	}
	for k, v1 := range c.config {
		if _, skip := ignored[k]; skip {
			continue
		}
		if v2, ok := c2.config[k]; !ok || v1 != v2 {
			return false
		}
	}
	for k := range c2.config {
		if _, skip := ignored[k]; skip {
			continue
		}
		if _, ok := c.config[k]; !ok {
			return false
		}
	}
	return true
}

// pdcFilePathKeys are the deprecated PDC certificate paths, which the proxy dialer reads only when no root CA contents are set.
var pdcFilePathKeys = map[string]struct{}{
	proxy.PluginSecureSocksProxyClientCert: {}, // nolint:staticcheck
	proxy.PluginSecureSocksProxyClientKey:  {}, // nolint:staticcheck
	proxy.PluginSecureSocksProxyRootCAs:    {}, // nolint:staticcheck
}

func unusedKeys(c, c2 *GrafanaCfg) map[string]struct{} {
	if c.hasValue(proxy.PluginSecureSocksProxyRootCAsContents) && c2.hasValue(proxy.PluginSecureSocksProxyRootCAsContents) {
		return pdcFilePathKeys
	}
	return nil
}

// Diff returns the names of config fields that differ between this config and c2.
// Returns field names for added, removed, or changed values.
// Always returns a slice, never nil.
func (c *GrafanaCfg) Diff(c2 *GrafanaCfg) []string {
	if c == nil && c2 == nil {
		return []string{}
	}
	if (c != nil && c.hashes != nil) || (c2 != nil && c2.hashes != nil) {
		return diffHashes(c.valueHashes(), c2.valueHashes())
	}
	if c == nil {
		var keys []string
		for k := range c2.config {
			keys = append(keys, k)
		}
		return keys
	}
	if c2 == nil {
		var keys []string
		for k := range c.config {
			keys = append(keys, k)
		}
		return keys
	}

	var changed []string
	keys := make(map[string]bool)
	for k := range c.config {
		keys[k] = true
	}
	for k := range c2.config {
		keys[k] = true
	}

	for key := range keys {
		v1, ok1 := c.config[key]
		v2, ok2 := c2.config[key]
		if ok1 != ok2 || v1 != v2 {
			changed = append(changed, key)
		}
	}

	return changed
}

func diffHashes(h1, h2 map[string]uint64) []string {
	changed := []string{}
	for k, v1 := range h1 {
		if v2, ok := h2[k]; !ok || v1 != v2 {
			changed = append(changed, k)
		}
	}
	for k := range h2 {
		if _, ok := h1[k]; !ok {
			changed = append(changed, k)
		}
	}
	return changed
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

// FeatureToggles represents Grafana feature toggles.
type FeatureToggles struct {
	// enabled is a set-like map of feature flags that are enabled.
	enabled map[string]struct{}
}

// IsEnabled returns true if feature f is contained in ft.enabled.
func (ft FeatureToggles) IsEnabled(f string) bool {
	_, exists := ft.enabled[f]
	return exists
}

// Proxy represents Grafana PDC proxy configuration.
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
				ClientCert:    c.Get(proxy.PluginSecureSocksProxyClientCert), // nolint:staticcheck
				ClientCertVal: c.Get(proxy.PluginSecureSocksProxyClientCertContents),
				ClientKey:     c.Get(proxy.PluginSecureSocksProxyClientKey), // nolint:staticcheck
				ClientKeyVal:  c.Get(proxy.PluginSecureSocksProxyClientKeyContents),
				RootCAs:       strings.Split(c.Get(proxy.PluginSecureSocksProxyRootCAs), " "), // nolint:staticcheck
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
