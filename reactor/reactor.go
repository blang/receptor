package reactor

import (
	"encoding/json"
	"github.com/blang/receptor/handler"
)

var Reactors = make(map[string]Reactor)

type Reactor interface {
	Setup(json.RawMessage) error
	Accept(json.RawMessage) (handler.Handler, error)
}
