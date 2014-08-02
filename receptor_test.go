package receptor

import (
	"encoding/json"
	"github.com/blang/receptor/config"
	"github.com/blang/receptor/discovery"
	"github.com/blang/receptor/handler"
	"github.com/blang/receptor/reactor"
	"strconv"
	"strings"
	"testing"
	"time"
)

type testWatcher struct {
	setupCalled  bool
	acceptCalled bool
}

func (w *testWatcher) Setup(json.RawMessage) error {
	w.setupCalled = true
	return nil
}

func (w *testWatcher) Accept(json.RawMessage) (handler.Handler, error) {
	w.acceptCalled = true
	return handler.HandlerFunc(func(eventCh chan handler.Event, closeCh chan struct{}) {
		for i := 0; i < 100; i++ {
			eventCh <- &handler.SingleEvent{
				EName: "test" + strconv.Itoa(i),
				EType: handler.EventNodeUp,
			}
		}
	}), nil
}

type testReactor struct {
	setupCalled    bool
	acceptCalled   bool
	receivedEvents []handler.Event
	eventRedirect  chan handler.Event
}

func (r *testReactor) Setup(json.RawMessage) error {
	r.setupCalled = true
	return nil
}
func (r *testReactor) Accept(json.RawMessage) (handler.Handler, error) {
	r.acceptCalled = true
	return handler.HandlerFunc(func(eventCh chan handler.Event, closeCh chan struct{}) {
		for e := range eventCh {
			r.receivedEvents = append(r.receivedEvents, e)
			r.eventRedirect <- e
		}
		close(r.eventRedirect)
	}), nil
}

func TestSystem(t *testing.T) {
	watcher := &testWatcher{}
	react := &testReactor{
		eventRedirect: make(chan handler.Event),
	}
	discovery.Watchers["testWatcher"] = watcher
	reactor.Reactors["testReactor"] = react

	//config
	serviceConfig := config.ServiceConfig{
		Watchers: make(map[string]json.RawMessage),
		Reactors: make(map[string]json.RawMessage),
	}
	serviceConfig.Watchers["testWatcher"] = nil
	serviceConfig.Reactors["testReactor"] = nil

	cfg := config.Config{
		Services: make(map[string]config.ServiceConfig),
		Watchers: make(map[string]json.RawMessage),
		Reactors: make(map[string]json.RawMessage),
	}

	cfg.Watchers["testWatcher"] = nil
	cfg.Reactors["testReactor"] = nil
	cfg.Services["testService"] = serviceConfig
	receptor := &Receptor{}
	err := receptor.Init(cfg)
	if err != nil {
		t.Fatalf("Init returned error: %s", err)
	}
	receptor.Run()

	count := 0
	timeout := time.After(5 * time.Second)
	for {
		select {
		case e, ok := <-react.eventRedirect:
			if !ok {
				return
			}
			count += len(e.Nodes())
			t.Logf("%d Event received: %s\n", len(e.Nodes()), stringEventNodes(e.Nodes()))
			if count == 100 {
				receptor.Stop()
			}
		case <-timeout:
			t.Error("Timeout")
			return
			// t.Fail("Timeout")

		}

	}

}

func stringEventNodes(nodes []handler.NodeEvent) string {
	var str []string
	for _, n := range nodes {
		str = append(str, n.Name())
	}
	return strings.Join(str, ", ")
}
