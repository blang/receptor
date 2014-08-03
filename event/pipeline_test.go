package event

import (
	"sort"
	"testing"
	"time"
)

// Test if Merger passes events through one by one if no congestion
func TestMergerPassThrough(t *testing.T) {
	inCh := make(chan Event, 1)
	outCh := Merger(inCh)
	inCh <- Event(&SingleNode{
		EName: "Test1",
	})
	e1 := <-outCh
	if e1 == nil {
		t.Fatal("First received event was nil")
	}
	if nodename := e1.Nodes()[0].Name(); nodename != "Test1" {
		t.Fatalf("First received event has wrong node name: %s", nodename)
	}

	inCh <- Event(&SingleNode{
		EName: "Test2",
	})
	e2 := <-outCh
	if e2 == nil {
		t.Fatal("First received event was nil")
	}
	if nodename := e2.Nodes()[0].Name(); nodename != "Test2" {
		t.Fatalf("First received event has wrong node name: %s", nodename)
	}
	close(inCh)
}

func TestMergerCongestionMerge(t *testing.T) {
	inCh := make(chan Event)
	outCh := Merger(inCh)
	inCh <- Event(&SingleNode{
		EName: "Test1",
	})
	inCh <- Event(&SingleNode{
		EName: "Test2",
		EType: EventNodeUp,
	})
	inCh <- Event(&SingleNode{
		EName: "Test2",
		EType: EventNodeDown, // Newest Event: Node is not up anymore
	})
	e1 := <-outCh
	if e1 == nil {
		t.Fatal("First received event was nil")
	}
	if nodeCount := len(e1.Nodes()); nodeCount != 2 {
		t.Fatalf("Expected first event to hold 2 nodes, instead received %d nodes", nodeCount)
	}

	nodeEvents := e1.Nodes()
	sort.Sort(NodeDataByName(nodeEvents))

	if nodename := e1.Nodes()[0].Name(); nodename != "Test1" {
		t.Fatalf("Expected Node %s, found instead %s", "Test1", nodename)
	}

	if nodename := e1.Nodes()[1].Name(); nodename != "Test2" {
		t.Fatalf("Expected Node %s, found instead %s", "Test2", nodename)
	}

	if eventType := e1.Nodes()[1].Type(); eventType != EventNodeDown {
		t.Fatalf("Not received last EventType, instead: %s", eventType)
	}

	inCh <- Event(&SingleNode{
		EName: "Test3",
	})
	e2 := <-outCh
	if e2 == nil {
		t.Fatal("First received event was nil")
	}
	if nodeCount := len(e2.Nodes()); nodeCount != 1 {
		t.Fatalf("Expected last event to has a single node, but had %d nodes", nodeCount)
	}
	if nodename := e2.Nodes()[0].Name(); nodename != "Test3" {
		t.Fatalf("vent has wrong node name: %s", nodename)
	}
	close(inCh)

	// If Input is closed, pipeline is shut down, pending events are ignored
	if !isChannelClosed(outCh) {
		t.Fatal("Output channel not closed")
	}
}

func TestBroadcaster(t *testing.T) {
	inCh := make(chan Event)
	outCh1 := make(chan Event)
	outCh2 := make(chan Event)
	event := Event(&SingleNode{
		EName: "Test1",
	})
	Broadcaster(inCh, []chan Event{outCh1, outCh2})
	inCh <- event
	e1 := <-outCh1
	e2 := <-outCh2

	if e1 != event || e1 != e2 {
		t.Fatal("Not received 2 identical events on both channels")
	}

	close(inCh)

	if !isChannelClosed(outCh1) || !isChannelClosed(outCh2) {
		t.Fatal("Output channels not closed")
	}

}

func isChannelClosed(ch chan Event) bool {
	timeout := time.After(5 * time.Second)
	for {
		select {
		case _, ok := <-ch:
			return !ok
		case <-timeout:
			return false
		default:
		}
	}
	return false
}
