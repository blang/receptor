package dummy

import (
	"encoding/json"
	"github.com/blang/receptor/pipe"
	"time"
)

type DummyWatcher struct {
}

type DummyServiceCfg struct {
	Name string `json:"name"`
	Host string `json:"host"`
	Port uint16 `json:"port"`
}

func (w *DummyWatcher) Setup(_ json.RawMessage) error {
	return nil
}

func (w *DummyWatcher) Accept(cfg json.RawMessage) (pipe.Endpoint, error) {
	serviceCfg := &DummyServiceCfg{
		Name: "TestNode",
		Host: "127.0.0.1",
		Port: 80,
	}
	err := json.Unmarshal(cfg, serviceCfg)
	if err != nil {
		return nil, err
	}
	return pipe.EndpointFunc(func(eventCh chan pipe.Event, closeCh chan struct{}) {
		status := true
		upEv := pipe.NewEventWithNode(serviceCfg.Name, pipe.NodeUp, serviceCfg.Host, serviceCfg.Port)
		downEv := pipe.NewEventWithNode(serviceCfg.Name, pipe.NodeDown, serviceCfg.Host, serviceCfg.Port)
		for {
			select {
			case <-time.After(2 * time.Second):
				if status {
					eventCh <- upEv
				} else {
					eventCh <- downEv
				}
				status = !status
			case <-closeCh:
				close(eventCh)
				return
			}
		}
	}), nil
}
