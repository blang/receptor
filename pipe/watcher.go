package pipe

import (
	"encoding/json"
)

type Watcher interface {

	// Setup configures the generic watcher with a config
	Setup(json.RawMessage) error

	// Handle job and return a handler to start watching
	Accept(json.RawMessage) (Endpoint, error)
}
