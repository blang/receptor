package receptor

import (
	"fmt"
	"github.com/blang/receptor/config"
	"github.com/blang/receptor/discovery"
	"github.com/blang/receptor/event"
	"github.com/blang/receptor/handler"
	"github.com/blang/receptor/reactor"
	"sync"
	"time"
)

type Service struct {
	Name         string
	Reactors     map[string]*ManagedHandler
	Watchers     map[string]*ManagedHandler
	EventCh      chan event.Event
	CloseTimeout time.Duration
	DoneCh       chan struct{}
}

func NewService() *Service {
	return &Service{
		Reactors:     make(map[string]*ManagedHandler),
		Watchers:     make(map[string]*ManagedHandler),
		EventCh:      make(chan event.Event),
		CloseTimeout: 5 * time.Second,
		DoneCh:       make(chan struct{}),
	}
}

type ManagedHandler struct {
	Handler handler.Handler
	DoneCh  chan struct{}
	CloseCh chan struct{}
}

// NewManagedHandler creates a wrapper around an handler which manages the shutdown behaviour.
func NewManagedHandler(handle handler.Handler) *ManagedHandler {
	return &ManagedHandler{
		Handler: handle,
		DoneCh:  make(chan struct{}),
		CloseCh: make(chan struct{}),
	}
}

// Handle starts the wrapped handler on the given eventChannel and closes the DoneCh if handler returns.
// Blocks until handler returns.
func (m *ManagedHandler) Handle(eventCh chan event.Event) {
	m.Handler.Handle(eventCh, m.CloseCh)
	close(m.DoneCh)
}

// Stop signals the handler to exit.
func (m *ManagedHandler) Stop() {
	close(m.CloseCh)
}

type Receptor struct {
	Services []*Service
}

func NewReceptor() *Receptor {
	return &Receptor{}
}

func (r *Receptor) Init(cfg config.Config) error {
	services, err := r.SetupByConfig(cfg)
	if err != nil {
		return err
	}
	r.Services = services
	return nil
}

func (r *Receptor) Start() {
	for _, service := range r.Services {
		service.Start()
	}
}

// Stop stops all registered services and blocks until all stopped or reached timeout.
func (r *Receptor) Stop() {
	wg := sync.WaitGroup{}
	wg.Add(len(r.Services))
	for _, service := range r.Services {
		go func() {
			service.Stop()
			wg.Done()
		}()
	}
	wg.Wait()
}

// Start starts the service.
// It creates a pipeline between all watchers and reactors and starts them.
func (s *Service) Start() {
	var outChs []chan event.Event

	// Start Reactors
	for _, reactorManHandler := range s.Reactors {
		outCh := make(chan event.Event)
		outChs = append(outChs, outCh)

		// Add Congestion control before each reactor
		controlledOutCh := event.Merger(outCh)

		go reactorManHandler.Handle(controlledOutCh)
	}

	// Broadcast from EventCh to all outChs
	event.Broadcaster(s.EventCh, outChs)

	// Start Watchers
	for _, watchManHandler := range s.Watchers {
		go watchManHandler.Handle(s.EventCh)
	}
}

// Stop stops the service and all its watchers and reactors.
// Blocks until all components are stopped or reach timeout.
// Closes service doneCh channel.
func (s *Service) Stop() {
	close(s.EventCh)
	for _, manWatcher := range s.Watchers {
		select {
		case <-manWatcher.DoneCh:
		case <-time.After(s.CloseTimeout):
			//TODO: Handle timeout, error message
		}
	}

	for _, manReactor := range s.Reactors {
		select {
		case <-manReactor.DoneCh:
		case <-time.After(s.CloseTimeout):
			//TODO: Handle timeout, error message
		}
	}
	close(s.DoneCh)
}

func (r *Receptor) SetupByConfig(cfg config.Config) ([]*Service, error) {
	err := SetupDefaultConfig(cfg)
	if err != nil {
		return nil, err
	}

	services, err := SetupServices(cfg.Services)
	if err != nil {
		return nil, err
	}

	return services, nil
}

func SetupServices(serviceCfgs map[string]config.ServiceConfig) ([]*Service, error) {
	var services []*Service
	for serviceName, serviceCfg := range serviceCfgs {
		service, err := SetupService(serviceName, serviceCfg)
		if err != nil {
			return nil, fmt.Errorf("Could not setup service %s: %s", serviceName, err)
		}
		services = append(services, service)
	}
	return services, nil
}

func SetupService(name string, cfg config.ServiceConfig) (*Service, error) {
	service := NewService()
	service.Name = name

	for actorName, actorCfg := range cfg.Watchers {
		handler, err := SetupWatcher(actorCfg)
		if err != nil {
			return nil, fmt.Errorf("Service %s, Watcher %s, Setup error: %s", service.Name, actorName, err)
		}
		service.Watchers[actorName] = NewManagedHandler(handler)
	}

	for actorName, actorCfg := range cfg.Reactors {
		handler, err := SetupReactor(actorCfg)
		if err != nil {
			return nil, fmt.Errorf("Service %s, Reactor %s, Setup error: %s", service.Name, actorName, err)
		}
		service.Reactors[actorName] = NewManagedHandler(handler)
	}
	return service, nil
}

func SetupWatcher(cfg config.ActorConfig) (handler.Handler, error) {
	watcher, ok := discovery.Watchers[cfg.Type]
	if !ok {
		return nil, fmt.Errorf("Could not setup watcher %s, watcher type not found", cfg.Type)
	}
	handler, err := watcher.Accept(cfg.Config)
	if err != nil {
		return nil, fmt.Errorf("Could not setup watcher %s, watcher did not accept service config: %s", cfg.Type, err)
	}
	return handler, nil
}

func SetupReactor(cfg config.ActorConfig) (handler.Handler, error) {
	react, ok := reactor.Reactors[cfg.Type]
	if !ok {
		return nil, fmt.Errorf("Could not setup reactor %s, reactor type not found", cfg.Type)
	}
	handler, err := react.Accept(cfg.Config)
	if err != nil {
		return nil, fmt.Errorf("Could not setup reactor %s, reactor did not accept service config: %s", cfg.Type, err)
	}
	return handler, nil
}

func SetupDefaultConfig(cfg config.Config) error {
	for reactName, reactCfg := range cfg.Reactors {
		react, ok := reactor.Reactors[reactName]
		if !ok {
			return fmt.Errorf("Could not configure react %s, react not found", reactName)
		}
		err := react.Setup(reactCfg)
		if err != nil {
			return fmt.Errorf("Could not configure react %s, react setup failed: %s", err)
		}
	}

	for watcherName, watcherCfg := range cfg.Watchers {
		watcher, ok := discovery.Watchers[watcherName]
		if !ok {
			return fmt.Errorf("Could not configure watcher %s, watcher not found", watcherName)
		}
		err := watcher.Setup(watcherCfg)
		if err != nil {
			return fmt.Errorf("Could not configure watcher %s, watcher setup failed: %s", err)
		}
	}
	return nil
}
