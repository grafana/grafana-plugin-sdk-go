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

// DataSourceConfig configuration for a datasource plugin.
type DataSourceConfig struct {
	ID                      int64
	Name                    string
	URL                     string
	User                    string
	Database                string
	BasicAuthEnabled        bool
	BasicAuthUser           string
	JSONData                json.RawMessage
	DecryptedSecureJSONData map[string]string
	Updated                 time.Time
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
