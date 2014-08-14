package pipe

import (
	"sync"
)

// Book is a datastructure needed for bookkeeping of node data.
// It receives incremental and full updates on the list of running nodes by events
// and tracks a list of all nodes currently up.
// Therefore it's capable of filtering redundant events
// and only returns necessary incremental updates.
// Is thread-safe.
type Book struct {
	mutex sync.RWMutex
	m     map[string]NodeInfo
}

// Creates a new empty book
func NewBook() *Book {
	return &Book{
		m: make(map[string]NodeInfo),
	}
}

// UpdateInc updates the book with an incremental event ev
// and returns an event if the update changed the book.
// If the update contains a node with status EventNodeDown but
// was never seen by the book, the node is ignored.
// Incremental updates only work if book receives events from start up
// and never miss an event.
func (b *Book) UpdateInc(ev Event) Event {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	outEv := NewEvent()
	for _, node := range ev {
		if bookNode, found := b.m[node.Name]; found {
			if bookNode.Status != node.Status ||
				bookNode.Host != node.Host ||
				bookNode.Port != node.Port {

				// Add/update if up, delete if down
				if node.Status == NodeUp {
					b.m[bookNode.Name] = node
				} else {
					delete(b.m, bookNode.Name)
				}
				outEv.AddNode(node)
			}
		} else { // Not in book
			if node.Status == NodeDown {
				continue // Down node never seen before, don't care
			}
			b.m[node.Name] = node
			outEv.AddNode(node)
		}
	}
	if len(outEv) > 0 {
		return outEv
	}
	return nil
}

// UpdateFull updates the book with the list of nodes inside the event.
// The event represents a full update, so only EventNodeUp is allowed.
// Missing nodes are marked as EventNodeDown and removed from book.
func (b *Book) UpdateFull(ev Event) Event {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	outEv := NewEvent()
	marked := make(map[string]struct{})
	for _, node := range ev {
		if node.Status == NodeDown {
			continue // EventNodeDown could not be used on full update
		}
		if bookNode, found := b.m[node.Name]; found {
			if bookNode.Status != node.Status ||
				bookNode.Host != node.Host ||
				bookNode.Port != node.Port {

				b.m[bookNode.Name] = node
				marked[bookNode.Name] = struct{}{}
				outEv.AddNode(node)
			} else {
				marked[bookNode.Name] = struct{}{}
			}
		} else { // Not found in book
			marked[node.Name] = struct{}{}
			b.m[node.Name] = node
			outEv.AddNode(node)
		}
	}
	for name, node := range b.m {
		if _, found := marked[name]; !found {
			outEv.AddNode(NewNodeInfo(node.Name, NodeDown, node.Host, node.Port))
			delete(b.m, name)
		}
	}
	if len(outEv) > 0 {
		return outEv
	}
	return nil
}

// Full returns a event containing all nodes currently up.
// If there are no nodes an event without nodes is returned.
func (b *Book) Full() Event {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	ev := NewEvent()
	for _, node := range b.m {
		ev.AddNode(node)
	}
	return ev
}

// Bookkeeper accepts full and incremental updates on his returned channels
// and sends redundant-free incremental events out on the incOutCh channel.
// It enables watchers which only get a full list of backends to send them without further bookkeeping.
// Full updates on fullInCh have to be a single Event with multiple nodes, only accepting EventNodeUp types.
func Bookkeeper(incOutCh chan Event) (chan Event, chan Event) {
	book := NewBook()
	incInCh := make(chan Event)
	fullInCh := make(chan Event)
	go func() {
		for {
			select {
			case incEv, ok := <-incInCh:
				if !ok {
					close(incOutCh)
					return
				}
				outEv := book.UpdateInc(incEv)
				if outEv != nil {
					incOutCh <- outEv
				}

			case fullEv, ok := <-fullInCh:
				if !ok {
					close(incOutCh)
					return
				}
				outEv := book.UpdateFull(fullEv)
				if outEv != nil {
					incOutCh <- outEv
				}
			}
		}
	}()
	return incInCh, fullInCh
}

// BookkeeperReceiver reads an incremental event channel incInCh
// and returns a channel with full updates send on request.
// If incInCh gets closed, the output channel is closed.
func BookkeeperReceiver(incInCh chan Event) chan Event {
	book := NewBook()
	fullOutCh := make(chan Event)
	go func() {
		for {
			select {
			case incEv, ok := <-incInCh:
				if !ok {
					close(fullOutCh)
					return
				}
				book.UpdateInc(incEv)
			case fullOutCh <- book.Full():
			}
		}

	}()
	return fullOutCh
}
