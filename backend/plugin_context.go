package backend

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend/useragent"
)

// PluginContext holds contextual information about a plugin request, such as
// Grafana organization, user and plugin instance settings.
type PluginContext struct {
	// OrgID is The Grafana organization identifier the request originating from.
	OrgID int64

	// PluginID is the unique identifier of the plugin that the request is for.
	PluginID string

	// PluginVersion is the version of the plugin that the request is for.
	PluginVersion string

	// User is the Grafana user making the request.
	//
	// Will not be provided if Grafana backend initiated the request,
	// for example when request is coming from Grafana Alerting.
	User *User

	// AppInstanceSettings is the configured app instance settings.
	//
	// In Grafana an app instance is an app plugin of certain
	// type that have been configured and enabled in a Grafana organization.
	//
	// Will only be set if request targeting an app instance.
	AppInstanceSettings *AppInstanceSettings

	// DataSourceConfig is the configured data source instance
	// settings.
	//
	// In Grafana a data source instance is a data source plugin of certain
	// type that have been configured and created in a Grafana organization.
	//
	// Will only be set if request targeting a data source instance.
	DataSourceInstanceSettings *DataSourceInstanceSettings

	// GrafanaConfig is the configuration settings provided by Grafana.
	GrafanaConfig *GrafanaCfg

	// UserAgent is the user agent of the Grafana server that initiated the gRPC request.
	// Will only be set if request is made from Grafana v10.2.0 or later.
	UserAgent *useragent.UserAgent

	// The requested API version
	APIVersion string
}

// GetSettingFromEnv retrieves the environment variable value based on the provided key.
// It first checks for a general plugin setting, then a plugin-specific setting, and finally
// a data source-specific setting.
// The search is case-insensitive for the key but case-sensitive for the data source UID.
// NOTE: This method can't be used in multi-tenant environment such as grafana cloud
func (pCtx *PluginContext) GetSettingFromEnv(key string) (output string) {
	key = strings.TrimSpace(strings.ToUpper(key))

	if v := strings.TrimSpace(os.Getenv(fmt.Sprintf("GF_PLUGIN_%s", key))); v != "" {
		output = v
	}

	if pCtx == nil {
		return output
	}

	pluginID := strings.TrimSpace(strings.ToUpper(pCtx.PluginID))
	if v := strings.TrimSpace(os.Getenv(fmt.Sprintf("GF_PLUGIN_%s_%s", pluginID, key))); v != "" && pluginID != "" {
		output = v
	}

	if pCtx.DataSourceInstanceSettings == nil {
		return output
	}

	dsUID := strings.TrimSpace(strings.ToUpper(pCtx.DataSourceInstanceSettings.UID))
	if v := strings.TrimSpace(os.Getenv(fmt.Sprintf("GF_DS_%s_%s", dsUID, key))); v != "" && dsUID != "" {
		output = v
	}

	caseSensitiveDsUID := strings.TrimSpace(pCtx.DataSourceInstanceSettings.UID)
	if v := strings.TrimSpace(os.Getenv(fmt.Sprintf("GF_DS_%s_%s", caseSensitiveDsUID, key))); v != "" && caseSensitiveDsUID != "" {
		output = v
	}

	return output
}

// GetSettingAsBoolFromEnv retrieves the environment variable value as boolean based on the provided key.
// NOTE: This method can't be used in multi-tenant environment such as grafana cloud
func (pCtx *PluginContext) GetSettingAsBoolFromEnv(key string, defaultValue bool) (output bool, err error) {
	if pCtx == nil {
		return defaultValue, errors.New("invalid plugin context")
	}
	strValue := pCtx.GetSettingFromEnv(key)
	if strValue == "" {
		return defaultValue, nil
	}
	value, err := strconv.ParseBool(strValue)
	if err != nil {
		return defaultValue, fmt.Errorf("environment variable '%s' is invalid bool value '%s'", key, strValue)
	}
	return value, nil
}
