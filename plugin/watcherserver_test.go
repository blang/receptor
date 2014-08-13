package plugin

import (
	"encoding/json"
	"errors"
	"github.com/blang/receptor/pipe"
	"net"
	"os"
	"reflect"
	"testing"
	"time"
)

type testWatcher struct {
	setupCfg       json.RawMessage
	setupError     string
	endpointCalled bool
}

func (w *testWatcher) Setup(cfg json.RawMessage) error {
	w.setupCfg = cfg
	if len(cfg) == 0 {
		w.setupError = "Wrong length"
		return errors.New(w.setupError)
	}
	return nil
}

func (w *testWatcher) Accept(cfg json.RawMessage) (pipe.Endpoint, error) {
	return pipe.EndpointFunc(func(eventCh chan pipe.Event, closeCh chan struct{}) {
		w.endpointCalled = true
		eventCh <- pipe.NewEventWithNode("localhost", pipe.NodeUp, "127.0.0.1", 8080)
		close(eventCh)
	}), nil
}

func TestWatcherServerSetup(t *testing.T) {
	socketPath, err := newSocket()
	if err != nil {
		t.Fatalf("Error creating socket: %s", err)
	}
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Error: %s", err)
	}
	defer os.Remove(socketPath)

	// Server side
	watcher := &testWatcher{}
	server := newWatcherServer(watcher, listener)

	go server.serve()

	// Calling side
	rpcWatcher, err := NewRPCWatcher(socketPath)
	if err != nil {
		t.Fatalf("Error creating RPC Watcher: %s", err)
	}

	// Test valid setup
	cfg := []byte("test")
	err = rpcWatcher.Setup(cfg)
	if err != nil {
		t.Fatalf("RPC setup failed: %s", err)
	}

	if reflect.DeepEqual(watcher.setupCfg, cfg) {
		t.Fatalf("Setup differs: expected %s, got %s", cfg, watcher.setupCfg)
	}

	// Test invalid setup
	wrongcfg := []byte{}
	err = rpcWatcher.Setup(wrongcfg)
	if err == nil {
		t.Fatal("Setup should return error")
	}
	if err.Error() != watcher.setupError {
		t.Errorf("Error message differs, expected %s, got %s", watcher.setupError, err.Error())
	}

}

func TestWatcherServerAccept(t *testing.T) {
	socketPath, err := newSocket()
	if err != nil {
		t.Fatalf("Error creating socket: %s", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Error: %s", err)
	}
	defer os.Remove(socketPath)

	// Server side
	watcher := &testWatcher{}
	server := newWatcherServer(watcher, listener)

	go server.serve()

	// Calling side
	rpcWatcher, err := NewRPCWatcher(socketPath)
	if err != nil {
		t.Fatalf("Error creating RPC Watcher: %s", err)
	}

	// Test valid setup
	cfg := []byte("test")
	endpoint, err := rpcWatcher.Accept(cfg)
	if err != nil {
		t.Fatalf("RPC accept failed: %s", err)
	}
	if endpoint == nil {
		t.Fatal("Endpoint nil")
	}
	eventCh := make(chan pipe.Event)
	closeCh := make(chan struct{})
	doneCh := make(chan struct{})

	go func() {
		endpoint.Handle(eventCh, closeCh)
		close(doneCh)
	}()
	e := <-eventCh
	if node, found := e["localhost"]; !found {
		t.Error("Node not found")
	} else if node.Host != "127.0.0.1" || node.Port != 8080 || node.Status != pipe.NodeUp {
		t.Errorf("Node incorrect: %s", node)
	}

	if !watcher.endpointCalled {
		t.Fatal("Endpoint not called")
	}

	if !isChannelClosed(eventCh) {
		t.Fatal("Event channel was not closed")
	}
	if !isGenericChannelClosed(doneCh) {
		t.Fatal("Handler did not return")
	}

}

type testWatcherClose struct {
	setupCfg       json.RawMessage
	setupError     string
	endpointCalled bool
}

func (w *testWatcherClose) Setup(cfg json.RawMessage) error {
	w.setupCfg = cfg
	if len(cfg) == 0 {
		w.setupError = "Wrong length"
		return errors.New(w.setupError)
	}
	return nil
}

func (w *testWatcherClose) Accept(cfg json.RawMessage) (pipe.Endpoint, error) {
	return pipe.EndpointFunc(func(eventCh chan pipe.Event, closeCh chan struct{}) {
		defer close(eventCh)
		w.endpointCalled = true
		for {
			select {
			case eventCh <- pipe.NewEventWithNode("localhost", pipe.NodeUp, "127.0.0.1", 8080):
			case <-closeCh:
				return
			}
			select {
			case <-time.After(time.Second):
			case <-closeCh:
				return
			}
		}
	}), nil
}

func TestWatcherServerCloseCh(t *testing.T) {
	socketPath, err := newSocket()
	if err != nil {
		t.Fatalf("Error creating socket: %s", err)
	}
	// fmt.Printf("Plugin socket: %s\n", socketPath) // Publish socketpath via stdout

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Error: %s", err)
	}
	defer os.Remove(socketPath)

	// Server side
	watcher := &testWatcherClose{}
	server := newWatcherServer(watcher, listener)

	go server.serve()

	// Calling side
	rpcWatcher, err := NewRPCWatcher(socketPath)
	if err != nil {
		t.Fatalf("Error creating RPC Watcher: %s", err)
	}

	// Test valid setup
	endpoint, err := rpcWatcher.Accept(nil)
	if err != nil {
		t.Fatalf("RPC accept failed: %s", err)
	}
	if endpoint == nil {
		t.Fatal("Endpoint nil")
	}
	eventCh := make(chan pipe.Event)
	closeCh := make(chan struct{})
	doneCh := make(chan struct{})

	go func() {
		endpoint.Handle(eventCh, closeCh)
		close(doneCh)
	}()

	e := <-eventCh
	if node, found := e["localhost"]; !found {
		t.Error("Node not found")
	} else if node.Host != "127.0.0.1" || node.Port != 8080 || node.Status != pipe.NodeUp {
		t.Errorf("Node incorrect: %s", node)
	}
	close(closeCh)
	if !watcher.endpointCalled {
		t.Fatal("Endpoint not called")
	}
	if !isGenericChannelClosed(doneCh) {
		t.Fatal("Handle does not return in time")
	}
	if !isChannelClosed(eventCh) {
		t.Fatal("EventCh was not closed")
	}

}

func TestWatcherServerMultipleHandlers(t *testing.T) {
	socketPath, err := newSocket()
	if err != nil {
		t.Fatalf("Error creating socket: %s", err)
	}
	// fmt.Printf("Plugin socket: %s\n", socketPath) // Publish socketpath via stdout

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Error: %s", err)
	}
	defer os.Remove(socketPath)

	// Server side
	watcher := &testWatcherClose{}
	server := newWatcherServer(watcher, listener)

	go server.serve()

	// Calling side
	rpcWatcher, err := NewRPCWatcher(socketPath)
	if err != nil {
		t.Fatalf("Error creating RPC Watcher: %s", err)
	}

	// Endpoint 1
	endpoint1, err := rpcWatcher.Accept(nil)
	if err != nil {
		t.Fatalf("RPC accept failed: %s", err)
	}
	if endpoint1 == nil {
		t.Fatal("Endpoint nil")
	}
	eventCh1 := make(chan pipe.Event)
	closeCh1 := make(chan struct{})
	doneCh1 := make(chan struct{})

	go func() {
		endpoint1.Handle(eventCh1, closeCh1)
		close(doneCh1)
	}()

	ev1 := <-eventCh1
	if node, found := ev1["localhost"]; !found {
		t.Error("Node not found")
	} else if node.Host != "127.0.0.1" || node.Port != 8080 || node.Status != pipe.NodeUp {
		t.Errorf("Node incorrect: %s", node)
	}

	// Endpoint 2
	endpoint2, err := rpcWatcher.Accept(nil)
	if err != nil {
		t.Fatalf("RPC accept failed: %s", err)
	}
	if endpoint1 == nil {
		t.Fatal("Endpoint nil")
	}
	eventCh2 := make(chan pipe.Event)
	closeCh2 := make(chan struct{})
	doneCh2 := make(chan struct{})

	go func() {
		endpoint2.Handle(eventCh2, closeCh2)
		close(doneCh2)
	}()

	ev2 := <-eventCh2
	if node, found := ev2["localhost"]; !found {
		t.Error("Node not found")
	} else if node.Host != "127.0.0.1" || node.Port != 8080 || node.Status != pipe.NodeUp {
		t.Errorf("Node incorrect: %s", node)
	}

	// Close closeChs
	close(closeCh1)
	close(closeCh2)

	if !isGenericChannelClosed(doneCh1) {
		t.Fatal("Handle1 does not return in time")
	}
	if !isChannelClosed(eventCh1) {
		t.Fatal("EventCh1 was not closed")
	}

	if !isGenericChannelClosed(doneCh2) {
		t.Fatal("Handle2 does not return in time")
	}
	if !isChannelClosed(eventCh2) {
		t.Fatal("EventCh2 was not closed")
	}

}

func isChannelClosed(ch chan pipe.Event) bool {
	timeout := time.After(5 * time.Second)
	for {
		select {
		case _, ok := <-ch:
			return !ok
		case <-timeout:
			return false
		default:
		}
	}
}

func isGenericChannelClosed(ch chan struct{}) bool {
	timeout := time.After(5 * time.Second)
	for {
		select {
		case _, ok := <-ch:
			return !ok
		case <-timeout:
			return false
		default:
		}
	}
}
