package pipe

import (
	"sync"
)

// EventMerger merges incoming events from inCh if sink could not keep up
func Merger(inCh chan Event) (eventCh chan Event) {
	outCh := make(chan Event)
	go func(inCh chan Event, outCh chan Event) {
		var tmpCh chan Event
		var curEvent Event
		for {
			select {
			case e, ok := <-inCh:
				if !ok {
					close(outCh)
					return // Ignore pending events
				}
				// Received multiple events, sink could not keep up, merge events to one up-to-date event
				if curEvent != nil {
					curEvent.Update(e)
				} else {
					curEvent = e
				}
				tmpCh = outCh

			case tmpCh <- curEvent: // Sent current event, close send channel for now and reset currentEvent
				curEvent = nil
				tmpCh = nil
			}
		}
	}(inCh, outCh)
	return outCh
}

func Broadcaster(inCh chan Event, outChs []chan Event) {
	go func() {
		for event := range inCh {
			for _, outch := range outChs {
				outch <- event
			}
		}
		for _, outCh := range outChs {
			close(outCh)
		}
	}()
}

type Forwarder struct {
	wg    sync.WaitGroup
	outCh chan Event
}

// NewForwarder creates a new forwarder for given output channel.
func NewForwarder(outCh chan Event) *Forwarder {
	return &Forwarder{
		outCh: outCh,
	}
}

// Forwards the given inChannel to output channel.
func (f *Forwarder) Forward(inCh chan Event) {
	f.wg.Add(1)
	go func() {
		for event := range inCh {
			f.outCh <- event
		}
		f.wg.Done()
	}()
}

// WaitClose waits until all forwarded input channels are closed and closes output channel.
// Does not block.
func (f *Forwarder) WaitClose() {
	go func() {
		f.wg.Wait()
		close(f.outCh)
	}()
}
