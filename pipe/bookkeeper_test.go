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
	inIncCh <- &SingleNode{
		EName: "Node1",
		EType: EventNodeUp,
		EHost: "127.0.0.1",
		EPort: 80,
	}
	ev, err := receiveEventTimeout(eventCh, 2*time.Second)
	if err != nil {
		t.Fatalf("Could not receive event: %s", err)
	}
	if ev == nil {
		t.Fatal("Expected event")
	}
	if len(ev.Nodes()) != 1 {
		t.Fatal("Invalid amount of nodes")
	}

	// Send same update again
	inIncCh <- &SingleNode{
		EName: "Node1",
		EType: EventNodeUp,
		EHost: "127.0.0.1",
		EPort: 80,
	}
	_, err = receiveEventOrError(eventCh)
	if err == nil {
		t.Fatal("Expected no event")
	}
}

func TestBookkeeperFull(t *testing.T) {
	eventCh := make(chan Event)
	inIncCh, inFullCh := Bookkeeper(eventCh)

	fullList1 := &MultiNode{}
	fullList1.Events = append(fullList1.Events, &SingleNode{
		EName: "Node1",
		EType: EventNodeUp,
		EHost: "127.0.0.1",
		EPort: 80,
	})
	fullList1.Events = append(fullList1.Events, &SingleNode{
		EName: "Node2",
		EType: EventNodeUp,
		EHost: "127.0.0.2",
		EPort: 81,
	})
	fullList1.Events = append(fullList1.Events, &SingleNode{
		EName: "Node3",
		EType: EventNodeUp,
		EHost: "127.0.0.3",
		EPort: 83,
	})

	inFullCh <- fullList1
	ev, err := receiveEventTimeout(eventCh, 2*time.Second)
	if err != nil {
		t.Fatalf("Could not receive event: %s", err)
	}
	if countNodes := len(ev.Nodes()); countNodes != 3 {
		t.Fatalf("Expected 3 nodes, got %d", countNodes)
	}

	inIncCh <- &SingleNode{
		EName: "Node1",
		EType: EventNodeDown,
		EHost: "127.0.0.1",
		EPort: 80,
	}
	ev, err = receiveEventTimeout(eventCh, 2*time.Second)
	if err != nil {
		t.Fatalf("Could not receive event: %s", err)
	}
	if countNodes := len(ev.Nodes()); countNodes != 1 {
		t.Fatalf("Expected 1 nodes, got %d", countNodes)
	}
	if node := ev.Nodes()[0]; node.Name() != "Node1" || node.Type() != EventNodeDown {
		t.Fatalf("Wrong event received: %s", node)
	}

	// Send full list again
	fullList2 := &MultiNode{}
	fullList2.Events = append(fullList2.Events, &SingleNode{
		EName: "Node1",
		EType: EventNodeUp,
		EHost: "127.0.0.1",
		EPort: 80,
	})
	fullList2.Events = append(fullList2.Events, &SingleNode{
		EName: "Node2",
		EType: EventNodeUp,
		EHost: "127.0.0.2",
		EPort: 81,
	})
	fullList2.Events = append(fullList2.Events, &SingleNode{
		EName: "Node3",
		EType: EventNodeUp,
		EHost: "127.0.0.3",
		EPort: 83,
	})
	inFullCh <- fullList2
	ev, err = receiveEventTimeout(eventCh, 2*time.Second)
	if err != nil {
		t.Fatalf("Could not receive event: %s", err)
	}
	if countNodes := len(ev.Nodes()); countNodes != 1 {
		t.Fatalf("Expected 1 nodes, got %d", countNodes)
	}
	if node := ev.Nodes()[0]; node.Name() != "Node1" || node.Type() != EventNodeUp {
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

	ev := b.UpdateInc(&SingleNode{
		EName: "Node1",
		EType: EventNodeUp,
		EHost: "127.0.0.1",
		EPort: 80,
	})
	if ev == nil {
		t.Error("No event was generated")
	}
	if len(ev.Nodes()) != 1 {
		t.Fatal("Event has wrong number of nodes")
	}
	if ev.Nodes()[0].Name() != "Node1" {
		t.Fatal("Wrong node")
	}

	// Bring same node up again
	ev = b.UpdateInc(&SingleNode{
		EName: "Node1",
		EType: EventNodeUp,
		EHost: "127.0.0.1",
		EPort: 80,
	})
	if ev != nil {
		t.Fatalf("Did not expect update event, got %s", ev)
	}

	// Same node down
	ev = b.UpdateInc(&SingleNode{
		EName: "Node1",
		EType: EventNodeDown,
		EHost: "127.0.0.1",
		EPort: 80,
	})
	if ev == nil {
		t.Fatal("Did expect update event, got none")
	}

	// Bring down unkown host
	ev = b.UpdateInc(&SingleNode{
		EName: "NodeUnkown",
		EType: EventNodeDown,
		EHost: "127.0.0.1",
		EPort: 80,
	})
	if ev != nil {
		t.Fatalf("Did not expect update event, got %s", ev)
	}
}

func TestBookUpdateFull(t *testing.T) {
	b := NewBook()

	fullList1 := &MultiNode{}
	fullList1.Events = append(fullList1.Events, &SingleNode{
		EName: "Node1",
		EType: EventNodeUp,
		EHost: "127.0.0.1",
		EPort: 80,
	})
	fullList1.Events = append(fullList1.Events, &SingleNode{
		EName: "Node2",
		EType: EventNodeUp,
		EHost: "127.0.0.2",
		EPort: 81,
	})
	fullList1.Events = append(fullList1.Events, &SingleNode{
		EName: "Node3",
		EType: EventNodeUp,
		EHost: "127.0.0.3",
		EPort: 83,
	})

	ev := b.UpdateFull(fullList1)
	if ev == nil {
		t.Fatal("No event received")
	}
	if countNodes := len(ev.Nodes()); countNodes != 3 {
		t.Fatalf("Expected 3 nodes in event, got %d", countNodes)
	}

	// Node1 unchanged, Node2 changed port, Node4 new
	fullList2 := &MultiNode{}
	fullList2.Events = append(fullList2.Events, &SingleNode{
		EName: "Node1",
		EType: EventNodeUp,
		EHost: "127.0.0.1",
		EPort: 80,
	})
	fullList2.Events = append(fullList2.Events, &SingleNode{
		EName: "Node2",
		EType: EventNodeUp,
		EHost: "127.0.0.2",
		EPort: 87, // Port changed
	})
	fullList2.Events = append(fullList2.Events, &SingleNode{
		EName: "Node4",
		EType: EventNodeUp,
		EHost: "127.0.0.4",
		EPort: 84,
	})

	ev = b.UpdateFull(fullList2)
	if ev == nil {
		t.Fatal("No event received")
	}

	if countNodes := len(ev.Nodes()); countNodes != 3 {
		t.Fatalf("Expected 2 nodes in event, got %d", countNodes)
	}

	tmpM := nodesArrToMap(ev.Nodes())

	// Node1
	if _, found := tmpM["Node1"]; found {
		t.Error("Did not expect node1")
	}

	// Node2
	if node, found := tmpM["Node2"]; found {
		if node.Port() != 87 {
			t.Errorf("Expected port of node2 to be updated, got: %d", node.Port())
		}
	} else {
		t.Error("Expected node2 in event")
	}

	// Node 3
	if node, found := tmpM["Node3"]; found {
		if node.Type() != EventNodeDown {
			t.Error("Expected node3 to be down now")
		}
	} else {
		t.Error("Expected node3 in event")
	}

	// Node 4
	if node, found := tmpM["Node4"]; found {
		if node.Type() != EventNodeUp || node.Host() != "127.0.0.4" || node.Port() != 84 {
			t.Errorf("Node4 data wrong: %s", node)
		}
	} else {
		t.Error("Expected node4 in event")
	}

	// Send same update again
	fullList3 := &MultiNode{}
	fullList3.Events = append(fullList3.Events, &SingleNode{
		EName: "Node1",
		EType: EventNodeUp,
		EHost: "127.0.0.1",
		EPort: 80,
	})
	fullList3.Events = append(fullList3.Events, &SingleNode{
		EName: "Node2",
		EType: EventNodeUp,
		EHost: "127.0.0.2",
		EPort: 87, // Port changed
	})
	fullList3.Events = append(fullList3.Events, &SingleNode{
		EName: "Node4",
		EType: EventNodeUp,
		EHost: "127.0.0.4",
		EPort: 84,
	})

	ev = b.UpdateFull(fullList3)
	if ev != nil {
		t.Fatal("Expected no event, nothing changed")
	}
}

func nodesArrToMap(nodeData []NodeData) map[string]NodeData {
	tmpM := make(map[string]NodeData)
	for _, node := range nodeData {
		tmpM[node.Name()] = node
	}
	return tmpM
}

func TestBookFull(t *testing.T) {
	b := NewBook()
	fullList := &MultiNode{}
	fullList.Events = append(fullList.Events, &SingleNode{
		EName: "Node1",
		EType: EventNodeUp,
		EHost: "127.0.0.1",
		EPort: 80,
	})
	fullList.Events = append(fullList.Events, &SingleNode{
		EName: "Node2",
		EType: EventNodeUp,
		EHost: "127.0.0.2",
		EPort: 87, // Port changed
	})
	fullList.Events = append(fullList.Events, &SingleNode{
		EName: "Node4",
		EType: EventNodeUp,
		EHost: "127.0.0.4",
		EPort: 84,
	})

	b.UpdateFull(fullList)

	ev := b.Full()
	if ev == nil {
		t.Fatal("Event was nil")
	}

	if nodeCount := len(ev.Nodes()); nodeCount != 3 {
		t.Fatalf("Expected 3 nodes, got %d", nodeCount)
	}
}

func TestBookkeeperReceiver(t *testing.T) {
	eventCh := make(chan Event)

	fullCh := BookkeeperReceiver(eventCh)
	eventCh <- &SingleNode{
		EName: "Node1",
		EType: EventNodeUp,
		EHost: "127.0.0.1",
		EPort: 80,
	}

	fullSet := <-fullCh
	if fullSet == nil {
		t.Fatal("Nil event received")
	}
	if nodeCount := len(fullSet.Nodes()); nodeCount != 1 {
		t.Fatalf("Expected one node, received: %d", nodeCount)
	}
	if node := fullSet.Nodes()[0]; node.Name() != "Node1" {
		t.Fatalf("Expected Node1, received %s", node.Name())
	}

	eventCh <- &SingleNode{
		EName: "Node2",
		EType: EventNodeUp,
		EHost: "127.0.0.1",
		EPort: 81,
	}

	fullSet = <-fullCh
	if fullSet == nil {
		t.Fatal("Nil event received")
	}
	if nodeCount := len(fullSet.Nodes()); nodeCount != 2 {
		t.Fatalf("Expected two nodes, received: %d", nodeCount)
	}

	m := nodesArrToMap(fullSet.Nodes())

	if _, found := m["Node1"]; !found {
		t.Fatal("Node1 not found")
	}
	if _, found := m["Node2"]; !found {
		t.Fatal("Node2 not found")
	}

	// Node2 goes down
	eventCh <- &SingleNode{
		EName: "Node1",
		EType: EventNodeDown,
		EHost: "127.0.0.1",
		EPort: 80,
	}

	fullSet = <-fullCh
	if fullSet == nil {
		t.Fatal("Nil event received")
	}
	if nodeCount := len(fullSet.Nodes()); nodeCount != 1 {
		t.Fatalf("Expected one nodes, received: %d", nodeCount)
	}

	if node := fullSet.Nodes()[0]; node.Name() != "Node2" {
		t.Fatalf("Expected Node2, received %s", node.Name())
	}

	close(eventCh)
	if !isChannelClosed(fullCh) {
		t.Fatal("Expect out channel to be closed")
	}
}
