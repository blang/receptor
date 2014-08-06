package receptor

// TODO: Move to receptor package

import (
	"encoding/json"
	"os"
)

type ServiceConfig struct {
	// Map unique user-defined name to actor config
	Watchers map[string]ActorConfig `json:"watchers"`
	Reactors map[string]ActorConfig `json:"reactors"`
}

type ActorConfig struct {
	Type   string          `json:"type"`
	Config json.RawMessage `json:"cfg"`
}

type Config struct {
	Services map[string]ServiceConfig   `json:"services"`
	Watchers map[string]json.RawMessage `json:"watchers"`
	Reactors map[string]json.RawMessage `json:"reactors"`
}

func NewConfigFromFile(filename string) (*Config, error) {
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
