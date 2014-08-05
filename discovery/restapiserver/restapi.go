package restapiserver

import (
	"encoding/json"
	"fmt"
	"github.com/blang/receptor/discovery"
	"github.com/blang/receptor/pipeline"
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

func init() {
	discovery.Watchers["restapiserver"] = &RestAPIServerWatcher{}
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
	Port int    `json:"port"`
}

func (w *RestAPIServerWatcher) Accept(cfgData json.RawMessage) (pipeline.Endpoint, error) {
	conf := ServiceConfig{
		Service: "default",
	}
	err := json.Unmarshal(cfgData, &conf)
	if err != nil {
		return nil, err
	}

	return pipeline.EndpointFunc(func(eventCh chan pipeline.Event, closeCh chan struct{}) {

		w.Router.HandleFunc("/service/"+conf.Service, func(w http.ResponseWriter, r *http.Request) {
			var restEvent RestEvent
			err := json.NewDecoder(r.Body).Decode(&restEvent)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "Bad request: %s", err)
				return
			}

			var eventType pipeline.EventType
			switch restEvent.Type {
			case "nodeup":
				eventType = pipeline.EventNodeUp
			case "nodedown":
				eventType = pipeline.EventNodeDown
			default:
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintln(w, "Bad request: Invalid event type")
				return
			}

			eventCh <- &pipeline.SingleNode{
				EName: restEvent.Name,
				EHost: restEvent.Host,
				EPort: restEvent.Port,
				EType: eventType,
			}

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
