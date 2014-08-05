package handler

import (
	"errors"
	"github.com/blang/receptor/pipeline"
	"time"
)

type Handler interface {
	// Started in seperate go routine
	Handle(eventCh chan pipeline.Event, closeCh chan struct{})
}

type HandlerFunc func(eventCh chan pipeline.Event, closeCh chan struct{})

func (f HandlerFunc) Handle(eventCh chan pipeline.Event, closeCh chan struct{}) {
	f(eventCh, closeCh)
}

// ManagedHandler wraps a Handler to control its shutdown behaviour.
type ManagedHandler struct {
	Handler Handler
	DoneCh  chan struct{}
	CloseCh chan struct{}
}

var ERROR_HANDLER_WAIT_TIMEOUT = errors.New("Handler wait timed out")

// NewManagedHandler creates a wrapper around an handler which manages the shutdown behaviour.
func NewManagedHandler(handle Handler) *ManagedHandler {
	return &ManagedHandler{
		Handler: handle,
		DoneCh:  make(chan struct{}),
		CloseCh: make(chan struct{}),
	}
}

// Handle starts the wrapped handler on the given eventChannel and closes the DoneCh if handler returns.
// Blocks until handler returns.
func (m *ManagedHandler) Handle(eventCh chan pipeline.Event) {
	m.Handler.Handle(eventCh, m.CloseCh)
	close(m.DoneCh)
}

// Stop signals the handler to exit by using its close channel, stop will not close the event channel.
func (m *ManagedHandler) Stop() {
	close(m.CloseCh)
}

// Wait waits for the handler to stop.
func (m *ManagedHandler) Wait() {
	<-m.DoneCh
}

// WaitTimeout waits for the handler to stop or timeout to occur.
func (m *ManagedHandler) WaitTimeout(timeout time.Duration) error {
	select {
	case <-m.DoneCh:
		return nil
	case <-time.After(timeout):
		return ERROR_HANDLER_WAIT_TIMEOUT
	}
}
