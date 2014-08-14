package restapiserver

import (
	"bytes"
	"encoding/json"
	"github.com/blang/receptor/pipe"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFunc(t *testing.T) {
	watcher := &RestAPIServerWatcher{}
	cfg := &Config{
		Listen: "127.0.0.1:9991",
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Could not marshal cfg: %s", string(b))
	}

	err = watcher.Setup(json.RawMessage(b))
	if err != nil {
		t.Fatalf("Watcher setup failed: %s", err)
	}

	serviceConfig := &ServiceConfig{
		Service: "testservice",
	}
	b, err = json.Marshal(serviceConfig)
	if err != nil {
		t.Fatalf("Could not marshal service cfg: %s", string(b))
	}

	handle, err := watcher.Accept(json.RawMessage(b))
	manHandle := pipe.NewManagedEndpoint(handle)
	if err != nil {
		t.Fatalf("Watcher accept failed: %s", err)
	}

	eventCh := make(chan pipe.Event, 1)

	watcher.IsRunning = true // Fake Running server
	testserver := httptest.NewServer(watcher.Router)
	defer testserver.Close()
	go manHandle.Handle(eventCh)

	restEvent := &RestEvent{
		Name: "testservice",
		Type: "nodeup",
		Host: "127.0.0.1",
		Port: 9991,
	}
	b, err = json.Marshal(restEvent)
	if err != nil {
		t.Fatalf("Could not marshal test restevent: %s", string(b))
	}
	br := bytes.NewReader(b)
	resp, err := http.Post(testserver.URL+"/service/testservice", "application/json", br)
	if err != nil {
		t.Fatalf("Error while Post: %s", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("RestAPI send wrong statuscode: %d", resp.StatusCode)
	}

	// Check if event was fired
	var recv pipe.Event
	select {
	case recv = <-eventCh:
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout: No event received")
	}

	// Test if event is correct
	if nlen := len(recv.Nodes()); nlen != 1 {
		t.Errorf("Event has %d nodes, expected 1", nlen)
	}
	if evName := recv.Nodes()[0].Name(); evName != restEvent.Name {
		t.Errorf("Event name was %s, expected %s", evName, restEvent.Name)
	}
	if evHost := recv.Nodes()[0].Host(); evHost != restEvent.Host {
		t.Errorf("Event host was %s, expected %s", evHost, restEvent.Host)
	}
	if evPort := recv.Nodes()[0].Port(); evPort != restEvent.Port {
		t.Errorf("Event port was %s, expected %s", evPort, restEvent.Port)
	}
	if evType := recv.Nodes()[0].Type(); evType != pipe.EventNodeUp {
		t.Errorf("Event type was %s, expected %s", evType, pipe.EventNodeUp)
	}

	// Check shutdown
	manHandle.Stop()
	err = manHandle.WaitTimeout(5 * time.Second)
	if err != nil {
		t.Errorf("Stop handle timeout: %s", err)
	}

}
