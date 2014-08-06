// +build watcher_restapiserver plugins_all

package watcher

import (
	"github.com/blang/receptor"
	"github.com/blang/receptor/plugins/watcher/restapiserver"
)

func init() {
	receptor.Watchers["restapiserver"] = &restapiserver.RestAPIServerWatcher{}
}
