package plugin

import (
	"encoding/json"
	"github.com/blang/receptor/pipe"
	"github.com/ugorji/go/codec"
	"log"
	"net"
	"net/rpc"
)

// RPCWatcher defines a remotely executed watcher
type RPCWatcher struct {
	socket string
	client *rpc.Client
}

func NewRPCWatcher(socket string) (*RPCWatcher, error) {
	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, err
	}
	var mh codec.MsgpackHandle
	rpcCodec := codec.GoRpc.ClientCodec(conn, &mh)
	client := rpc.NewClientWithCodec(rpcCodec)
	return &RPCWatcher{
		socket: socket,
		client: client,
	}, nil
}

func (w *RPCWatcher) Setup(cfg json.RawMessage) error {
	return w.client.Call("Watcher.Setup", &cfg, nil)
}

// Handle job and return a handler to start watching
func (w *RPCWatcher) Accept(cfg json.RawMessage) (pipe.Endpoint, error) {
	var res int
	err := w.client.Call("Watcher.Accept", &cfg, &res)
	if err != nil {
		return nil, err
	}

	return &RPCWatcherEndpoint{
		session: res,
		watcher: w,
	}, nil
}

type RPCWatcherEndpoint struct {
	session int
	watcher *RPCWatcher
}

func (e *RPCWatcherEndpoint) Handle(eventCh chan pipe.Event, closeCh chan struct{}) {
	conn, err := net.Dial("unix", e.watcher.socket)
	if err != nil {
		log.Printf("Error while connecting: %s", err)
		return
	}
	var mh codec.MsgpackHandle
	enc := codec.NewEncoder(conn, &mh)
	err = enc.Encode(e.session)
	if err != nil {
		//TODO: handle error
		return
	}

	go func() {
		var mh codec.MsgpackHandle
		dec := codec.NewDecoder(conn, &mh)
		for {
			var ev pipe.Event
			err := dec.Decode(&ev)
			if err != nil {
				close(eventCh)
				return
			}
			select {
			case eventCh <- ev:
			case <-closeCh:
				return

			}

		}
	}()

	go func() {
		<-closeCh
		e.watcher.client.Call("Watcher.CloseHandle", &e.session, nil)
	}()

	err = e.watcher.client.Call("Watcher.Handle", &e.session, nil)
	if err != nil {
		conn.Close() // TODO: Close if err == nil?
		return
	}
}
