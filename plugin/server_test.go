package plugin

import (
	"testing"
)

func TestExtrSocketInfo(t *testing.T) {
	args := []string{"receptord", "unix", "/tmp/test.sock"}
	_, _, err := extrSocketInfo(args[:1])
	if err == nil {
		t.Fatal("Expected error")
	}
	net, laddr, err := extrSocketInfo(args)
	if err != nil {
		t.Fatal("No error expected")
	}
	if net != "unix" || laddr != "/tmp/test.sock" {
		t.Fatal("Wrong parsed arguments")
	}

}
