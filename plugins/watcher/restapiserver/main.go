package main

import (
	"github.com/blang/receptor/plugin"
	"github.com/blang/receptor/plugins/watcher/restapiserver/restapi"
)

func main() {
	plugin.ServeWatcher(&restapi.RestAPIServerWatcher{})
}
