package plugin

import (
	"encoding/json"
	"github.com/blang/receptor/pipe"
	"github.com/ugorji/go/codec"
	"log"
	"net"
	"net/rpc"
)

// RPCReactor defines a remotely executed reactor
type RPCReactor struct {
	socket string
	client *rpc.Client
}

func NewRPCReactor(socket string) (*RPCReactor, error) {
	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, err
	}
	var mh codec.MsgpackHandle
	rpcCodec := codec.GoRpc.ClientCodec(conn, &mh)
	client := rpc.NewClientWithCodec(rpcCodec)
	return &RPCReactor{
		socket: socket,
		client: client,
	}, nil
}

func (r *RPCReactor) Setup(cfg json.RawMessage) error {
	return r.client.Call("Reactor.Setup", &cfg, nil)
}

// Handle job and return a handler to start
func (r *RPCReactor) Accept(cfg json.RawMessage) (pipe.Endpoint, error) {
	var res int
	err := r.client.Call("Reactor.Accept", &cfg, &res)
	if err != nil {
		return nil, err
	}

	return &RPCReactorEndpoint{
		session: res,
		reactor: r,
	}, nil
}

type RPCReactorEndpoint struct {
	session int
	reactor *RPCReactor
}

func (e *RPCReactorEndpoint) Handle(eventCh chan pipe.Event, closeCh chan struct{}) {
	conn, err := net.Dial("unix", e.reactor.socket)
	if err != nil {
		log.Printf("Error while connecting: %s", err)
		return
	}
	var mh codec.MsgpackHandle
	en := codec.NewEncoder(conn, &mh)
	err = en.Encode(e.session)
	if err != nil {
		//TODO: handle error
		return
	}

	go func() {
		for {
			select {
			case ev, ok := <-eventCh:
				if !ok {
					conn.Close()
					return
				}
				err := en.Encode(&ev)
				if err != nil {
					conn.Close()
					return
				}
			case <-closeCh:
				conn.Close()
				return
			}
		}
	}()

	go func() {
		<-closeCh
		e.reactor.client.Call("Reactor.CloseHandle", &e.session, nil)
	}()

	err = e.reactor.client.Call("Reactor.Handle", &e.session, nil)
	if err != nil {
		conn.Close() // TODO: Close if err == nil?
		return
	}
}
