package restapi

import (
	"encoding/json"
	"fmt"
	"github.com/blang/receptor/pipe"
	"net/http"
)

type RestAPIServerWatcher struct {
	Listen string
	Server *http.Server
	Router *http.ServeMux
	//TODO: Add Mutex
	IsRunning bool
}

type Config struct {
	Listen string `json:"listen"`
}

type ServiceConfig struct {
	Service string `json:"service"`
}

func (w *RestAPIServerWatcher) Setup(cfgData json.RawMessage) error {
	conf := Config{
		Listen: "127.0.0.1:8080",
	}
	err := json.Unmarshal(cfgData, &conf)
	if err != nil {
		return err
	}
	w.Router = http.NewServeMux()
	w.Server = &http.Server{Addr: conf.Listen, Handler: w.Router}
	return nil
}

type RestEvent struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Host string `json:"host"`
	Port uint16 `json:"port"`
}

func (w *RestAPIServerWatcher) Accept(cfgData json.RawMessage) (pipe.Endpoint, error) {
	conf := ServiceConfig{
		Service: "default",
	}
	err := json.Unmarshal(cfgData, &conf)
	if err != nil {
		return nil, err
	}

	return pipe.EndpointFunc(func(eventCh chan pipe.Event, closeCh chan struct{}) {

		w.Router.HandleFunc("/service/"+conf.Service, func(w http.ResponseWriter, r *http.Request) {
			var restEvent RestEvent
			err := json.NewDecoder(r.Body).Decode(&restEvent)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "Bad request: %s", err)
				return
			}

			var nodeStatus pipe.NodeStatus
			switch restEvent.Type {
			case "nodeup":
				nodeStatus = pipe.NodeUp
			case "nodedown":
				nodeStatus = pipe.NodeDown
			default:
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintln(w, "Bad request: Invalid event type")
				return
			}

			eventCh <- pipe.NewEventWithNode(restEvent.Name, nodeStatus, restEvent.Host, restEvent.Port)

			fmt.Fprintln(w, "OK")
		})
		go func() {
			if !w.IsRunning {
				w.IsRunning = true
				go func() {
					w.Server.ListenAndServe() // Log error
				}()
			}
			<-closeCh
			close(eventCh)
		}()
	}), nil
}
