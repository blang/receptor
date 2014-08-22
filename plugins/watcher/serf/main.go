package main

import (
	"github.com/blang/receptor/plugin"
	"github.com/blang/receptor/plugins/watcher/serf/serfwatcher"
)

func main() {
	plugin.ServeWatcher(&serfwatcher.Watcher{})
}
