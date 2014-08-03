package filelog

import (
	"encoding/json"
	"github.com/blang/receptor/event"
	"github.com/blang/receptor/reactor"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestFunc(t *testing.T) {
	filelog := &FileLogReactor{}
	react := reactor.Reactor(filelog) // Check if conform interface

	err := react.Setup(nil)
	if err != nil {
		t.Fatal("Does not accept empty config")
	}

	tmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("Could not create tmp file: %s", err)
	}
	defer os.Remove(tmpFile.Name())

	cfg := ServiceConfig{
		Filename: tmpFile.Name(),
	}

	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Test internal failed: %s\n", err)
	}

	handler, err := react.Accept(json.RawMessage(b))
	if err != nil {
		t.Fatalf("Does not accept config: %s\n", err)
	}

	eventCh := make(chan event.Event)
	doneCh := make(chan struct{})
	go handler.Handle(eventCh, doneCh)

	eventCh <- &event.SingleNode{
		EName: "Node1",
		EType: event.EventNodeUp,
		EHost: "127.0.0.1",
		EPort: 8080,
	}

	eventCh <- &event.SingleNode{
		EName: "Node2",
		EType: event.EventNodeDown,
		EHost: "127.0.0.1",
		EPort: 8080,
	}

	close(eventCh)

	<-doneCh
	data, err := ioutil.ReadFile(tmpFile.Name())
	lines := strings.Split(string(data), "\n")
	if countLines := len(lines); countLines != 3 {
		t.Errorf("Found %d lines instead of 2\n", countLines)
	}
	t.Logf("Logger output:\n%s", string(data))
}
