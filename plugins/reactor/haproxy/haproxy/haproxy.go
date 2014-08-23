package haproxy

import (
	"bytes"
	"encoding/json"
	"github.com/blang/receptor/pipe"
	"strconv"
	"sync"
)

type HAProxyWatcher struct {
	sync.RWMutex
	globalConfig      *SetupConfig
	globalConfigCache string
	endpoints         map[*HAProxyEndpoint]struct{}
	daemonRunning     bool
}

type SetupConfig struct {
	ReloadCommand  string              `json:"reload_command"`
	ConfigFilePath string              `json:"config_file_path"`
	SocketFilePath string              `json:"socket_file_path"`
	DoSocket       bool                `json:"do_socket"`
	DoReloads      bool                `json:"do_reloads"`
	DoWrites       bool                `json:"do_writes"`
	Global         []string            `json:"global"`
	Defaults       []string            `json:"default"`
	ExtraSections  map[string][]string `json:"extra_sections"`
}

type ServiceConfig struct {
	ServiceName   string   `json:"service_name"`
	Frontend      []string `json:"frontend"`
	Backend       []string `json:"backend"`
	ServerOptions string   `json:"server_options"`
}

func (w *HAProxyWatcher) generateGlobalConfig() {
	var buffer bytes.Buffer
	buffer.WriteString("global\n")
	for _, cfgline := range w.globalConfig.Global {
		buffer.WriteString("\t")
		buffer.WriteString(cfgline)
		buffer.WriteString("\n")
	}
	buffer.WriteString("defaults\n")
	for _, cfgline := range w.globalConfig.Global {
		buffer.WriteString("\t")
		buffer.WriteString(cfgline)
		buffer.WriteString("\n")
	}
	for key, arr := range w.globalConfig.ExtraSections {
		buffer.WriteString(key)
		buffer.WriteString("\n")
		for _, cfgline := range arr {
			buffer.WriteString("\t")
			buffer.WriteString(cfgline)
			buffer.WriteString("\n")
		}
	}
	w.globalConfigCache = buffer.String()
}

func (w *HAProxyWatcher) Setup(cfgData json.RawMessage) error {
	cfg := &SetupConfig{}
	err := json.Unmarshal(cfgData, cfg)
	if err != nil {
		return err
	}
	w.globalConfig = cfg
	w.generateGlobalConfig()
	return nil
}

func (w *HAProxyWatcher) Accept(cfgData json.RawMessage) (pipe.Endpoint, error) {
	cfg := &ServiceConfig{}
	err := json.Unmarshal(cfgData, cfg)
	if err != nil {
		return nil, err
	}

	return NewHAProxyEndpoint(cfg), nil
}

func (w *HAProxyWatcher) monitorEndpoint(endpoint *HAProxyEndpoint) {
	w.Lock()
	defer w.Unlock()
	w.endpoints[endpoint] = struct{}{}
	if !w.daemonRunning {
		go w.Daemon()
		w.daemonRunning = true
	}
}

func (w *HAProxyWatcher) Daemon() {
	for {
		w.RLock()
		if len(w.endpoints) == 0 {
			w.daemonRunning = false
			w.Unlock()
			return
		}
		endpoints := w.endpoints //TODO: Check if this works with concurrency
		w.Unlock()

		var buffer bytes.Buffer
		for endpoint, _ := range endpoints {
			buffer.WriteString(endpoint.frontendConfigCache)
			buffer.WriteString(endpoint.backendConfigCache)
			ev, ok := <-endpoint.fullCh
			if !ok {
				continue
			}
			for _, nodeInfo := range ev {
				if nodeInfo.Status != pipe.NodeUp {
					continue
				}
				buffer.WriteString("server ")
				buffer.WriteString(endpoint.cfg.ServiceName)
				buffer.WriteString("_")
				buffer.WriteString(nodeInfo.Name)
				buffer.WriteString(" ")
				buffer.WriteString(nodeInfo.Host)
				buffer.WriteString(":")
				buffer.WriteString(strconv.FormatUint(uint64(nodeInfo.Port), 10))
				buffer.WriteString(" ")
				//TODO: Add cookie statement?
				buffer.WriteString(endpoint.cfg.ServerOptions)
				buffer.WriteString("\n")
			}
		}
		//TODO: Write buffer to file

	}
}

func (w *HAProxyWatcher) unmonitorEndpoint(endpoint *HAProxyEndpoint) {
	w.Lock()
	defer w.Unlock()
	delete(w.endpoints, endpoint)
}

func NewHAProxyEndpoint(cfg *ServiceConfig) *HAProxyEndpoint {
	endpoint := &HAProxyEndpoint{
		cfg: cfg,
	}
	endpoint.cacheFrontendConfig()
	endpoint.cacheBackendConfig()
	return endpoint
}

type HAProxyEndpoint struct {
	cfg                 *ServiceConfig
	watcher             *HAProxyWatcher
	frontendConfigCache string
	backendConfigCache  string
	fullCh              chan pipe.Event
}

func (e *HAProxyEndpoint) cacheFrontendConfig() {
	var buffer bytes.Buffer
	buffer.WriteString("frontend ")
	buffer.WriteString(e.cfg.ServiceName)
	buffer.WriteString("\n")
	for _, line := range e.cfg.Frontend {
		buffer.WriteString("\t")
		buffer.WriteString(line)
		buffer.WriteString("\n")
	}
	e.frontendConfigCache = buffer.String()
}

func (e *HAProxyEndpoint) cacheBackendConfig() {
	var buffer bytes.Buffer
	buffer.WriteString("backend ")
	buffer.WriteString(e.cfg.ServiceName)
	buffer.WriteString("\n")
	for _, line := range e.cfg.Backend {
		buffer.WriteString("\t")
		buffer.WriteString(line)
		buffer.WriteString("\n")
	}
	e.backendConfigCache = buffer.String()
}

func (e *HAProxyEndpoint) Handle(eventCh chan pipe.Event, closeCh chan struct{}) {
	e.fullCh = pipe.BookkeeperReceiver(eventCh)
	e.watcher.monitorEndpoint(e)
	<-closeCh
	e.watcher.unmonitorEndpoint(e)
}
