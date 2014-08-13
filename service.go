package receptor

import (
	"github.com/blang/receptor/pipe"
	"time"
)

type Service struct {
	name     string
	reactors map[string]*pipe.ManagedEndpoint
	watchers map[string]*pipe.ManagedEndpoint
}

func NewService(name string) *Service {
	return &Service{
		name:     name,
		reactors: make(map[string]*pipe.ManagedEndpoint),
		watchers: make(map[string]*pipe.ManagedEndpoint),
	}
}

func (s *Service) Name() string {
	return s.name
}

func (s *Service) AddReactorEndpoint(name string, endpoint *pipe.ManagedEndpoint) {
	s.reactors[name] = endpoint
}

func (s *Service) AddWatcherEndpoint(name string, endpoint *pipe.ManagedEndpoint) {
	s.watchers[name] = endpoint
}

// Start starts the service.
// It creates a pipe between all watchers and reactors and starts them, does not block.
func (s *Service) Start() {
	eventCh := make(chan pipe.Event)
	var outChs []chan pipe.Event

	// Start Reactors
	for _, reactorManHandler := range s.reactors {
		outCh := make(chan pipe.Event)
		outChs = append(outChs, outCh)

		// Add Congestion control before each reactor
		controlledOutCh := make(chan pipe.Event)
		pipe.Merger(outCh, controlledOutCh)

		go reactorManHandler.Handle(controlledOutCh)
	}

	// Broadcast from EventCh to all reactors
	pipe.Broadcaster(eventCh, outChs)

	// Forwarder will forward each watchchannel to eventCh
	forwarder := pipe.NewForwarder(eventCh)

	// Start Watchers
	for _, watchManHandler := range s.watchers {
		watcherEventCh := make(chan pipe.Event)
		forwarder.Forward(watcherEventCh) // Forward watcherEventCh to eventCh

		go watchManHandler.Handle(watcherEventCh)
	}

	// Close eventCh if all watcherEventChs are closed. Propagates to reactors.
	forwarder.WaitClose()

}

// Stop stops the service and all its watchers and reactors.
// Blocks until all components are stopped or reach timeout.
// Closes service doneCh channel.
func (s *Service) Stop(timeout time.Duration) {
	for _, manWatcher := range s.watchers {
		manWatcher.Stop()
		manWatcher.WaitTimeout(timeout)
	}

	for _, manReactor := range s.reactors {
		manReactor.Stop()
		manReactor.WaitTimeout(timeout)
	}
}
