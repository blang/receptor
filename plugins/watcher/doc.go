/*
Package watcher is a container for watcher plugins.

How to register my own watcher?

1. Create a package with your name e.g. restapiserver inside the plugins/watcher package

2. Create a file called packagename.go e.g. restapiserver.go to register your package

You packagename.go should look like this:

	// +build watcher_restapiserver plugins_all

	package watcher

	import (
		"github.com/blang/receptor/discovery"
		"github.com/blang/receptor/plugins/watcher/restapiserver"
	)

	func init() {
		discovery.Watchers["restapiserver"] = &restapiserver.RestAPIServerWatcher{}
	}

*/
package watcher
