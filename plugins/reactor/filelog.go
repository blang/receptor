// +build reactor_filelog plugins_all

package reactor

import (
	"github.com/blang/receptor"
	"github.com/blang/receptor/plugins/reactor/filelog"
)

func init() {
	receptor.Reactors["filelog"] = &filelog.FileLogReactor{}
}
