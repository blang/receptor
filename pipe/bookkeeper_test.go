package pipe

import (
	"errors"
	"testing"
	"time"
)

func TestBookkeeperCloseInc(t *testing.T) {
	eventCh := make(chan Event)

	inIncCh, _ := Bookkeeper(eventCh)
	close(inIncCh)
	if !isChannelClosed(eventCh) {
		t.Fatal("Channel was not closed")
	}
}

func TestBookkeeperCloseFull(t *testing.T) {
	eventCh := make(chan Event)

	_, inFullCh := Bookkeeper(eventCh)
	close(inFullCh)
	if !isChannelClosed(eventCh) {
		t.Fatal("Channel was not closed")
	}
}

func TestBookkeeperInc(t *testing.T) {
	eventCh := make(chan Event)
	inIncCh, _ := Bookkeeper(eventCh)
	inIncCh <- NewEventWithNode("Node1", NodeUp, "127.0.0.1", 80)
	ev, err := receiveEventTimeout(eventCh, 3*time.Second)
	if err != nil {
		t.Fatalf("Could not receive event: %s, event: %s", err, ev)
	}
	if ev == nil {
		t.Fatal("Expected event")
	}
	if len(ev) != 1 {
		t.Fatal("Invalid amount of nodes")
	}

	// Send same update again
	inIncCh <- NewEventWithNode("Node1", NodeUp, "127.0.0.1", 80)
	_, err = receiveEventOrError(eventCh)
	if err == nil {
		t.Fatal("Expected no event")
	}
}

func TestBookkeeperFull(t *testing.T) {
	eventCh := make(chan Event)
	inIncCh, inFullCh := Bookkeeper(eventCh)

	fullList1 := NewEvent()
	fullList1.AddNewNode("Node1", NodeUp, "127.0.0.1", 80)
	fullList1.AddNewNode("Node2", NodeUp, "127.0.0.2", 81)
	fullList1.AddNewNode("Node3", NodeUp, "127.0.0.3", 83)

	inFullCh <- fullList1
	ev, err := receiveEventTimeout(eventCh, 2*time.Second)
	if err != nil {
		t.Fatalf("Could not receive event: %s", err)
	}
	if countNodes := len(ev); countNodes != 3 {
		t.Fatalf("Expected 3 nodes, got %d", countNodes)
	}

	inIncCh <- NewEventWithNode("Node1", NodeDown, "127.0.0.1", 80)
	ev, err = receiveEventTimeout(eventCh, 2*time.Second)
	if err != nil {
		t.Fatalf("Could not receive event: %s", err)
	}
	if countNodes := len(ev); countNodes != 1 {
		t.Fatalf("Expected 1 nodes, got %d", countNodes)
	}
	if node, found := ev["Node1"]; !found {
		t.Fatal("Event should have info about Node1")
	} else if node.Status != NodeDown {
		t.Fatal("Node should be down")
	}

	// Send full list again
	fullList2 := NewEvent()
	fullList2.AddNewNode("Node1", NodeUp, "127.0.0.1", 80)
	fullList2.AddNewNode("Node2", NodeUp, "127.0.0.2", 81)
	fullList2.AddNewNode("Node3", NodeUp, "127.0.0.3", 83)
	inFullCh <- fullList2

	ev, err = receiveEventTimeout(eventCh, 2*time.Second)
	if err != nil {
		t.Fatalf("Could not receive event: %s", err)
	}
	if countNodes := len(ev); countNodes != 1 {
		t.Fatalf("Expected 1 nodes, got %d", countNodes)
	}
	if node, found := ev["Node1"]; !found {
		t.Fatal("Node not found in event")
	} else if node.Status != NodeUp {
		t.Fatalf("Wrong event received: %s", node)
	}
}

func receiveEventTimeout(eventCh chan Event, timeout time.Duration) (Event, error) {
	select {
	case ev, ok := <-eventCh:
		if !ok {
			return nil, errors.New("Channel closed")
		}
		return ev, nil
	case <-time.After(timeout):
		return nil, errors.New("Timeout")
	}
}

func receiveEventOrError(eventCh chan Event) (Event, error) {
	select {
	case ev, ok := <-eventCh:
		if !ok {
			return nil, errors.New("Channel closed")
		}
		return ev, nil
	default:
		return nil, errors.New("No Event received")
	}
}

func TestBookUpdateInc(t *testing.T) {
	b := NewBook()

	ev := b.UpdateInc(NewEventWithNode("Node1", NodeUp, "127.0.0.1", 80))
	if ev == nil {
		t.Error("No event was generated")
	}
	if len(ev) != 1 {
		t.Fatal("Event has wrong number of nodes")
	}
	if _, found := ev["Node1"]; !found {
		t.Fatal("Node not found")
	}

	// Bring same node up again
	ev = b.UpdateInc(NewEventWithNode("Node1", NodeUp, "127.0.0.1", 80))
	if ev != nil {
		t.Fatalf("Did not expect update event, got %s", ev)
	}

	// Same node down
	ev = b.UpdateInc(NewEventWithNode("Node1", NodeDown, "127.0.0.1", 80))
	if ev == nil {
		t.Fatal("Did expect update event, got none")
	}

	// Bring down unkown host
	ev = b.UpdateInc(NewEventWithNode("NodeUnkown", NodeDown, "127.0.0.1", 80))
	if ev != nil {
		t.Fatalf("Did not expect update event, got %s", ev)
	}
}

func TestBookUpdateFull(t *testing.T) {
	b := NewBook()

	fullList1 := NewEvent()
	fullList1.AddNewNode("Node1", NodeUp, "127.0.0.1", 80)
	fullList1.AddNewNode("Node2", NodeUp, "127.0.0.2", 81)
	fullList1.AddNewNode("Node3", NodeUp, "127.0.0.3", 83)

	ev := b.UpdateFull(fullList1)
	if ev == nil {
		t.Fatal("No event received")
	}
	if countNodes := len(ev); countNodes != 3 {
		t.Fatalf("Expected 3 nodes in event, got %d", countNodes)
	}

	// Node1 unchanged, Node2 changed port, Node4 new
	fullList2 := NewEvent()
	fullList2.AddNewNode("Node1", NodeUp, "127.0.0.1", 80)
	fullList2.AddNewNode("Node2", NodeUp, "127.0.0.2", 87)
	fullList2.AddNewNode("Node4", NodeUp, "127.0.0.4", 84)

	ev = b.UpdateFull(fullList2)
	if ev == nil {
		t.Fatal("No event received")
	}

	if countNodes := len(ev); countNodes != 3 {
		t.Fatalf("Expected 2 nodes in event, got %d", countNodes)
	}

	// Node1
	if _, found := ev["Node1"]; found {
		t.Error("Did not expect node1")
	}

	// Node2
	if node, found := ev["Node2"]; found {
		if node.Port != 87 {
			t.Errorf("Expected port of node2 to be updated, got: %d", node.Port)
		}
	} else {
		t.Error("Expected node2 in event")
	}

	// Node 3
	if node, found := ev["Node3"]; found {
		if node.Status != NodeDown {
			t.Error("Expected node3 to be down now")
		}
	} else {
		t.Error("Expected node3 in event")
	}

	// Node 4
	if node, found := ev["Node4"]; found {
		if node.Status != NodeUp || node.Host != "127.0.0.4" || node.Port != 84 {
			t.Errorf("Node4 data wrong: %s", node)
		}
	} else {
		t.Error("Expected node4 in event")
	}

	// Send same update again
	fullList3 := NewEvent()
	fullList3.AddNewNode("Node1", NodeUp, "127.0.0.1", 80)
	fullList3.AddNewNode("Node2", NodeUp, "127.0.0.2", 87)
	fullList3.AddNewNode("Node4", NodeUp, "127.0.0.4", 84)

	ev = b.UpdateFull(fullList3)
	if ev != nil {
		t.Fatal("Expected no event, nothing changed")
	}
}

func TestBookkeeperReceiver(t *testing.T) {
	eventCh := make(chan Event)

	fullCh := BookkeeperReceiver(eventCh)
	eventCh <- NewEventWithNode("Node1", NodeUp, "127.0.0.1", 80)

	fullSet := <-fullCh
	if fullSet == nil {
		t.Fatal("Nil event received")
	}
	if nodeCount := len(fullSet); nodeCount != 1 {
		t.Fatalf("Expected one node, received: %d", nodeCount)
	}
	if _, found := fullSet["Node1"]; !found {
		t.Fatal("Expected Node1")
	}

	eventCh <- NewEventWithNode("Node2", NodeUp, "127.0.0.1", 81)

	fullSet = <-fullCh
	if fullSet == nil {
		t.Fatal("Nil event received")
	}
	if nodeCount := len(fullSet); nodeCount != 2 {
		t.Fatalf("Expected two nodes, received: %d", nodeCount)
	}

	if _, found := fullSet["Node1"]; !found {
		t.Fatal("Node1 not found")
	}
	if _, found := fullSet["Node2"]; !found {
		t.Fatal("Node2 not found")
	}

	// Node2 goes down
	eventCh <- NewEventWithNode("Node1", NodeDown, "127.0.0.1", 80)

	fullSet = <-fullCh
	if fullSet == nil {
		t.Fatal("Nil event received")
	}
	if nodeCount := len(fullSet); nodeCount != 1 {
		t.Fatalf("Expected one nodes, received: %d", nodeCount)
	}

	if _, found := fullSet["Node2"]; !found {
		t.Fatal("Expected Node2")
	}

	close(eventCh)
	if !isChannelClosed(fullCh) {
		t.Fatal("Expect out channel to be closed")
	}
}
