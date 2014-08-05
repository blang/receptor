package filelog

import (
	"encoding/json"
	"github.com/blang/receptor/pipeline"
	"github.com/blang/receptor/handler"
	"github.com/blang/receptor/reactor"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"
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

	handle, err := react.Accept(json.RawMessage(b))
	if err != nil {
		t.Fatalf("Does not accept config: %s\n", err)
	}
	manHandle := handler.NewManagedHandler(handle)

	eventCh := make(chan pipeline.Event)
	go manHandle.Handle(eventCh)

	eventCh <- &pipeline.SingleNode{
		EName: "Node1",
		EType: pipeline.EventNodeUp,
		EHost: "127.0.0.1",
		EPort: 8080,
	}

	eventCh <- &pipeline.SingleNode{
		EName: "Node2",
		EType: pipeline.EventNodeDown,
		EHost: "127.0.0.1",
		EPort: 8080,
	}

	close(eventCh)
	manHandle.Stop()
	err = manHandle.WaitTimeout(2 * time.Second)
	if err != nil {
		t.Fatal("Stop timed out")
	}

	data, err := ioutil.ReadFile(tmpFile.Name())
	lines := strings.Split(string(data), "\n")
	if countLines := len(lines); countLines != 3 {
		t.Errorf("Found %d lines instead of 2\n", countLines)
	}
	t.Logf("Logger output:\n%s", string(data))
}
