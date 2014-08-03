package handler

import (
	"github.com/blang/receptor/event"
)

// TODO: Watcher needs to be shutted down, therefore needs shutdown channel
type Handler interface {
	// Started in seperate go routine
	Handle(eventCh chan event.Event, doneCh chan struct{})
}

type HandlerFunc func(eventCh chan event.Event, doneCh chan struct{})

func (f HandlerFunc) Handle(eventCh chan event.Event, doneCh chan struct{}) {
	f(eventCh, doneCh)
}
