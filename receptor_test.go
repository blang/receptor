package receptor

import (
	"encoding/json"
	"github.com/blang/receptor/pipe"
	"strconv"
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

func (w *testWatcher) Accept(json.RawMessage) (pipe.Endpoint, error) {
	w.acceptCalled = true
	return pipe.EndpointFunc(func(eventCh chan pipe.Event, closeCh chan struct{}) {
		for i := 0; i < 100; i++ {
			eventCh <- pipe.NewEventWithNode("test"+strconv.Itoa(i), pipe.NodeUp, "127.0.0."+strconv.Itoa(i), 80)
		}
		close(eventCh)
	}), nil
}

type testReactor struct {
	setupCalled    bool
	acceptCalled   bool
	receivedEvents []pipe.Event
	eventRedirect  chan pipe.Event
}

func (r *testReactor) Setup(json.RawMessage) error {
	r.setupCalled = true
	return nil
}
func (r *testReactor) Accept(json.RawMessage) (pipe.Endpoint, error) {
	r.acceptCalled = true
	return pipe.EndpointFunc(func(eventCh chan pipe.Event, closeCh chan struct{}) {
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
		eventRedirect: make(chan pipe.Event),
	}
	Watchers["testWatcher"] = watcher
	Reactors["testReactor"] = react

	//config
	serviceConfig := ServiceConfig{
		Watchers: make(map[string]ActorConfig),
		Reactors: make(map[string]ActorConfig),
	}
	serviceConfig.Watchers["testWatcher1"] = ActorConfig{
		Type:   "testWatcher",
		Config: nil,
	}
	serviceConfig.Reactors["testReactor1"] = ActorConfig{
		Type:   "testReactor",
		Config: nil,
	}

	cfg := &Config{
		Services: make(map[string]ServiceConfig),
		Watchers: make(map[string]json.RawMessage),
		Reactors: make(map[string]json.RawMessage),
	}

	cfg.Watchers["testWatcher"] = nil
	cfg.Reactors["testReactor"] = nil
	cfg.Services["testService"] = serviceConfig
	receptor := NewReceptor()
	err := receptor.Init(cfg)
	if err != nil {
		t.Fatalf("Init returned error: %s", err)
	}
	receptor.Start()

	count := 0
	timeout := time.After(5 * time.Second)
	for {
		select {
		case e, ok := <-react.eventRedirect:
			if !ok {
				return
			}
			count += len(e)
			t.Logf("%d Event received: %s\n", len(e), e)
			if count == 100 {
				receptor.Stop()
			}
		case <-timeout:
			t.Errorf("Timeout, events received: %d\n", count)
			return
		}
	}
}
