package handler

import (
	"github.com/blang/receptor/event"
)

type Handler interface {
	// Started in seperate go routine
	Handle(eventCh chan event.Event, closeCh chan struct{})
}

type HandlerFunc func(eventCh chan event.Event, closeCh chan struct{})

func (f HandlerFunc) Handle(eventCh chan event.Event, closeCh chan struct{}) {
	f(eventCh, closeCh)
}
