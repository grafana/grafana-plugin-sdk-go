package backend

import (
	"encoding/json"
	"time"
)

// User represents the Grafana user.
type User struct {
	Login string
	Name  string
	Email string
	Role  string
}

// DataSourceConfig holds configuration for a data source instance.
type DataSourceConfig struct {
	// ID is the Grafana assigned numeric identifier of the the data source instance.
	ID int64

	// Name is the configured name of the data source instance.
	Name string

	// URL is the configured URL of a data source instance (e.g. the URL of an API endpoint).
	URL string

	// User is a configured user for a data source instance. This is not a Grafana user, rather an arbitrary string.
	User string

	// Database is the configured database for a data source instance. (e.g. the default Database a SQL data source would connect to).
	Database string

	// BasicAuthEnabled indicates if this data source instance should use basic authentication.
	BasicAuthEnabled bool

	// BasicAuthUser is the configured user for basic authentication. (e.g. when a data source uses basic authentication to connect to whatever API it fetches data from).
	BasicAuthUser string

	// JSONData contains the raw DataSourceConfig as JSON as stored by Grafana server. It repeats the properties in this object and includes custom properties.
	JSONData json.RawMessage

	// DecryptedSecureJSONData contains key,value pairs where the encrypted configuration in Grafana server have been decrypted before passing them to the plugin.
	DecryptedSecureJSONData map[string]string

	// Updated is the last time the configuration for the data source instance was updated.
	Updated time.Time
}

// PluginConfig holds the configuration for a plugin instance.
//
// Grafana supports multiple organizations and only one plugin instance per Grafana organization. A plugin instance can have multiple data source instances.
//
// PluginConfig is attached to incoming requests to uniquely identify the Plugin instance the request belongs to.
// If the request is a data source request, it also contains the configuration of the data source instance.
type PluginConfig struct {
	// OrgID is the Grafana identifier of the Grafana organization this plugin instance belongs too.
	OrgID int64

	// PluginID is the unique identifer name of the plugin instance.
	PluginID string

	// JSONData repeats the properties at this level of the object (excluding DataSourceConfig), and also includes any custom properties associated with the plugin config instance.
	JSONData json.RawMessage

	// DecryptedSecureJSONData contains key,value pairs where the encrypted configuration plugin instance in Grafana server have been decrypted before passing them to the plugin.
	DecryptedSecureJSONData map[string]string

	// Updated is the last time this plugin instance's configuration was updated.
	Updated time.Time

	// DataSourceConfig is the configuration of a data source instance for this plugin.
	// If the request is a data source request, this will be set to the specific datasource instance
	// the request is for.
	DataSourceConfig *DataSourceConfig
}
