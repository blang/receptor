// +build reactor_filelog plugins_all

package reactor

import (
	"github.com/blang/receptor/plugins/reactor/filelog"
	"github.com/blang/receptor/reactor"
)

func init() {
	reactor.Reactors["filelog"] = &filelog.FileLogReactor{}
}
