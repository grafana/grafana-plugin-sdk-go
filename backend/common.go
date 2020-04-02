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

// DataSourceConfig holds configuration for a datasource instance.
type DataSourceConfig struct {
	ID                      int64             // ID is the numeric identifer of the the datasource instance.
	Name                    string            // Name is the name of the datasource instance.
	URL                     string            // URL is the configured URL of a datasource instance (e.g. the url of an API endpoint).
	User                    string            // User is a configured user for a datasource instance. This is not a grafana user, rather an arbitrary string.
	Database                string            // Database is the configured database for a datasource instance. (e.g. the default Database a SQL datasource would connect to).
	BasicAuthEnabled        bool              // BasicAuthEnabled indicates if this datasource instance should use basic auth.
	BasicAuthUser           string            // BasicAuthUser is the configured user for basic authentication. (e.g. when a datasource uses basic auth to connect to whatever API it fetches data from).
	JSONData                json.RawMessage   // JSONData contains the raw DataSourceConfig as JSON as stored by Grafana server. It repeats the properties in this object and includes custom properties.
	DecryptedSecureJSONData map[string]string // DecryptedSecureJSONData contains key,value pairs where the encrypted configuration in Grafana server have been decrypted before passing them to the plugin.
	Updated                 time.Time         // Updated is the last time the configuration for the datasource instance was updated.
}

// PluginConfig configuration for a plugin.
type PluginConfig struct {
	OrgID                   int64
	PluginID                string
	JSONData                json.RawMessage
	DecryptedSecureJSONData map[string]string
	Updated                 time.Time
	DataSourceConfig        *DataSourceConfig
}
