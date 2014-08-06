// Container to add build dependencies to watcher and reactor plugins. See subpackages for documentation.
package plugins

import (
	_ "github.com/blang/receptor/plugins/reactor"
	_ "github.com/blang/receptor/plugins/watcher"
)
