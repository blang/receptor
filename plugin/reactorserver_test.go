package plugin

import (
	"encoding/json"
	"errors"
	"github.com/blang/receptor/pipe"
	"net"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"
)

type testReactor struct {
	setupCfg  json.RawMessage
	acceptCfg json.RawMessage
}

func (r *testReactor) Setup(cfg json.RawMessage) error {
	r.setupCfg = cfg
	if len(cfg) == 0 {
		return errors.New("Wrong length")
	}
	return nil
}

func (r *testReactor) Accept(cfg json.RawMessage) (pipe.Endpoint, error) {
	r.acceptCfg = cfg
	if len(cfg) == 0 {
		return nil, errors.New("Wrong length")
	}
	return pipe.EndpointFunc(func(eventCh chan pipe.Event, closeCh chan struct{}) {
	}), nil
}

func TestReactorServerSetup(t *testing.T) {
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
	reactor := &testReactor{}
	server := newReactorServer(reactor, listener)

	go server.serve()

	// Client side
	rpcReactor, err := NewRPCReactor(socketPath)
	if err != nil {
		t.Fatalf("Error creating RPC Reactor: %s", err)
	}

	// Test valid setup
	cfg := []byte("test")
	err = rpcReactor.Setup(cfg)
	if err != nil {
		t.Fatalf("RPC setup failed: %s", err)
	}

	if reflect.DeepEqual(reactor.setupCfg, cfg) {
		t.Fatalf("Setup differs: expected %s, got %s", cfg, reactor.setupCfg)
	}

	// Test invalid setup
	wrongcfg := []byte{}
	err = rpcReactor.Setup(wrongcfg)
	if err == nil {
		t.Fatal("Setup should return error")
	}
}

func TestReactorServerAccept(t *testing.T) {
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
	reactor := &testReactor{}
	server := newReactorServer(reactor, listener)

	go server.serve()

	// Client side
	rpcReactor, err := NewRPCReactor(socketPath)
	if err != nil {
		t.Fatalf("Error creating RPC Reactor: %s", err)
	}

	// Test valid setup
	cfg := []byte("test")
	handler, err := rpcReactor.Accept(cfg)
	if err != nil || handler == nil {
		t.Fatalf("RPC setup failed: %s", err)
	}

	if reflect.DeepEqual(reactor.acceptCfg, cfg) {
		t.Fatalf("Setup differs: expected %s, got %s", cfg, reactor.setupCfg)
	}

	// Test invalid setup
	wrongcfg := []byte{}
	_, err = rpcReactor.Accept(wrongcfg)
	if err == nil {
		t.Fatal("Setup should return error")
	}
}

type testReactorFull struct {
	redirectCh chan pipe.Event
	once       sync.Once
}

func (r *testReactorFull) Setup(_ json.RawMessage) error {
	return nil
}

func (r *testReactorFull) Accept(_ json.RawMessage) (pipe.Endpoint, error) {
	return pipe.EndpointFunc(func(eventCh chan pipe.Event, closeCh chan struct{}) {
		for {
			select {
			case ev, ok := <-eventCh:
				if !ok {
					r.once.Do(func() { close(r.redirectCh) })
					return
				}
				r.redirectCh <- ev
			case <-closeCh:
				r.once.Do(func() { close(r.redirectCh) })
				return
			}
		}
	}), nil
}

func TestReactorServerClosedEvCh(t *testing.T) {
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
	redirectCh := make(chan pipe.Event)
	reactor := &testReactorFull{redirectCh: redirectCh}
	server := newReactorServer(reactor, listener)

	go server.serve()

	// Client side
	rpcReactor, err := NewRPCReactor(socketPath)
	if err != nil {
		t.Fatalf("Error creating RPC Reactor: %s", err)
	}

	handler, _ := rpcReactor.Accept(nil)

	eventCh := make(chan pipe.Event)
	closeCh := make(chan struct{})
	doneCh := make(chan struct{})

	go func() {
		handler.Handle(eventCh, closeCh)
		close(doneCh)
	}()

	eventCh <- pipe.NewEventWithNode("localhost", pipe.NodeUp, "127.0.0.1", 8080)

	select {
	case ev, ok := <-redirectCh:
		if !ok {
			t.Fatal("Redirect channel closed")
		}
		if _, found := ev["localhost"]; !found {
			t.Fatal("Wrong event received")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout on receive event")
	}

	// Close eventCh
	close(eventCh)

	if !isGenericChannelClosed(doneCh) {
		t.Fatal("Handler did not return")
	}
}

func TestReactorServerCloseCh(t *testing.T) {
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
	redirectCh := make(chan pipe.Event)
	reactor := &testReactorFull{redirectCh: redirectCh}
	server := newReactorServer(reactor, listener)

	go server.serve()

	// Client side
	rpcReactor, err := NewRPCReactor(socketPath)
	if err != nil {
		t.Fatalf("Error creating RPC Reactor: %s", err)
	}

	handler, _ := rpcReactor.Accept(nil)

	eventCh := make(chan pipe.Event)
	closeCh := make(chan struct{})
	doneCh := make(chan struct{})

	go func() {
		handler.Handle(eventCh, closeCh)
		close(doneCh)
	}()

	eventCh <- pipe.NewEventWithNode("localhost", pipe.NodeUp, "127.0.0.1", 8080)

	select {
	case ev, ok := <-redirectCh:
		if !ok {
			t.Fatal("Redirect channel closed")
		}
		if _, found := ev["localhost"]; !found {
			t.Fatal("Wrong event received")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout on receive event")
	}

	// Close closeCh
	close(closeCh)

	if !isGenericChannelClosed(doneCh) {
		t.Fatal("Handler did not return")
	}
}

func TestReactorServerMultiple(t *testing.T) {
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
	redirectCh := make(chan pipe.Event)
	reactor := &testReactorFull{redirectCh: redirectCh}
	server := newReactorServer(reactor, listener)

	go server.serve()

	// Client side
	rpcReactor, err := NewRPCReactor(socketPath)
	if err != nil {
		t.Fatalf("Error creating RPC Reactor: %s", err)
	}

	handler1, _ := rpcReactor.Accept(nil)

	eventCh1 := make(chan pipe.Event)
	closeCh1 := make(chan struct{})
	doneCh1 := make(chan struct{})

	go func() {
		handler1.Handle(eventCh1, closeCh1)
		close(doneCh1)
	}()

	handler2, _ := rpcReactor.Accept(nil)

	eventCh2 := make(chan pipe.Event)
	closeCh2 := make(chan struct{})
	doneCh2 := make(chan struct{})

	go func() {
		handler2.Handle(eventCh2, closeCh2)
		close(doneCh2)
	}()

	eventCh1 <- pipe.NewEventWithNode("localhost", pipe.NodeUp, "127.0.0.1", 8080)
	eventCh2 <- pipe.NewEventWithNode("localhost", pipe.NodeUp, "127.0.0.1", 8080)

	// Receive 2 events
	select {
	case ev, ok := <-redirectCh:
		if !ok {
			t.Fatal("Redirect channel closed")
		}
		if _, found := ev["localhost"]; !found {
			t.Fatal("Wrong event received")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout on receive event")
	}

	select {
	case ev, ok := <-redirectCh:
		if !ok {
			t.Fatal("Redirect channel closed")
		}
		if _, found := ev["localhost"]; !found {
			t.Fatal("Wrong event received")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout on receive event")
	}

	// Close eventChs
	close(eventCh1)
	close(eventCh2)

	if !isGenericChannelClosed(doneCh1) {
		t.Fatal("Handler did not return")
	}

	if !isGenericChannelClosed(doneCh2) {
		t.Fatal("Handler did not return")
	}
}

func TestReactorServerMultipleEvents(t *testing.T) {
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
	redirectCh := make(chan pipe.Event)
	reactor := &testReactorFull{redirectCh: redirectCh}
	server := newReactorServer(reactor, listener)

	go server.serve()

	// Client side
	rpcReactor, err := NewRPCReactor(socketPath)
	if err != nil {
		t.Fatalf("Error creating RPC Reactor: %s", err)
	}

	handler, _ := rpcReactor.Accept(nil)

	eventCh := make(chan pipe.Event)
	closeCh := make(chan struct{})
	doneCh := make(chan struct{})

	go func() {
		handler.Handle(eventCh, closeCh)
		close(doneCh)
	}()

	for i := 0; i < 1000; i += 1 {
		eventCh <- pipe.NewEventWithNode("localhost", pipe.NodeUp, "127.0.0.1", 8080)

		select {
		case ev, ok := <-redirectCh:
			if !ok {
				t.Fatalf("Redirect channel closed on %d", i)
			}
			if _, found := ev["localhost"]; !found {
				t.Fatal("Wrong event received")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout on receive event")
		}
	}

	// Close closeCh
	close(closeCh)

	if !isGenericChannelClosed(doneCh) {
		t.Fatal("Handler did not return")
	}
}
