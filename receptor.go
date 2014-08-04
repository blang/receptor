package receptor

import (
	"fmt"
	"github.com/blang/receptor/config"
	"github.com/blang/receptor/discovery"
	"github.com/blang/receptor/event"
	"github.com/blang/receptor/handler"
	"github.com/blang/receptor/reactor"
)

type Service struct {
	Name     string
	Reactors map[string]handler.Handler
	Watchers map[string]handler.Handler
	EventCh  chan event.Event
	DoneCh   chan struct{}
}

func NewService() *Service {
	return &Service{
		Reactors: make(map[string]handler.Handler),
		Watchers: make(map[string]handler.Handler),
		EventCh:  make(chan event.Event),
		DoneCh:   make(chan struct{}),
	}
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

func (r *Receptor) Run() {

	for _, service := range r.Services {
		var outChs []chan event.Event

		// Start Reactors
		for _, reactorHandler := range service.Reactors {
			outCh := make(chan event.Event)
			outChs = append(outChs, outCh)

			// Add Congestion control before each reactor
			controlledOutCh := event.Merger(outCh)

			go reactorHandler.Handle(controlledOutCh, service.DoneCh)
		}

		// Broadcast from EventCh to all outChs
		event.Broadcaster(service.EventCh, outChs)

		// Start Watchers
		for _, watchHandler := range service.Watchers {
			go watchHandler.Handle(service.EventCh, service.DoneCh)
		}

	}
}

func (r *Receptor) Stop() {
	for _, service := range r.Services {
		close(service.EventCh)
	}
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
		service.Watchers[actorName] = handler
	}

	for actorName, actorCfg := range cfg.Reactors {
		handler, err := SetupReactor(actorCfg)
		if err != nil {
			return nil, fmt.Errorf("Service %s, Reactor %s, Setup error: %s", service.Name, actorName, err)
		}
		service.Reactors[actorName] = handler
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
