package reactor

import (
	"encoding/json"
	"github.com/blang/receptor/pipeline"
)

var Reactors = make(map[string]Reactor)

type Reactor interface {
	Setup(json.RawMessage) error
	Accept(json.RawMessage) (pipeline.Endpoint, error)
}
