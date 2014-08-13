package main

import (
	"github.com/blang/receptor/plugin"
	"github.com/blang/receptor/plugins/reactor/filelogger/filelog"
)

func main() {
	plugin.ServeReactor(&filelog.FileLogReactor{})
}
