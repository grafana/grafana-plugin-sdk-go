package backend

import (
	"encoding/json"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
)

const dataCustomOptionsKey = "grafanaData"
const secureDataCustomOptionsKey = "grafanaSecureData"

// User represents a Grafana user.
type User struct {
	Login string
	Name  string
	Email string
	Role  string
}

// AppInstanceSettings represents settings for an app instance.
//
// In Grafana an app instance is an app plugin of certain
// type that have been configured and enabled in a Grafana organization.
type AppInstanceSettings struct {
	// JSONData repeats the properties at this level of the object (excluding DataSourceConfig), and also includes any
	// custom properties associated with the plugin config instance.
	JSONData json.RawMessage

	// DecryptedSecureJSONData contains key,value pairs where the encrypted configuration plugin instance in Grafana
	// server have been decrypted before passing them to the plugin.
	DecryptedSecureJSONData map[string]string

	// Updated is the last time this plugin instance's configuration was updated.
	Updated time.Time
}

// HTTPClientOptions creates httpclient.Options based on settings.
func (s *AppInstanceSettings) HTTPClientOptions() (httpclient.Options, error) {
	httpSettings, err := parseHTTPSettings(s.JSONData, s.DecryptedSecureJSONData)
	if err != nil {
		return httpclient.Options{}, err
	}

	opts := httpSettings.HTTPClientOptions()
	setCustomOptionsFromHTTPSettings(&opts, httpSettings)

	return opts, nil
}

// DataSourceInstanceSettings represents settings for a data source instance.
//
// In Grafana a data source instance is a data source plugin of certain
// type that have been configured and created in a Grafana organization.
type DataSourceInstanceSettings struct {
	// ID is the Grafana assigned numeric identifier of the the data source instance.
	ID int64

	// UID is the Grafana assigned string identifier of the the data source instance.
	UID string

	// Type is the unique identifier of the plugin that the request is for.
	// This should be the same value as PluginContext.PluginId.
	Type string

	// Name is the configured name of the data source instance.
	Name string

	// URL is the configured URL of a data source instance (e.g. the URL of an API endpoint).
	URL string

	// User is a configured user for a data source instance. This is not a Grafana user, rather an arbitrary string.
	User string

	// Database is the configured database for a data source instance.
	// Only used by Elasticsearch and Influxdb.
	// Please use JSONData to store information related to database.
	Database string

	// BasicAuthEnabled indicates if this data source instance should use basic authentication.
	BasicAuthEnabled bool

	// BasicAuthUser is the configured user for basic authentication. (e.g. when a data source uses basic
	// authentication to connect to whatever API it fetches data from).
	BasicAuthUser string

	// JSONData contains the raw DataSourceConfig as JSON as stored by Grafana server. It repeats the properties in
	// this object and includes custom properties.
	JSONData json.RawMessage

	// DecryptedSecureJSONData contains key,value pairs where the encrypted configuration in Grafana server have been
	// decrypted before passing them to the plugin.
	DecryptedSecureJSONData map[string]string

	// Updated is the last time the configuration for the data source instance was updated.
	Updated time.Time
}

// HTTPClientOptions creates httpclient.Options based on settings.
func (s *DataSourceInstanceSettings) HTTPClientOptions() (httpclient.Options, error) {
	httpSettings, err := parseHTTPSettings(s.JSONData, s.DecryptedSecureJSONData)
	if err != nil {
		return httpclient.Options{}, err
	}

	if s.BasicAuthEnabled {
		httpSettings.BasicAuthEnabled = s.BasicAuthEnabled
		httpSettings.BasicAuthUser = s.BasicAuthUser
		httpSettings.BasicAuthPassword = s.DecryptedSecureJSONData["basicAuthPassword"]
	} else if s.User != "" {
		httpSettings.BasicAuthEnabled = true
		httpSettings.BasicAuthUser = s.User
		httpSettings.BasicAuthPassword = s.DecryptedSecureJSONData["password"]
	}

	opts := httpSettings.HTTPClientOptions()
	opts.Labels["datasource_name"] = s.Name
	opts.Labels["datasource_uid"] = s.UID
	opts.Labels["datasource_type"] = s.Type

	setCustomOptionsFromHTTPSettings(&opts, httpSettings)

	return opts, nil
}

// PluginContext holds contextual information about a plugin request, such as
// Grafana organization, user and plugin instance settings.
type PluginContext struct {
	// OrgID is The Grafana organization identifier the request originating from.
	OrgID int64

	// PluginID is the unique identifier of the plugin that the request is for.
	PluginID string

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
}

func setCustomOptionsFromHTTPSettings(opts *httpclient.Options, httpSettings *HTTPSettings) {
	opts.CustomOptions = map[string]interface{}{}

	if httpSettings.JSONData != nil {
		opts.CustomOptions[dataCustomOptionsKey] = httpSettings.JSONData
	}

	if httpSettings.SecureJSONData != nil {
		opts.CustomOptions[secureDataCustomOptionsKey] = httpSettings.SecureJSONData
	}
}

// JSONDataFromHTTPClientOptions extracts JSON data from CustomOptions of httpclient.Options.
func JSONDataFromHTTPClientOptions(opts httpclient.Options) (res map[string]interface{}) {
	if opts.CustomOptions == nil {
		return
	}

	val, exists := opts.CustomOptions[dataCustomOptionsKey]
	if !exists {
		return
	}

	jsonData, ok := val.(map[string]interface{})
	if !ok {
		return
	}

	return jsonData
}

// SecureJSONDataFromHTTPClientOptions extracts secure JSON data from CustomOptions of httpclient.Options.
func SecureJSONDataFromHTTPClientOptions(opts httpclient.Options) (res map[string]string) {
	if opts.CustomOptions == nil {
		return
	}

	val, exists := opts.CustomOptions[secureDataCustomOptionsKey]
	if !exists {
		return
	}

	secureJSONData, ok := val.(map[string]string)
	if !ok {
		return
	}

	return secureJSONData
}
