package filelog

import (
	"encoding/json"
	"github.com/blang/receptor/pipe"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"
)

func TestFunc(t *testing.T) {
	filelog := &FileLogReactor{}
	react := pipe.Reactor(filelog) // Check if conform interface

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

	handle, err := react.Accept(json.RawMessage(b))
	if err != nil {
		t.Fatalf("Does not accept config: %s\n", err)
	}
	manHandle := pipe.NewManagedEndpoint(handle)

	eventCh := make(chan pipe.Event)
	go manHandle.Handle(eventCh)

	eventCh <- pipe.NewEventWithNode("Node1", pipe.NodeUp, "127.0.0.1", 80)

	eventCh <- pipe.NewEventWithNode("Node2", pipe.NodeDown, "127.0.0.1", 80)

	close(eventCh)
	manHandle.Stop()
	err = manHandle.WaitTimeout(2 * time.Second)
	if err != nil {
		t.Fatal("Stop timed out")
	}

	data, err := ioutil.ReadFile(tmpFile.Name())
	lines := strings.Split(string(data), "\n")
	if countLines := len(lines); countLines != 3 {
		t.Errorf("Found %d lines instead of 3: %q\n", countLines, string(data))
	}
	t.Logf("Logger output:\n%s", string(data))
}
