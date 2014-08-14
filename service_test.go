package receptor

import (
	"github.com/blang/receptor/pipe"
	"strings"
	"testing"
	"time"
)

func BenchmarkPipelineThroughput(b *testing.B) {
	notifiyWatcher := make(chan chan pipe.Event)
	notifiyReactor := make(chan chan pipe.Event)
	s := NewService("testservice")
	s.AddWatcherEndpoint("watch1", pipe.NewManagedEndpoint(pipe.EndpointFunc(func(eventCh chan pipe.Event, closeCh chan struct{}) {
		notifiyWatcher <- eventCh
		<-closeCh
		close(eventCh)
	})))
	s.AddReactorEndpoint("react1", pipe.NewManagedEndpoint(pipe.EndpointFunc(func(eventCh chan pipe.Event, closeCh chan struct{}) {
		notifiyReactor <- eventCh
		<-closeCh
	})))
	s.Start()

	var inputCh chan pipe.Event
	var outputCh chan pipe.Event

	//Get input channel of pipe
	select {
	case inputCh = <-notifiyWatcher:
	case <-time.After(2 * time.Second):
		b.Fatal("Timeout: Failed to get event channel")
	}

	//Get output channel of pipe
	select {
	case outputCh = <-notifiyReactor:
	case <-time.After(2 * time.Second):
		b.Fatal("Timeout: Failed to get event channel")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		inputCh <- &pipe.SingleNode{}
		<-outputCh
	}
}

func BenchmarkPipelineThroughputTwice(b *testing.B) {
	notifiyWatcher := make(chan chan pipe.Event)
	notifiyReactor := make(chan chan pipe.Event)
	s := NewService("testservice")
	s.AddWatcherEndpoint("watch1", pipe.NewManagedEndpoint(pipe.EndpointFunc(func(eventCh chan pipe.Event, closeCh chan struct{}) {
		notifiyWatcher <- eventCh
		<-closeCh
		close(eventCh)
	})))
	s.AddWatcherEndpoint("watch2", pipe.NewManagedEndpoint(pipe.EndpointFunc(func(eventCh chan pipe.Event, closeCh chan struct{}) {
		notifiyWatcher <- eventCh
		<-closeCh
		close(eventCh)
	})))
	s.AddReactorEndpoint("react1", pipe.NewManagedEndpoint(pipe.EndpointFunc(func(eventCh chan pipe.Event, closeCh chan struct{}) {
		notifiyReactor <- eventCh
		<-closeCh
	})))
	s.AddReactorEndpoint("react2", pipe.NewManagedEndpoint(pipe.EndpointFunc(func(eventCh chan pipe.Event, closeCh chan struct{}) {
		notifiyReactor <- eventCh
		<-closeCh
	})))
	s.Start()

	var inputCh1 chan pipe.Event
	var inputCh2 chan pipe.Event
	var outputCh1 chan pipe.Event
	var outputCh2 chan pipe.Event

	//Get input channel of pipe
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

	//Get output channel of pipe
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
		inputCh1 <- &pipe.SingleNode{}
		<-outputCh1
		<-outputCh2
		inputCh2 <- &pipe.SingleNode{}
		<-outputCh1
		<-outputCh2
	}
}

func stringEventNodes(nodes []pipe.NodeData) string {
	var str []string
	for _, n := range nodes {
		str = append(str, n.Name())
	}
	return strings.Join(str, ", ")
}
