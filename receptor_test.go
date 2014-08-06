package receptor

import (
	"encoding/json"
	"github.com/blang/receptor/config"
	"github.com/blang/receptor/discovery"
	"github.com/blang/receptor/pipeline"
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

func (w *testWatcher) Accept(json.RawMessage) (pipeline.Endpoint, error) {
	w.acceptCalled = true
	return pipeline.EndpointFunc(func(eventCh chan pipeline.Event, closeCh chan struct{}) {
		for i := 0; i < 100; i++ {
			eventCh <- &pipeline.SingleNode{
				EName: "test" + strconv.Itoa(i),
				EType: pipeline.EventNodeUp,
			}
		}
		close(eventCh)
	}), nil
}

type testReactor struct {
	setupCalled    bool
	acceptCalled   bool
	receivedEvents []pipeline.Event
	eventRedirect  chan pipeline.Event
}

func (r *testReactor) Setup(json.RawMessage) error {
	r.setupCalled = true
	return nil
}
func (r *testReactor) Accept(json.RawMessage) (pipeline.Endpoint, error) {
	r.acceptCalled = true
	return pipeline.EndpointFunc(func(eventCh chan pipeline.Event, closeCh chan struct{}) {
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
		eventRedirect: make(chan pipeline.Event),
	}
	discovery.Watchers["testWatcher"] = watcher
	reactor.Reactors["testReactor"] = react

	//config
	serviceConfig := config.ServiceConfig{
		Watchers: make(map[string]config.ActorConfig),
		Reactors: make(map[string]config.ActorConfig),
	}
	serviceConfig.Watchers["testWatcher1"] = config.ActorConfig{
		Type:   "testWatcher",
		Config: nil,
	}
	serviceConfig.Reactors["testReactor1"] = config.ActorConfig{
		Type:   "testReactor",
		Config: nil,
	}

	cfg := &config.Config{
		Services: make(map[string]config.ServiceConfig),
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
			count += len(e.Nodes())
			t.Logf("%d Event received: %s\n", len(e.Nodes()), stringEventNodes(e.Nodes()))
			if count == 100 {
				receptor.Stop()
			}
		case <-timeout:
			t.Errorf("Timeout, events received: %d\n", count)
			return
		}
	}
}

func BenchmarkPipelineThroughput(b *testing.B) {
	notifiyWatcher := make(chan chan pipeline.Event)
	notifiyReactor := make(chan chan pipeline.Event)
	s := NewService()
	s.Watchers["watch"] = pipeline.NewManagedEndpoint(pipeline.EndpointFunc(func(eventCh chan pipeline.Event, closeCh chan struct{}) {
		notifiyWatcher <- eventCh
		<-closeCh
		close(eventCh)
	}))
	s.Reactors["react"] = pipeline.NewManagedEndpoint(pipeline.EndpointFunc(func(eventCh chan pipeline.Event, closeCh chan struct{}) {
		notifiyReactor <- eventCh
		<-closeCh
	}))
	s.Start()

	var inputCh chan pipeline.Event
	var outputCh chan pipeline.Event

	//Get input channel of pipeline
	select {
	case inputCh = <-notifiyWatcher:
	case <-time.After(2 * time.Second):
		b.Fatal("Timeout: Failed to get event channel")
	}

	//Get output channel of pipeline
	select {
	case outputCh = <-notifiyReactor:
	case <-time.After(2 * time.Second):
		b.Fatal("Timeout: Failed to get event channel")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		inputCh <- &pipeline.SingleNode{}
		<-outputCh
	}
}

func BenchmarkPipelineThroughputTwice(b *testing.B) {
	notifiyWatcher := make(chan chan pipeline.Event)
	notifiyReactor := make(chan chan pipeline.Event)
	s := NewService()
	s.Watchers["watch1"] = pipeline.NewManagedEndpoint(pipeline.EndpointFunc(func(eventCh chan pipeline.Event, closeCh chan struct{}) {
		notifiyWatcher <- eventCh
		<-closeCh
		close(eventCh)
	}))
	s.Watchers["watch2"] = pipeline.NewManagedEndpoint(pipeline.EndpointFunc(func(eventCh chan pipeline.Event, closeCh chan struct{}) {
		notifiyWatcher <- eventCh
		<-closeCh
		close(eventCh)
	}))
	s.Reactors["react1"] = pipeline.NewManagedEndpoint(pipeline.EndpointFunc(func(eventCh chan pipeline.Event, closeCh chan struct{}) {
		notifiyReactor <- eventCh
		<-closeCh
	}))
	s.Reactors["react2"] = pipeline.NewManagedEndpoint(pipeline.EndpointFunc(func(eventCh chan pipeline.Event, closeCh chan struct{}) {
		notifiyReactor <- eventCh
		<-closeCh
	}))
	s.Start()

	var inputCh1 chan pipeline.Event
	var inputCh2 chan pipeline.Event
	var outputCh1 chan pipeline.Event
	var outputCh2 chan pipeline.Event

	//Get input channel of pipeline
	select {
	case inputCh1 = <-notifiyWatcher:
	case <-time.After(2 * time.Second):
		b.Fatal("Timeout: Failed to get event channel")
	}
	select {
	case inputCh2 = <-notifiyWatcher:
	case <-time.After(2 * time.Second):
		b.Fatal("Timeout: Failed to get event channel")
	}

	//Get output channel of pipeline
	select {
	case outputCh1 = <-notifiyReactor:
	case <-time.After(2 * time.Second):
		b.Fatal("Timeout: Failed to get event channel")
	}
	select {
	case outputCh2 = <-notifiyReactor:
	case <-time.After(2 * time.Second):
		b.Fatal("Timeout: Failed to get event channel")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		inputCh1 <- &pipeline.SingleNode{}
		<-outputCh1
		<-outputCh2
		inputCh2 <- &pipeline.SingleNode{}
		<-outputCh1
		<-outputCh2
	}
}

func stringEventNodes(nodes []pipeline.NodeData) string {
	var str []string
	for _, n := range nodes {
		str = append(str, n.Name())
	}
	return strings.Join(str, ", ")
}
