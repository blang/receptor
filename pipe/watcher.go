package pipe

import (
	"encoding/json"
)

// Watcher is the source of events, it "watches" a system and generates
// events which are then send down the pipeline.
type Watcher interface {

	// Setup configures the watcher with a global config
	Setup(cfg json.RawMessage) error

	// Accept configures an instance of the watcher dedicated to a service.
	// Returns an Endpoint ready to communicate on the pipeline.
	Accept(cfg json.RawMessage) (Endpoint, error)
}
