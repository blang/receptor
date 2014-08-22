// +build integration_test

package serfwatcher

import (
	"encoding/json"
	serfc "github.com/hashicorp/serf/client"
	"testing"
)

func TestSerf(t *testing.T) {
	serfConfig := &serfc.Config{
		Addr:    "127.0.0.1:7373",
		AuthKey: "",
	}

	client, err := serfc.ClientFromConfig(serfConfig)
	if err != nil {
		t.Fatalf("Connect error: %s", err)
	}
	members, err := client.Members()
	if err != nil {
		t.Fatalf("Members: %s", err)
	}
	b, err := json.MarshalIndent(members, "", "  ")
	if err != nil {
		t.Fatalf("Members marshal %s", err)
	}
	t.Logf("output: %s", string(b))
}
