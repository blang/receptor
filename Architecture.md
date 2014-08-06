# Architecture and Design

## Questions

Why not send full backend updates instead of incremental updates?

> If we would send full backend updates, the reactors would receive these full updates, but every watcher has its own set of backends watched. This means that an update send by watcher `A` would overwrite the update send before by watcher `B` on the same channel. There is no way to handle this without identifying every watcher.

Why using this kind of plugin system with build tags?

> I found it to be the best option and it adds the power of golang instead of a scripting language like lua. You can import your own plugin package if your plugin.go imports from github etc which makes it possible to support third party plugins. Build tags make receptor as small or as big as the user wants and only imports plugins needed.