package config

import (
	"encoding/json"
	"io/ioutil"
)

// Config is the configuration for the proxy.
type Config struct {
	Storage  *StorageConfig `json:"storage"`
	Address  string         `json:"address"`
	Hosts    []string       `json:"hosts"`
	CAConfig CAConfig       `json:"ca_keypair"`
}

// StorageConfig defines the storage configuration for the proxy.
type StorageConfig struct {
	Type StorageType `json:"type"`
	Path string      `json:"path"`
}

// CAConfig contains the keypair for the CA.
type CAConfig struct {
	Cert       string `json:"cert"`
	PrivateKey string `json:"private_key"`
}

// StorageType defines the type of storage used by the proxy.
type StorageType string

const (
	// StorageTypeHAR is the HAR file storage type.
	StorageTypeHAR StorageType = "har"
)

// LoadConfig loads the configuration from a JSON file path.
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		path = "proxy.json"
	}
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return &Config{
			Storage: &StorageConfig{
				Type: StorageTypeHAR,
				Path: "fixtures/e2e.har",
			},
			Address: "127.0.0.1:9999",
			Hosts:   make([]string, 0),
		}, nil
	}
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
