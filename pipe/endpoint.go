package pipe

import (
	"errors"
	"time"
)

type Endpoint interface {
	// Started in seperate go routine
	Handle(eventCh chan Event, closeCh chan struct{})
}

type EndpointFunc func(eventCh chan Event, closeCh chan struct{})

func (f EndpointFunc) Handle(eventCh chan Event, closeCh chan struct{}) {
	f(eventCh, closeCh)
}

// ManagedHandler wraps a Handler to control its shutdown behaviour.
type ManagedEndpoint struct {
	Endpoint Endpoint
	DoneCh   chan struct{}
	CloseCh  chan struct{}
	closed   bool
}

var ERROR_HANDLER_WAIT_TIMEOUT = errors.New("Handler wait timed out")

// NewManagedHandler creates a wrapper around an handler which manages the shutdown behaviour.
func NewManagedEndpoint(handle Endpoint) *ManagedEndpoint {
	return &ManagedEndpoint{
		Endpoint: handle,
		DoneCh:   make(chan struct{}),
		CloseCh:  make(chan struct{}),
	}
}

// Handle starts the wrapped handler on the given eventChannel and closes the DoneCh if handler returns.
// Blocks until handler returns.
func (m *ManagedEndpoint) Handle(eventCh chan Event) {
	m.Endpoint.Handle(eventCh, m.CloseCh)
	close(m.DoneCh)
}

// Stop signals the handler to exit by using its close channel, stop will not close the event channel.
func (m *ManagedEndpoint) Stop() {
	if !m.closed {
		close(m.CloseCh)
		m.closed = true
	}
}

// Wait waits for the handler to stop.
func (m *ManagedEndpoint) Wait() {
	<-m.DoneCh
}

// WaitTimeout waits for the handler to stop or timeout to occur.
func (m *ManagedEndpoint) WaitTimeout(timeout time.Duration) error {
	select {
	case <-m.DoneCh:
		return nil
	case <-time.After(timeout):
		return ERROR_HANDLER_WAIT_TIMEOUT
	}
}
