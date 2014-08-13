# Architecture and Design

## Questions

Why not send full backend updates instead of incremental updates?

> If we would send full backend updates, the reactors would receive these full updates, but every watcher has its own set of backends watched. This means that an update send by watcher `A` would overwrite the update send before by watcher `B` on the same channel. There is no way to handle this without identifying every watcher.

What was wrong with the old plugin system using build tags and integrated plugins?

> I found it to be the best option and it adds the power of golang instead of a scripting language like lua. You can import your own plugin package if your plugin.go imports from github etc which makes it possible to support third party plugins. Build tags make receptor as small or as big as the user wants and only imports plugins needed. The fact that every plugin has it's own dependencies and might need special versions of them was one point. Also some plugins might need cgo features (or a special build process in general) which would result in changing the whole build process.