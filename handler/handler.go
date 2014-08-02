package handler

import (
	"sort"
)

type Handler interface {
	// Started in seperate go routine
	Handle(eventCh chan Event, closeCh chan struct{})
}

type HandlerFunc func(eventCh chan Event, closeCh chan struct{})

func (f HandlerFunc) Handle(eventCh chan Event, closeCh chan struct{}) {
	f(eventCh, closeCh)
}

type EventType int

const (
	EventNodeUp       EventType = iota + 1 // Node is up now
	EventNodeDown                          // Node is down now
	EventNodeTempDown                      // Node is temporarily down (e.g. Used to mark node as disabled )
)

type Event interface {
	Nodes() []NodeEvent
	Merge(older Event) Event
}

type NodeEvent interface {
	Name() string
	Type() EventType
	Host() string
	Port() int
}

// SingleEvent can be used as Event.
type SingleEvent struct {
	EName string
	EType EventType
	EHost string
	EPort int
}

type MultiEvent struct {
	Events []NodeEvent
}

func (e *MultiEvent) Nodes() []NodeEvent {
	return e.Events
}

func (e *SingleEvent) Name() string {
	return e.EName
}

func (e *SingleEvent) Type() EventType {
	return e.EType
}

func (e *SingleEvent) Host() string {
	return e.EHost
}

func (e *SingleEvent) Port() int {
	return e.EPort
}

func (e *SingleEvent) Nodes() []NodeEvent {
	return []NodeEvent{NodeEvent(e)}
}

type NodeEventByName []NodeEvent

func (n NodeEventByName) Len() int {
	return len(n)
}

func (n NodeEventByName) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}

func (n NodeEventByName) Less(i, j int) bool {
	return n[i].Name() < n[j].Name()
}

func (e *SingleEvent) Merge(older Event) Event {
	merge := &MultiEvent{}
	for _, oldNE := range older.Nodes() {
		if oldNE.Name() != e.Name() {
			merge.Events = append(merge.Events, oldNE)
		}
	}
	merge.Events = append(merge.Events, e)

	return merge
}

func (e *MultiEvent) Merge(older Event) Event {
	merge := &MultiEvent{}
	for _, newNE := range e.Nodes() {
		merge.Events = append(merge.Events, newNE)
	}
	sort.Sort(NodeEventByName(e.Events))

	for _, oldNE := range older.Nodes() {

		// Search NewNodes for oldNE, if nothing was found we need to add this to the new event
		if sort.Search(len(e.Events), func(i int) bool { return e.Events[i].Name() >= oldNE.Name() }) == -1 {
			merge.Events = append(merge.Events, oldNE)
		}
	}

	return merge
}
