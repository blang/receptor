/*
Package reactor is a container for reactor plugins.

How to register my own reactor?

1. Create a package with your name e.g. filelog inside the plugins/reactor package

2. Create a file called packagename.go e.g. filelog.go to register your package

You packagename.go should look like this:

	// +build reactor_filelog plugins_all

	package reactor

	import (
		"github.com/blang/receptor/plugins/reactor/filelog"
		"github.com/blang/receptor/reactor"
	)

	func init() {
		reactor.Reactors["filelog"] = &filelog.FileLogReactor{}
	}
*/
package reactor
