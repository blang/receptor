package receptor

import (
	"github.com/blang/receptor/pipe"
	"sync"
	"time"
)

type Service struct {
	name      string
	reactors  map[string]*pipe.ManagedEndpoint
	watchers  map[string]*pipe.ManagedEndpoint
	DoneCh    chan struct{} // Is closed if all service components are shut down
	FailureCh chan struct{} // Is closed if a failure occurs
}

func NewService(name string) *Service {
	return &Service{
		name:      name,
		reactors:  make(map[string]*pipe.ManagedEndpoint),
		watchers:  make(map[string]*pipe.ManagedEndpoint),
		DoneCh:    make(chan struct{}),
		FailureCh: make(chan struct{}),
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
	s.shutdownHook()

}

// Stop stops the service and all its watchers and reactors.
// Blocks until all components are stopped or reach timeout.
// Closes service doneCh channel.
func (s *Service) Stop(timeout time.Duration) {
	s.Shutdown()
	for _, manWatcher := range s.watchers {
		manWatcher.WaitTimeout(timeout)
	}

	for _, manReactor := range s.reactors {
		manReactor.WaitTimeout(timeout)
	}
}

// Shutdown sends a stop signal to all watchers and reactors. Does not block.
func (s *Service) Shutdown() {
	for _, manWatcher := range s.watchers {
		manWatcher.Stop()
	}

	for _, manReactor := range s.reactors {
		manReactor.Stop()
	}
}

// shutdownHook registers a hook on all watchers and reactors
// and shuts down the whole service if one component fails.
// Closes Service.DoneCh if all components are done. Does not block.
func (s *Service) shutdownHook() {
	failureCh := make(chan struct{}, len(s.watchers)+len(s.reactors))
	wg := sync.WaitGroup{}
	wg.Add(len(s.watchers) + len(s.reactors))
	for _, manWatcher := range s.watchers {
		go func(manEndpoint *pipe.ManagedEndpoint) {
			<-manEndpoint.DoneCh
			failureCh <- struct{}{}
			wg.Done()
		}(manWatcher)
	}
	for _, manReactor := range s.reactors {
		go func(manEndpoint *pipe.ManagedEndpoint) {
			<-manEndpoint.DoneCh
			failureCh <- struct{}{}
			wg.Done()
		}(manReactor)
	}

	// Shutdown service if component fails
	// Close doneCh if all components are down
	go func() {
		<-failureCh
		close(s.FailureCh)
		s.Shutdown()
		wg.Wait()
		close(s.DoneCh)
	}()
}
