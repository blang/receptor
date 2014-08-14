package pipe

import (
	"fmt"
	"strings"
)

type NodeStatus int

const (
	NodeUp NodeStatus = iota + 1 // Node is up
	NodeDown
)

func (s NodeStatus) String() string {
	switch s {
	case NodeUp:
		return "NodeUp"
	case NodeDown:
		return "NodeDown"
	default:
		return "Unknown"
	}
}

type NodeInfo struct {
	Name   string
	Status NodeStatus
	Host   string
	Port   uint16
}

func NewNodeInfo(name string, status NodeStatus, host string, port uint16) NodeInfo {
	return NodeInfo{
		Name:   name,
		Status: status,
		Host:   host,
		Port:   port,
	}
}

func (n NodeInfo) String() string {
	return fmt.Sprintf("Name: %s, Status: %s, Host: %s, Port: %d", n.Name, n.Status, n.Host, n.Port)
}

type Event map[string]NodeInfo

func NewEvent() Event {
	return Event(make(map[string]NodeInfo))
}

func NewEventWithNode(name string, status NodeStatus, host string, port uint16) Event {
	event := NewEvent()
	event[name] = NewNodeInfo(name, status, host, port)
	return event
}

func (e Event) Update(newer Event) {
	for key, val := range newer {
		e[key] = val
	}
}

func (e Event) AddNode(node NodeInfo) {
	e[node.Name] = node
}

func (e Event) AddNewNode(name string, status NodeStatus, host string, port uint16) {
	e[name] = NewNodeInfo(name, status, host, port)
}

func (e Event) String() string {
	var parts []string
	for _, node := range e {
		parts = append(parts, node.String())
	}
	return "Event Nodes: " + strings.Join(parts, ", ")
}
