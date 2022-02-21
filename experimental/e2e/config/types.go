package config

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	Storage *StorageConfig `json:"storage"`
	Address string         `json:"address"`
	Hosts   []string       `json:"hosts"`
}

type StorageConfig struct {
	Type StorageType `json:"type"`
	Path string      `json:"path"`
}

type StorageType string

const (
	StorageTypeHAR StorageType = "har"
)

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
