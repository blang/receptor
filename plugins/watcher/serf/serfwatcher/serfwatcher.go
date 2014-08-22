package serfwatcher

import (
	"encoding/json"
	"github.com/blang/receptor/pipe"
	serfc "github.com/hashicorp/serf/client"
	"log"
	"time"
)

var PULL_INTERVAL = 5 * time.Second
var RECONNECT_INTERVAL = 2 * time.Second

type Watcher struct {
}

type ServiceConfig struct {
	Addr    string            `json:"addr"`
	AuthKey string            `json:"authkey"`
	Tags    map[string]string `json:"tags"`
	TagHost string            `json:"tag_host"`
	TagPort string            `json:"tag_port"`
}

func (w *Watcher) Setup(cfgData json.RawMessage) error {
	return nil
}

func (w *Watcher) Accept(cfgData json.RawMessage) (pipe.Endpoint, error) {
	var serviceCfg ServiceConfig
	err := json.Unmarshal(cfgData, &serviceCfg)
	if err != nil {
		return nil, err
	}

	serfConfig := &serfc.Config{
		Addr:    serviceCfg.Addr,
		AuthKey: serviceCfg.AuthKey,
	}

	return pipe.EndpointFunc(func(eventCh chan pipe.Event, closeCh chan struct{}) {
		var client *serfc.RPCClient
		_, fullEventCh := pipe.Bookkeeper(eventCh)
		defer func() {
			if client != nil {
				client.Close()
			}
			close(fullEventCh)
		}()

		for {
			client, err := serfc.ClientFromConfig(serfConfig)
			if err != nil {
				select {
				case <-closeCh:
					return
				case <-time.After(RECONNECT_INTERVAL):
				}
				continue // try to connect again
			}

			for {
				members, err := client.Members()
				if err != nil {
					log.Printf("Error while fetching serf members: %s\n", err)
					fullEventCh <- pipe.NewEvent() // Mark all nodes as down
					break
				}

				evt := pipe.NewEvent()
				for _, m := range members {
					if m.Status != "alive" || !matchTags(serviceCfg.Tags, m.Tags) {
						continue
					}
					log.Printf("Node: %s:%d is %s\n", m.Addr.String(), m.Port, m.Status)
					evt.AddNewNode(m.Name, pipe.NodeUp, m.Addr.String(), m.Port)
				}
				if len(evt) > 0 {
					fullEventCh <- evt
				}
				select {
				case <-closeCh:
					return
				case <-time.After(PULL_INTERVAL):
				}
			}
		}
	}), nil
}

func matchTags(required map[string]string, input map[string]string) bool {
	if len(required) > 0 && input == nil {
		return false
	}
	for key, value := range required {
		if val2, found := input[key]; !found {
			return false
		} else {
			if value == "*" || value == val2 {
				continue
			}
			return false
		}
	}
	return true
}
