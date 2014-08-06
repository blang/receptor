// +build watcher_restapiserver plugins_all

package watcher

import (
	"github.com/blang/receptor/discovery"
	"github.com/blang/receptor/plugins/watcher/restapiserver"
)

func init() {
	discovery.Watchers["restapiserver"] = &restapiserver.RestAPIServerWatcher{}
}
