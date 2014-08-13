package receptor

import (
	"fmt"
	"github.com/blang/receptor/pipe"
	"github.com/blang/receptor/plugin"
	"log"
	"sync"
	"time"
)

var (
	SERVICE_STOP_TIMEOUT = 5 * time.Second // Timeout for EACH service to cleanup
	PLUGIN_STOP_TIMEOUT  = 5 * time.Second // Timeout for EACH plugin process to cleanup
)

type Receptor struct {
	Services     []*Service
	PluginLookup plugin.LookupService
}

func NewReceptor(lookup plugin.LookupService) *Receptor {
	return &Receptor{
		PluginLookup: lookup,
	}
}

func (r *Receptor) Init(cfg *Config) error {
	services, err := r.Setup(cfg)
	if err != nil {
		return err
	}
	r.Services = services
	return nil
}

func (r *Receptor) Start() {
	for _, service := range r.Services {
		service.Start()
		log.Printf("[Service %s] started", service.Name())
	}
}

// Stop stops all registered services and blocks until all stopped or reached timeout.
func (r *Receptor) Stop() {
	wg := sync.WaitGroup{}
	wg.Add(len(r.Services))
	for _, service := range r.Services {
		go func(service *Service) {
			service.Stop(SERVICE_STOP_TIMEOUT)
			log.Printf("[Service %s] stopped", service.Name())
			wg.Done()
		}(service)
	}
	wg.Wait()
	r.PluginLookup.Cleanup(PLUGIN_STOP_TIMEOUT)
}

// Setup sets up all services defined by config.
func (r *Receptor) Setup(cfg *Config) ([]*Service, error) {
	err := r.SetupGlobalConfig(cfg)
	if err != nil {
		return nil, err
	}

	services, err := r.SetupServices(cfg.Services)
	if err != nil {
		return nil, err
	}

	return services, nil
	// return nil, nil
}

// SetupGlobalConfig configures watchers and reactors with their global config.
// Only setup watchers/reactors used by services.
func (r *Receptor) SetupGlobalConfig(cfg *Config) error {
	// Only setup watchers/reactors really needed
	filteredReactors := filterReactors(cfg)
	filteredWatchers := filterWatchers(cfg)

	for reactName, reactCfg := range cfg.Reactors {
		if _, ok := filteredReactors[reactName]; !ok {
			continue
		}
		react, err := r.PluginLookup.Reactor(reactName)
		if err != nil {
			return fmt.Errorf("Could not configure reactor %s: %s", reactName, err)
		}
		err = react.Setup(reactCfg)
		if err != nil {
			return fmt.Errorf("Could not configure react %s, react setup failed: %s", reactName, err)
		}
	}

	for watcherName, watcherCfg := range cfg.Watchers {
		if _, ok := filteredWatchers[watcherName]; !ok {
			continue
		}
		watcher, err := r.PluginLookup.Watcher(watcherName)
		if err != nil {
			return fmt.Errorf("Could not configure watcher: %s", err)
		}
		err = watcher.Setup(watcherCfg)
		if err != nil {
			return fmt.Errorf("Could not configure watcher %s, watcher setup failed: %s", watcherName, err)
		}
	}
	return nil
}

// filterWatchers creates a set of watchers needed for services
func filterWatchers(cfg *Config) map[string]struct{} {
	neededWatchers := make(map[string]struct{})
	for _, service := range cfg.Services {
		for _, watcher := range service.Watchers {
			neededWatchers[watcher.Type] = struct{}{}
		}
	}
	return neededWatchers
}

// filterReactors creates a set of reactors needed for services
func filterReactors(cfg *Config) map[string]struct{} {
	neededReactors := make(map[string]struct{})
	for _, service := range cfg.Services {
		for _, reactor := range service.Reactors {
			neededReactors[reactor.Type] = struct{}{}
		}
	}
	return neededReactors
}

// SetupServices sets up multiple services and their components defined by their service config.
func (r *Receptor) SetupServices(serviceCfgs map[string]ServiceConfig) ([]*Service, error) {
	var services []*Service
	for serviceName, serviceCfg := range serviceCfgs {
		service, err := r.SetupService(serviceName, serviceCfg)
		if err != nil {
			return nil, fmt.Errorf("Could not setup service %s: %s", serviceName, err)
		}
		services = append(services, service)
	}
	return services, nil
}

// SetupService sets up all watchers and reactors of the service with their service specific configuration.
func (r *Receptor) SetupService(name string, cfg ServiceConfig) (*Service, error) {
	service := NewService(name)

	for actorName, actorCfg := range cfg.Watchers {
		handle, err := r.SetupWatcher(actorCfg)
		if err != nil {
			return nil, fmt.Errorf("Service %s, Watcher %s, Setup error: %s", name, actorName, err)
		}
		log.Printf("[Service %s:%s] Setup done", name, actorName)
		service.AddWatcherEndpoint(actorName, pipe.NewManagedEndpoint(handle))
	}

	for actorName, actorCfg := range cfg.Reactors {
		handle, err := r.SetupReactor(actorCfg)
		if err != nil {
			return nil, fmt.Errorf("Service %s, Reactor %s, Setup error: %s", service.name, actorName, err)
		}
		log.Printf("[Service %s:%s] Setup done", name, actorName)
		service.AddReactorEndpoint(actorName, pipe.NewManagedEndpoint(handle))
	}
	return service, nil
}

// SetupWatcher registers a watcher with the service specific config.
func (r *Receptor) SetupWatcher(cfg ActorConfig) (pipe.Endpoint, error) {
	watcher, err := r.PluginLookup.Watcher(cfg.Type)
	if err != nil {
		return nil, fmt.Errorf("Could not setup watcher: %s", err)
	}
	handler, err := watcher.Accept(cfg.Config)
	if err != nil {
		return nil, fmt.Errorf("Could not setup watcher %s, watcher did not accept service config: %s", cfg.Type, err)
	}
	return handler, nil
}

// SetupReactor registers a reactor with the service specific config.
func (r *Receptor) SetupReactor(cfg ActorConfig) (pipe.Endpoint, error) {
	react, err := r.PluginLookup.Reactor(cfg.Type)
	if err != nil {
		return nil, fmt.Errorf("Could not setup reactor: %s", err)
	}
	handler, err := react.Accept(cfg.Config)
	if err != nil {
		return nil, fmt.Errorf("Could not setup reactor %s, reactor did not accept service config: %s", cfg.Type, err)
	}
	return handler, nil
}
