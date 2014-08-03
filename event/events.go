package event

import (
	"sort"
)

type EventType int

const (
	EventNodeUp   EventType = iota + 1 // Node is up now
	EventNodeDown                      // Node is down now
)

func (t EventType) String() string {
	switch t {
	case EventNodeUp:
		return "Node Up"
	case EventNodeDown:
		return "Node Down"
	default:
		return "Unknown EventType"
	}
}

type Event interface {
	Nodes() []NodeData
	Merge(older Event) Event
}

type NodeData interface {
	Name() string
	Type() EventType
	Host() string
	Port() int
}

// SingleEvent can be used as Event.
type SingleNode struct {
	EName string
	EType EventType
	EHost string
	EPort int
}

type MultiNode struct {
	Events []NodeData
}

func (e *MultiNode) Nodes() []NodeData {
	return e.Events
}

func (e *SingleNode) Name() string {
	return e.EName
}

func (e *SingleNode) Type() EventType {
	return e.EType
}

func (e *SingleNode) Host() string {
	return e.EHost
}

func (e *SingleNode) Port() int {
	return e.EPort
}

func (e *SingleNode) Nodes() []NodeData {
	return []NodeData{NodeData(e)}
}

type NodeDataByName []NodeData

func (n NodeDataByName) Len() int {
	return len(n)
}

func (n NodeDataByName) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}

func (n NodeDataByName) Less(i, j int) bool {
	return n[i].Name() < n[j].Name()
}

func (e *SingleNode) Merge(older Event) Event {
	merge := &MultiNode{}
	for _, oldNE := range older.Nodes() {
		if oldNE.Name() != e.Name() {
			merge.Events = append(merge.Events, oldNE)
		}
	}
	merge.Events = append(merge.Events, e)

	return merge
}

func (e *MultiNode) Merge(older Event) Event {
	merge := &MultiNode{}
	for _, newNE := range e.Nodes() {
		merge.Events = append(merge.Events, newNE)
	}
	sort.Sort(NodeDataByName(e.Events))

	for _, oldNE := range older.Nodes() {

		// Search NewNodes for oldNE, if nothing was found we need to add this to the new event
		if sort.Search(len(e.Events), func(i int) bool { return e.Events[i].Name() >= oldNE.Name() }) == -1 {
			merge.Events = append(merge.Events, oldNE)
		}
	}

	return merge
}
