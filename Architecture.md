# Architecture and Design

## Questions

Why not send full backend updates instead of incremental updates?

> If we would send full backend updates, the reactors would receive these full updates, but every watcher has its own set of backends watched. This means that an update send by watcher `A` would overwrite the update send before by watcher `B` on the same channel. There is no way to handle this without identifying every watcher.