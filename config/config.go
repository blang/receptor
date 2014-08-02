package config

// TODO: Move to receptor package

import (
	"encoding/json"
	"os"
)

type ServiceConfig struct {
	Watchers map[string]json.RawMessage `json:"watchers"`
	Reactors map[string]json.RawMessage `json:"reactors"`
}

type Config struct {
	Services map[string]ServiceConfig   `json:"services"`
	Watchers map[string]json.RawMessage `json:"watchers"`
	Reactors map[string]json.RawMessage `json:"reactors"`
}

func ReadFromFile(filename string) (*Config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var config Config
	err = json.NewDecoder(f).Decode(&config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
