package plugin

import (
	"errors"
	"github.com/blang/receptor/pipe"
	"sync"
)

// endpointManager manages endpoints by creating the needed channels and link them to a sessionid
type endpointManager struct {
	endpoints []endpointWrapper
	lock      sync.RWMutex
}

func newEndpointManager() *endpointManager {
	return &endpointManager{
		lock: sync.RWMutex{},
	}
}

func (m *endpointManager) addEndpoint(ep pipe.Endpoint) int {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.endpoints = append(m.endpoints, endpointWrapper{ //TODO: Generate sessionid
		endpoint: ep,
		eventCh:  make(chan pipe.Event),
		closeCh:  make(chan struct{}),
	})
	return (len(m.endpoints) - 1)
}

func (m *endpointManager) getEndpoint(id int) (pipe.Endpoint, chan pipe.Event, chan struct{}, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if !(id >= 0 && id < len(m.endpoints)) {
		return nil, nil, nil, errors.New("Endpoint not found")
	}
	epw := m.endpoints[id]

	return epw.endpoint, epw.eventCh, epw.closeCh, nil
}

type endpointWrapper struct {
	endpoint pipe.Endpoint
	eventCh  chan pipe.Event
	closeCh  chan struct{}
}
