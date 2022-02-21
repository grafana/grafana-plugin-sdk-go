package config

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	Address string   `json:"address"`
	Path    string   `json:"path"`
	Hosts   []string `json:"hosts"`
}

func LoadConfig(path string) (*Config, error) {
	if path == "" {
		path = "proxy.json"
	}
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		return &Config{
			Address: ":9999",
			Path:    "fixtures/e2e.har",
			Hosts:   make([]string, 0),
		}, nil
	}
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
