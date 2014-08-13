package main

import (
	"github.com/blang/receptor/plugin"
	"github.com/blang/receptor/plugins/watcher/dummy/dummy"
)

func main() {
	plugin.ServeWatcher(&dummy.DummyWatcher{})
}
