package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"runtime/debug"
	"sync"

	"github.com/blang/receptor/pipe"
	"github.com/ugorji/go/codec"
)

// ServeWatcher exports the watcher as a plugin
func ServeWatcher(watcher pipe.Watcher) {
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
	server := newWatcherServer(watcher, listener)
	server.serve()
}

// watcherServer wraps a pipe.Watcher implementation and exposes it over golang rpc and event connections.
type watcherServer struct {
	watcher   pipe.Watcher
	listener  net.Listener
	epManager *endpointManager
}

func newWatcherServer(watcher pipe.Watcher, listener net.Listener) *watcherServer {
	return &watcherServer{
		watcher:   watcher,
		listener:  listener,
		epManager: newEndpointManager(),
	}
}

func (s *watcherServer) serve() {
	// First connection reserved for rpc
	conn, err := s.listener.Accept()
	if err != nil {
		log.Printf("Error accepting connection: %s\n", err)
		return
	}

	rpcServ := rpc.NewServer()
	rpcServ.RegisterName("Watcher", s)
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

func (s *watcherServer) Setup(cfg *json.RawMessage, _ *struct{}) error {
	if cfg == nil {
		return s.watcher.Setup(nil)
	}
	return s.watcher.Setup(*cfg)
}

func (s *watcherServer) Accept(cfg *json.RawMessage, res *int) error {
	var end pipe.Endpoint
	var err error
	if cfg == nil {
		end, err = s.watcher.Accept(nil)
	} else {
		end, err = s.watcher.Accept(*cfg)
	}
	if err != nil {
		return err
	}
	if end == nil {
		return errors.New("Endpoint was nil")
	}
	session := s.epManager.addEndpoint(end)
	*res = session
	return nil
}

func (s *watcherServer) Handle(sessionid *int, _ *struct{}) error {
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
	mergeIn := make(chan pipe.Event)
	pipe.Merger(mergeIn, eventCh)
	ep.Handle(mergeIn, closeCh) //TODO: if watcher ends, signal to eventchannel etc

	return nil
}

func (s *watcherServer) CloseHandle(sessionid *int, _ *struct{}) error {
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

func (s *watcherServer) serveEvents(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %s\n", err)
			return
		}
		go s.handleEventConnection(conn)
	}
}

func (s *watcherServer) handleEventConnection(conn net.Conn) {
	var mh codec.MsgpackHandle
	dec := codec.NewDecoder(conn, &mh)
	var session int
	err := dec.Decode(&session)
	if err != nil {
		log.Println("Could not read session id")
		conn.Close()
		return
	}

	_, eventCh, closeCh, err := s.epManager.getEndpoint(session) //TODO: What to do with closeCh?
	if err != nil {
		log.Printf("Endpoint manager error: %s", err)
		conn.Close()
		return
	}

	enc := codec.NewEncoder(conn, &mh)
	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				conn.Close()
				return
			}
			err := enc.Encode(event)
			if err != nil {
				log.Printf("Could not encode event, connection closed?: %s", err)
				close(closeCh)
				conn.Close()
				return
			}
		}
	}

}
