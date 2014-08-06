// +build watcher_serf plugins_all

package watcher

import (
	"github.com/blang/receptor"
	"github.com/blang/receptor/plugins/watcher/serf"
)

func init() {
	receptor.Watchers["serf"] = &serf.Watcher{}
}
