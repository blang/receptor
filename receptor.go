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
	Reactors []handler.Handler
	Watchers []handler.Handler
	EventCh  chan event.Event
	CloseCh  chan struct{}
}

type Receptor struct {
	Services []*Service
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
		// TODO: Add Event Merger middleware
		service.EventCh = make(chan event.Event)
		service.CloseCh = make(chan struct{})
		var outChs []chan event.Event

		// Start Reactors
		for _, reactorHandler := range service.Reactors {
			outCh := make(chan event.Event)
			outChs = append(outChs, outCh)

			// Add Congestion control before each reactor
			controlledOutCh := event.Merger(outCh)

			go reactorHandler.Handle(controlledOutCh, service.CloseCh)
		}

		// Broadcast from EventCh to all outChs
		event.Broadcaster(service.EventCh, outChs)

		// Start Watchers
		for _, watchHandler := range service.Watchers {
			go watchHandler.Handle(service.EventCh, service.CloseCh)
		}

	}
}

func (r *Receptor) Stop() {
	for _, service := range r.Services {
		close(service.EventCh)
	}
}

func (r *Receptor) SetupByConfig(cfg config.Config) ([]*Service, error) {
	err := setupDefaultConfig(cfg)
	if err != nil {
		return nil, err
	}

	services, err := setupServices(cfg.Services)
	if err != nil {
		return nil, err
	}

	return services, nil
}

func setupServices(serviceCfgs map[string]config.ServiceConfig) ([]*Service, error) {
	var services []*Service
	for serviceName, serviceCfg := range serviceCfgs {
		service := &Service{}
		service.Name = serviceName

		for watcherName, watcherCfg := range serviceCfg.Watchers {
			watcher, ok := discovery.Watchers[watcherName]
			if !ok {
				return nil, fmt.Errorf("Could not setup watcher %s for service %s, watcher not found", watcherName, serviceName)
			}
			handler, err := watcher.Accept(watcherCfg)
			if err != nil {
				return nil, fmt.Errorf("Could not setup watcher %s for service %s, watcher did not accept service config: %s", watcherName, serviceName, err)
			}
			service.Watchers = append(service.Watchers, handler)
		}

		for reactName, reactCfg := range serviceCfg.Reactors {
			react, ok := reactor.Reactors[reactName]
			if !ok {
				return nil, fmt.Errorf("Could not setup reactor %s for service %s, reactor not found", reactName, serviceName)
			}
			handler, err := react.Accept(reactCfg)
			if err != nil {
				return nil, fmt.Errorf("Could not setup reactor %s for service %s, reactor did not accept service config: %s", reactName, serviceName, err)
			}
			service.Reactors = append(service.Reactors, handler)
		}
		services = append(services, service)
	}
	return services, nil
}

func setupDefaultConfig(cfg config.Config) error {
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
