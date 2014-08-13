package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/blang/receptor/pipe"
	"github.com/ugorji/go/codec"
	"log"
	"net"
	"net/rpc"
	"os"
	"runtime/debug"
	"sync"
)

func ServeReactor(reactor pipe.Reactor) {
	setLogFlags()
	lnet, laddr, err := extrSocketInfo(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "This is a receptor plugin, don't call directly!")
		os.Exit(1)
	}

	fmt.Printf("Plugin socket: %s://%s\n", lnet, laddr) // Publish socketpath via stdout

	listener, err := net.Listen(lnet, laddr)
	if err != nil {
		log.Printf("Error: %s", err)
		os.Exit(1)
	}

	server := newReactorServer(reactor, listener)
	server.serve()
}

// reactorServer wraps a pipe.Reactor implementation and exposes it over golang rpc and event connections.
type reactorServer struct {
	reactor   pipe.Reactor
	listener  net.Listener
	epManager *endpointManager
}

func newReactorServer(reactor pipe.Reactor, listener net.Listener) *reactorServer {
	return &reactorServer{
		reactor:   reactor,
		listener:  listener,
		epManager: newEndpointManager(),
	}
}

func (s *reactorServer) serve() {
	// First connection reserved for rpc
	conn, err := s.listener.Accept()
	if err != nil {
		log.Printf("Error accepting connection: %s\n", err)
		return
	}

	rpcServ := rpc.NewServer()
	rpcServ.RegisterName("Reactor", s)
	var mh codec.MsgpackHandle
	rpcCodec := codec.GoRpc.ServerCodec(conn, &mh)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		rpcServ.ServeCodec(rpcCodec)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		s.serveEvents(s.listener)
		wg.Done()
	}()

	wg.Wait()
}

// RPC Method: Reactor.Setup(..)
func (s *reactorServer) Setup(cfg *json.RawMessage, _ *struct{}) error {
	return s.reactor.Setup(*cfg)
}

// RPC Method: Reactor.Accept(..)
func (s *reactorServer) Accept(cfg *json.RawMessage, res *int) error {
	end, err := s.reactor.Accept(*cfg)
	if err != nil {
		return err
	}
	session := s.epManager.addEndpoint(end)
	*res = session
	return nil
}

// RPC Method: Reactor.Handle(sessionid)
func (s *reactorServer) Handle(sessionid *int, _ *struct{}) error {
	// Panic recovery
	defer func() {
		if err := recover(); err != nil {
			trace := debug.Stack()
			log.Printf("PANIC: %s, stack: %s\n", err, trace)
		}
	}()
	if sessionid == nil {
		return errors.New("Invalid sessionid")
	}
	ep, eventCh, closeCh, err := s.epManager.getEndpoint(*sessionid)
	if err != nil {
		return err
	}
	mergeOut := make(chan pipe.Event)
	pipe.Merger(eventCh, mergeOut)
	ep.Handle(mergeOut, closeCh) //TODO: if watcher ends, signal to eventchannel etc
	return nil
}

// RPC Method: Reactor.CloseHandle(sessionid)
func (s *reactorServer) CloseHandle(sessionid *int, _ *struct{}) error {
	if sessionid == nil {
		return errors.New("Invalid sessionid")
	}
	_, _, closeCh, err := s.epManager.getEndpoint(*sessionid)
	if err != nil {
		return err
	}
	close(closeCh)
	return nil
}

func (s *reactorServer) serveEvents(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %s\n", err)
			return
		}
		go s.handleEventConnection(conn)
	}
}

func (s *reactorServer) handleEventConnection(conn net.Conn) {
	var mh codec.MsgpackHandle
	dec := codec.NewDecoder(conn, &mh)
	var session int
	err := dec.Decode(&session)
	if err != nil {
		log.Println("Could not read session id")
		conn.Close()
		return
	}
	_, eventCh, _, err := s.epManager.getEndpoint(session) //TODO: What to do with closeCh?
	if err != nil {
		log.Printf("Endpoint manager error: %s", err)
		conn.Close()
		return
	}

	for {
		var ev pipe.Event
		err := dec.Decode(&ev)
		if err != nil {
			log.Printf("Decode failed: %s", err)
			conn.Close()
			close(eventCh)
			return
		}
		eventCh <- ev // Merger will accept this immediately
	}

}
