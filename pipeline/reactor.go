package pipeline

import (
	"encoding/json"
)

type Reactor interface {
	Setup(json.RawMessage) error
	Accept(json.RawMessage) (Endpoint, error)
}
