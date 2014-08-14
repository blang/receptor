package pipe

import (
	"testing"
	"time"
)

// Test if Merger passes events through one by one if no congestion
func TestMergerPassThrough(t *testing.T) {
	inCh := make(chan Event, 1)
	outCh := Merger(inCh)
	inCh <- NewEventWithNode("Test1", NodeUp, "127.0.0.1", 80)

	e1 := <-outCh
	if e1 == nil {
		t.Fatal("First received event was nil")
	}
	if _, found := e1["Test1"]; !found {
		t.Fatal("Expected Test1")
	}

	inCh <- NewEventWithNode("Test2", NodeUp, "127.0.0.1", 81)
	e2 := <-outCh
	if e2 == nil {
		t.Fatal("First received event was nil")
	}
	if _, found := e2["Test2"]; !found {
		t.Fatal("Expected Test2")
	}
	close(inCh)
}

func TestMergerCongestionMerge(t *testing.T) {
	inCh := make(chan Event)
	outCh := Merger(inCh)
	inCh <- NewEventWithNode("Test1", NodeUp, "127.0.0.1", 80)
	inCh <- NewEventWithNode("Test2", NodeUp, "127.0.0.1", 81)
	inCh <- NewEventWithNode("Test2", NodeDown, "127.0.0.1", 81) // Newest Event: Node is not up anymore

	e1 := <-outCh
	if e1 == nil {
		t.Fatal("Event was nil")
	}
	if nodeCount := len(e1); nodeCount != 2 {
		t.Fatalf("Expected event to hold 2 nodes, instead received %d nodes", nodeCount)
	}

	if _, found := e1["Test1"]; !found {
		t.Fatal("Expected Node Test1")
	}

	if node, found := e1["Test2"]; !found {
		t.Fatal("Expected Test2")
	} else if node.Status != NodeDown {
		t.Fatal("Expected Test2 to be down")
	}

	inCh <- NewEventWithNode("Test3", NodeUp, "127.0.0.3", 83)
	e2 := <-outCh
	if e2 == nil {
		t.Fatal("First received event was nil")
	}
	if nodeCount := len(e2); nodeCount != 1 {
		t.Fatalf("Expected last event to has a single node, but had %d nodes", nodeCount)
	}
	if _, found := e2["Test3"]; !found {
		t.Fatal("Expected Test3")
	}
	close(inCh)

	// If Input is closed, pipe is shut down, pending events are ignored
	if !isChannelClosed(outCh) {
		t.Fatal("Output channel not closed")
	}
}

func TestBroadcaster(t *testing.T) {
	inCh := make(chan Event)
	outCh1 := make(chan Event)
	outCh2 := make(chan Event)
	event := NewEventWithNode("Test1", NodeUp, "127.0.0.1", 80)
	Broadcaster(inCh, []chan Event{outCh1, outCh2})
	inCh <- event
	e1 := <-outCh1
	e2 := <-outCh2

	if len(e1) != 1 || len(e2) != 1 {
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

func TestForwarder(t *testing.T) {
	outCh := make(chan Event)
	inCh1 := make(chan Event)
	inCh2 := make(chan Event)
	f := NewForwarder(outCh)

	f.Forward(inCh1)
	f.Forward(inCh2)
	f.WaitClose()

	inCh1 <- NewEventWithNode("Test1", NodeUp, "127.0.0.1", 80)
	inCh2 <- NewEventWithNode("Test2", NodeUp, "127.0.0.2", 81)

	select {
	case _, ok := <-outCh:
		if !ok {
			t.Fatal("Channel closed unexpected")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout: Could not receive on output channel")
	}

	select {
	case _, ok := <-outCh:
		if !ok {
			t.Fatal("Channel closed unexpected")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout: Could not receive on output channel")
	}

	close(inCh1)

	// Test if pipe works after one forwarder turns down
	inCh2 <- NewEventWithNode("Test3", NodeUp, "127.0.0.3", 83)

	select {
	case _, ok := <-outCh:
		if !ok {
			t.Fatal("Channel closed unexpected")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout: Could not receive on output channel")
	}

	close(inCh2)
	// Test if outCh is closed if all input channels are closed
	if !isChannelClosed(outCh) {
		t.Errorf("Channel should be closed")
	}
}
