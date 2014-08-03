package event

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
					curEvent = e.Merge(curEvent)
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
