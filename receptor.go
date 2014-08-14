package receptor

import (
	"fmt"
	"github.com/blang/receptor/pipe"
	"sync"
	"time"
)

var (
	Watchers = make(map[string]pipe.Watcher)
	Reactors = make(map[string]pipe.Reactor)
)

var SERVICE_STOP_TIMEOUT = 5 * time.Second

type Receptor struct {
	Services []*Service
}

func NewReceptor() *Receptor {
	return &Receptor{}
}

func (r *Receptor) Init(cfg *Config) error {
	services, err := Setup(cfg)
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
			service.Stop(SERVICE_STOP_TIMEOUT)
			wg.Done()
		}()
	}
	wg.Wait()
}

// Setup sets up all services defined by config.
func Setup(cfg *Config) ([]*Service, error) {
	err := SetupGlobalConfig(cfg)
	if err != nil {
		return nil, err
	}

	services, err := SetupServices(cfg.Services)
	if err != nil {
		return nil, err
	}

	return services, nil
}

// SetupGlobalConfig configures watchers and reactors with their global config.
func SetupGlobalConfig(cfg *Config) error {
	for reactName, reactCfg := range cfg.Reactors {
		react, ok := Reactors[reactName]
		if !ok {
			return fmt.Errorf("Could not configure react %s, react not found", reactName)
		}
		err := react.Setup(reactCfg)
		if err != nil {
			return fmt.Errorf("Could not configure react %s, react setup failed: %s", err)
		}
	}

	for watcherName, watcherCfg := range cfg.Watchers {
		watcher, ok := Watchers[watcherName]
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

// SetupServices sets up multiple services and their components defined by their service config.
func SetupServices(serviceCfgs map[string]ServiceConfig) ([]*Service, error) {
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

// SetupService sets up all watchers and reactors of the service with their service specific configuration.
func SetupService(name string, cfg ServiceConfig) (*Service, error) {
	service := NewService(name)

	for actorName, actorCfg := range cfg.Watchers {
		handle, err := SetupWatcher(actorCfg)
		if err != nil {
			return nil, fmt.Errorf("Service %s, Watcher %s, Setup error: %s", service.Name, actorName, err)
		}
		service.AddWatcherEndpoint(actorName, pipe.NewManagedEndpoint(handle))
	}

	for actorName, actorCfg := range cfg.Reactors {
		handle, err := SetupReactor(actorCfg)
		if err != nil {
			return nil, fmt.Errorf("Service %s, Reactor %s, Setup error: %s", service.Name, actorName, err)
		}
		service.AddReactorEndpoint(actorName, pipe.NewManagedEndpoint(handle))
	}
	return service, nil
}

// SetupWatcher registers a watcher with the service specific config.
func SetupWatcher(cfg ActorConfig) (pipe.Endpoint, error) {
	watcher, ok := Watchers[cfg.Type]
	if !ok {
		return nil, fmt.Errorf("Could not setup watcher %s, watcher type not found", cfg.Type)
	}
	handler, err := watcher.Accept(cfg.Config)
	if err != nil {
		return nil, fmt.Errorf("Could not setup watcher %s, watcher did not accept service config: %s", cfg.Type, err)
	}
	return handler, nil
}

// SetupReactor registers a reactor with the service specific config.
func SetupReactor(cfg ActorConfig) (pipe.Endpoint, error) {
	react, ok := Reactors[cfg.Type]
	if !ok {
		return nil, fmt.Errorf("Could not setup reactor %s, reactor type not found", cfg.Type)
	}
	handler, err := react.Accept(cfg.Config)
	if err != nil {
		return nil, fmt.Errorf("Could not setup reactor %s, reactor did not accept service config: %s", cfg.Type, err)
	}
	return handler, nil
}
