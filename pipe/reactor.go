package pipe

import (
	"encoding/json"
)

// Reactor is the sink of events, it "reacts" on incoming events.
type Reactor interface {

	// Setup configures the reactor with a global config
	Setup(cfg json.RawMessage) error

	// Accept configures an instance of the reactor dedicated to a service.
	// Returns an Endpoint ready to communicate on the pipeline.
	Accept(cfg json.RawMessage) (Endpoint, error)
}
