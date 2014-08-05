package discovery

import (
	"encoding/json"
	"github.com/blang/receptor/pipeline"
)

var Watchers = make(map[string]Watcher)

type Watcher interface {

	// Setup configures the generic watcher with a config
	Setup(json.RawMessage) error

	// Handle job and return a handler to start watching
	Accept(json.RawMessage) (pipeline.Endpoint, error)
}
