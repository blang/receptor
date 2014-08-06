package receptor

import (
	"github.com/blang/receptor/pipeline"
	"time"
)

type Service struct {
	name     string
	reactors map[string]*pipeline.ManagedEndpoint
	watchers map[string]*pipeline.ManagedEndpoint
}

func NewService(name string) *Service {
	return &Service{
		name:     name,
		reactors: make(map[string]*pipeline.ManagedEndpoint),
		watchers: make(map[string]*pipeline.ManagedEndpoint),
	}
}

func (s *Service) Name() string {
	return s.name
}

func (s *Service) AddReactorEndpoint(name string, endpoint *pipeline.ManagedEndpoint) {
	s.reactors[name] = endpoint
}

func (s *Service) AddWatcherEndpoint(name string, endpoint *pipeline.ManagedEndpoint) {
	s.watchers[name] = endpoint
}

// Start starts the service.
// It creates a pipeline between all watchers and reactors and starts them, does not block.
func (s *Service) Start() {
	eventCh := make(chan pipeline.Event)
	var outChs []chan pipeline.Event

	// Start Reactors
	for _, reactorManHandler := range s.reactors {
		outCh := make(chan pipeline.Event)
		outChs = append(outChs, outCh)

		// Add Congestion control before each reactor
		controlledOutCh := pipeline.Merger(outCh)

		go reactorManHandler.Handle(controlledOutCh)
	}

	// Broadcast from EventCh to all reactors
	pipeline.Broadcaster(eventCh, outChs)

	// Forwarder will forward each watchchannel to eventCh
	forwarder := pipeline.NewForwarder(eventCh)

	// Start Watchers
	for _, watchManHandler := range s.watchers {
		watcherEventCh := make(chan pipeline.Event)
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
		manWatcher.WaitTimeout(timeout) // TODO: Handle timeout error
	}

	for _, manReactor := range s.reactors {
		manReactor.Stop()
		manReactor.WaitTimeout(timeout) // TODO: Handle timeout error
	}
}
