package serf

import (
	"encoding/json"
	"github.com/blang/receptor/pipeline"
	serfc "github.com/hashicorp/serf/client"
	"log"
	"time"
)

var PULL_INTERVAL = 5 * time.Second

type Watcher struct {
}

type ServiceConfig struct {
	Addr    string `json:"addr"`
	AuthKey string `json:"authkey"`
}

func (w *Watcher) Setup(cfgData json.RawMessage) error {
	return nil
}

func (w *Watcher) Accept(cfgData json.RawMessage) (pipeline.Endpoint, error) {
	var serviceCfg ServiceConfig
	err := json.Unmarshal(cfgData, &serviceCfg)
	if err != nil {
		return nil, err
	}

	serfConfig := &serfc.Config{
		Addr:    serviceCfg.Addr,
		AuthKey: serviceCfg.AuthKey,
	}

	client, err := serfc.ClientFromConfig(serfConfig)
	if err != nil {
		return nil, err
	}

	return pipeline.EndpointFunc(func(eventCh chan pipeline.Event, closeCh chan struct{}) {
		defer func() {
			client.Close()
			close(eventCh)
		}()
		for {
			select {
			case <-closeCh:
				return
			case <-time.After(PULL_INTERVAL):
				members, err := client.Members()
				if err != nil {
					log.Printf("Error while fetching serf members: %s\n", err)
					break
				}
				for _, m := range members {
					eventCh <- &pipeline.SingleNode{
						EName: m.Name,
						EType: pipeline.EventNodeUp,
						EHost: m.Addr.String(),
						EPort: int(m.Port),
					}
				}
			}
		}

	}), nil
}
