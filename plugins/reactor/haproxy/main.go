package main

import (
	"github.com/blang/receptor/plugin"
	"github.com/blang/receptor/plugins/reactor/haproxy/haproxy"
)

func main() {
	plugin.ServeReactor(&haproxy.HAProxyWatcher{})
}
