package receptor

import (
	"github.com/blang/receptor/pipeline"
	"strings"
	"testing"
	"time"
)

func BenchmarkPipelineThroughput(b *testing.B) {
	notifiyWatcher := make(chan chan pipeline.Event)
	notifiyReactor := make(chan chan pipeline.Event)
	s := NewService("testservice")
	s.AddWatcherEndpoint("watch1", pipeline.NewManagedEndpoint(pipeline.EndpointFunc(func(eventCh chan pipeline.Event, closeCh chan struct{}) {
		notifiyWatcher <- eventCh
		<-closeCh
		close(eventCh)
	})))
	s.AddReactorEndpoint("react1", pipeline.NewManagedEndpoint(pipeline.EndpointFunc(func(eventCh chan pipeline.Event, closeCh chan struct{}) {
		notifiyReactor <- eventCh
		<-closeCh
	})))
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
	s := NewService("testservice")
	s.AddWatcherEndpoint("watch1", pipeline.NewManagedEndpoint(pipeline.EndpointFunc(func(eventCh chan pipeline.Event, closeCh chan struct{}) {
		notifiyWatcher <- eventCh
		<-closeCh
		close(eventCh)
	})))
	s.AddWatcherEndpoint("watch2", pipeline.NewManagedEndpoint(pipeline.EndpointFunc(func(eventCh chan pipeline.Event, closeCh chan struct{}) {
		notifiyWatcher <- eventCh
		<-closeCh
		close(eventCh)
	})))
	s.AddReactorEndpoint("react1", pipeline.NewManagedEndpoint(pipeline.EndpointFunc(func(eventCh chan pipeline.Event, closeCh chan struct{}) {
		notifiyReactor <- eventCh
		<-closeCh
	})))
	s.AddReactorEndpoint("react2", pipeline.NewManagedEndpoint(pipeline.EndpointFunc(func(eventCh chan pipeline.Event, closeCh chan struct{}) {
		notifiyReactor <- eventCh
		<-closeCh
	})))
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
